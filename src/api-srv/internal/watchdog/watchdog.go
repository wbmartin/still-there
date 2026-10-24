// Package watchdog raises, reminds about and resolves alarms, and sends the
// texts they cause. Each Tick:
//
//  1. Per device: device_offline when its check-ins stopped; per band,
//     beacon_absent when the Pi reports it gone. A condition that cleared
//     resolves its open alarm ("reactivated").
//  2. Per open alarm: a reminder every tenant remind interval while
//     raised; once acknowledged, one daily reminder at the tenant's time;
//     turned off, nothing until it resolves.
//  3. Owed invite texts.
//  4. The alert_event outbox: each unsent event to every active contact of
//     the device who wants that event.
//
// Every state change is a row the store claims (open alert unique index,
// last_notified_ts / last_daily_date compare-and-set), so a tick that
// overlaps another cannot double a text.
package watchdog

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/sms"
)

// Store is the slice of *repository.WatchdogRepository the watchdog uses.
type Store interface {
	WatchdogDeviceList(ctx context.Context) ([]*repository.WatchDevice, error)
	WatchdogOpenAlertList(ctx context.Context) ([]*repository.WatchAlert, error)
	WatchdogRaiseAlert(ctx context.Context, securityContext int, deviceID int, beaconID *int, kind string, now float64) (bool, error)
	WatchdogQueueReminder(ctx context.Context, a *repository.WatchAlert, eventType string, localDate string, now float64) (bool, error)
	WatchdogResolveAlert(ctx context.Context, a *repository.WatchAlert, now float64) (bool, error)
	WatchdogUnsentEventList(ctx context.Context, limit int) ([]*repository.WatchEvent, error)
	WatchdogMarkEventSent(ctx context.Context, alertEventID int, securityContext int, now float64) error
	WatchdogDeviceContactList(ctx context.Context, securityContext int, deviceID int) ([]*repository.WatchContact, error)
	WatchdogPendingInviteList(ctx context.Context) ([]*repository.WatchContact, error)
	WatchdogMarkInviteSent(ctx context.Context, contactID int, securityContext int, now float64) error
	InsertSMS(ctx context.Context, m repository.SMSRecord, now float64) error
}

// Logger is the part of the app logger the watchdog uses
// (*logging.Logger satisfies it).
type Logger interface {
	Info(code string, message string, keyvals ...any)
	Error(code string, message string, keyvals ...any)
}

const (
	KindDeviceOffline = "device_offline"
	KindBeaconAbsent  = "beacon_absent"

	eventBatchSize = 100
)

type Watchdog struct {
	Store  Store
	Sender sms.Sender
	Log    Logger
	// Now is the clock (tests set it); nil is the wall clock.
	Now func() float64
	// StartTS is when this process started watching. A device's check-in
	// gap is counted from no earlier than this, so a server outage does
	// not raise device_offline for every home at restart. Run sets it.
	StartTS float64
}

func (w *Watchdog) now() float64 {
	if w.Now != nil {
		return w.Now()
	}
	return model.Now()
}

// Run ticks at once and then every interval until ctx is done.
func (w *Watchdog) Run(ctx context.Context, interval time.Duration) {
	if w.StartTS == 0 {
		w.StartTS = w.now()
	}
	w.Tick(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Tick(ctx)
		}
	}
}

// Tick is one pass. A failed step is logged and the next tick retries.
func (w *Watchdog) Tick(ctx context.Context) {
	if err := w.evaluate(ctx); err != nil {
		w.Log.Error("WATCHDOG_EVALUATE", "evaluating alarms", "err", err)
	}
	if err := w.sendInvites(ctx); err != nil {
		w.Log.Error("WATCHDOG_INVITE", "sending invites", "err", err)
	}
	if err := w.sendOutbox(ctx); err != nil {
		w.Log.Error("WATCHDOG_OUTBOX", "sending alert texts", "err", err)
	}
}

type alertKey struct {
	deviceID int
	beaconID int // 0 for device_offline
	kind     string
}

func keyOf(a *repository.WatchAlert) alertKey {
	k := alertKey{deviceID: a.DeviceID, kind: a.AlertKind}
	if a.BeaconID != nil {
		k.beaconID = *a.BeaconID
	}
	return k
}

func (w *Watchdog) evaluate(ctx context.Context) error {
	now := w.now()
	deviceList, err := w.Store.WatchdogDeviceList(ctx)
	if err != nil {
		return err
	}
	alertList, err := w.Store.WatchdogOpenAlertList(ctx)
	if err != nil {
		return err
	}
	openMap := make(map[alertKey]*repository.WatchAlert, len(alertList))
	for _, a := range alertList {
		openMap[keyOf(a)] = a
	}

	for _, d := range deviceList {
		offline, known := deviceOffline(d, now, w.StartTS)
		if !known {
			continue // never checked in: nothing to compare against yet
		}
		w.apply(ctx, d, nil, KindDeviceOffline, offline, openMap, now)
		if offline {
			continue // band reports are stale while the Pi is silent
		}
		for i := range d.BeaconList {
			b := &d.BeaconList[i]
			if b.Present == nil {
				continue
			}
			w.apply(ctx, d, &b.BeaconID, KindBeaconAbsent, !*b.Present, openMap, now)
		}
	}

	for _, d := range deviceList {
		for _, a := range alertList {
			if a.DeviceID == d.DeviceID && a.SecurityContextID == d.SecurityContextID {
				w.remind(ctx, a, d.Tenant, now)
			}
		}
	}
	return nil
}

// deviceOffline reports whether d has gone more than its alarm threshold
// without a check-in; known is false for a device that never checked in.
func deviceOffline(d *repository.WatchDevice, now float64, startTS float64) (offline bool, known bool) {
	if d.LastCheckinTS == nil {
		return false, false
	}
	last := max(*d.LastCheckinTS, startTS)
	limit := float64(d.CheckinIntervalSec * d.MissedCheckinBeforeAlarm)
	return now-last > limit, true
}

// apply raises an alarm for a condition that holds with none open, and
// resolves the open one when it no longer holds.
func (w *Watchdog) apply(
	ctx context.Context, d *repository.WatchDevice, beaconID *int, kind string,
	holds bool, openMap map[alertKey]*repository.WatchAlert, now float64,
) {
	key := alertKey{deviceID: d.DeviceID, kind: kind}
	if beaconID != nil {
		key.beaconID = *beaconID
	}
	open := openMap[key]
	switch {
	case holds && open == nil:
		raised, err := w.Store.WatchdogRaiseAlert(ctx, d.SecurityContextID, d.DeviceID, beaconID, kind, now)
		if err != nil {
			w.Log.Error("WATCHDOG_RAISE", "raising alarm", "deviceId", d.DeviceID, "kind", kind, "err", err)
		} else if raised {
			w.Log.Info("WATCHDOG_RAISE", "alarm raised", "deviceId", d.DeviceID, "kind", kind)
		}
	case !holds && open != nil:
		resolved, err := w.Store.WatchdogResolveAlert(ctx, open, now)
		if err != nil {
			w.Log.Error("WATCHDOG_RESOLVE", "resolving alarm", "alertId", open.AlertID, "err", err)
		} else if resolved {
			w.Log.Info("WATCHDOG_RESOLVE", "alarm resolved", "alertId", open.AlertID)
		}
		delete(openMap, key)
	}
}

// remind queues the reminder an open alarm is due, if any.
func (w *Watchdog) remind(ctx context.Context, a *repository.WatchAlert, tenant repository.WatchTenant, now float64) {
	eventType, localDate := reminderDue(a, tenant, now)
	if eventType == "" {
		return
	}
	if _, err := w.Store.WatchdogQueueReminder(ctx, a, eventType, localDate, now); err != nil {
		w.Log.Error("WATCHDOG_REMIND", "queueing reminder", "alertId", a.AlertID, "err", err)
	}
}

// reminderDue says which reminder an open alarm owes now: "reminder" while
// raised and the remind interval has passed since the last text;
// "daily_reminder" once acknowledged, when the tenant's local daily time
// has passed today, it was acknowledged before that time, and today's has
// not gone. localDate is the tenant's date for the daily claim.
func reminderDue(a *repository.WatchAlert, tenant repository.WatchTenant, now float64) (eventType string, localDate string) {
	switch a.AlertState {
	case "raised":
		last := a.RaisedTS
		if a.LastNotifiedTS != nil {
			last = *a.LastNotifiedTS
		}
		if now-last >= float64(tenant.RemindIntervalMin*60) {
			return "reminder", ""
		}
	case "acknowledged":
		loc, err := time.LoadLocation(tenant.TimeZone)
		if err != nil {
			loc = time.UTC
		}
		local := epochTime(now).In(loc)
		today := local.Format(time.DateOnly)
		dailyAt, err := time.ParseInLocation(time.DateOnly+" 15:04", today+" "+tenant.DailyReminderTime, loc)
		if err != nil {
			return "", ""
		}
		ackTS := a.RaisedTS
		if a.AckTS != nil {
			ackTS = *a.AckTS
		}
		alreadyToday := a.LastDailyDate != nil && *a.LastDailyDate == today
		if !local.Before(dailyAt) && ackTS < model.EpochSeconds(dailyAt) && !alreadyToday {
			return "daily_reminder", today
		}
	}
	return "", ""
}

func (w *Watchdog) sendInvites(ctx context.Context) error {
	inviteList, err := w.Store.WatchdogPendingInviteList(ctx)
	if err != nil {
		return err
	}
	for _, c := range inviteList {
		body := InviteText(c.DeviceName)
		if !w.send(ctx, c, nil, body) {
			continue // the next tick tries again
		}
		if err := w.Store.WatchdogMarkInviteSent(ctx, c.ContactID, c.SecurityContextID, w.now()); err != nil {
			w.Log.Error("WATCHDOG_INVITE", "marking invite sent", "contactId", c.ContactID, "err", err)
		}
	}
	return nil
}

// sendOutbox texts each unsent event to the device's active contacts who
// want it, then marks it sent. A failed text is logged in sms_message and
// not retried: retrying the event would repeat it to everyone it reached.
func (w *Watchdog) sendOutbox(ctx context.Context) error {
	eventList, err := w.Store.WatchdogUnsentEventList(ctx, eventBatchSize)
	if err != nil {
		return err
	}
	for _, e := range eventList {
		contactList, err := w.Store.WatchdogDeviceContactList(ctx, e.SecurityContextID, e.DeviceID)
		if err != nil {
			w.Log.Error("WATCHDOG_OUTBOX", "listing contacts", "alertEventId", e.AlertEventID, "err", err)
			continue
		}
		body := EventText(e)
		eventID := e.AlertEventID
		for _, c := range contactList {
			if c.ContactStatus != "active" || !slices.Contains(c.NotifyEventList, e.EventType) {
				continue
			}
			w.send(ctx, c, &eventID, body)
		}
		if err := w.Store.WatchdogMarkEventSent(ctx, e.AlertEventID, e.SecurityContextID, w.now()); err != nil {
			w.Log.Error("WATCHDOG_OUTBOX", "marking event sent", "alertEventId", e.AlertEventID, "err", err)
		}
	}
	return nil
}

// send texts one contact and records it; ok is false when the provider
// refused it.
func (w *Watchdog) send(ctx context.Context, c *repository.WatchContact, alertEventID *int, body string) bool {
	providerID, status, err := w.Sender.Send(ctx, c.PhoneE164, body)
	contactID := c.ContactID
	rec := repository.SMSRecord{
		SecurityContextID: c.SecurityContextID,
		Direction:         "out",
		PhoneE164:         c.PhoneE164,
		Body:              body,
		ContactID:         &contactID,
		AlertEventID:      alertEventID,
		Status:            status,
	}
	if providerID != "" {
		rec.ProviderMessageID = &providerID
	}
	if err != nil {
		rec.Status = "failed"
		msg := err.Error()
		rec.ErrorText = &msg
		w.Log.Error("WATCHDOG_SEND", "text failed", "contactId", c.ContactID, "to", sms.MaskPhone(c.PhoneE164), "err", err)
	}
	if recErr := w.Store.InsertSMS(ctx, rec, w.now()); recErr != nil {
		w.Log.Error("WATCHDOG_SEND", "recording text", "contactId", c.ContactID, "err", recErr)
	}
	return err == nil
}

func epochTime(ts float64) time.Time {
	sec := int64(ts)
	return time.Unix(sec, int64((ts-float64(sec))*1e9))
}

// --- texts -------------------------------------------------------------------

const brand = "Still There"

func InviteText(deviceName string) string {
	return fmt.Sprintf("%s: you were added to alerts for %s. Reply Y to join, STOP to opt out, HELP for help. Msg & data rates may apply.", brand, deviceName)
}

func HelpText() string {
	return brand + " sends home-monitoring alerts. Reply OK to acknowledge an alert, STOP to opt out."
}

func JoinedText(deviceNameList []string) string {
	return fmt.Sprintf("%s: you will get alerts for %s. Reply STOP to opt out.", brand, strings.Join(deviceNameList, ", "))
}

func AckReplyText(count int) string {
	if count == 0 {
		return brand + ": no open alert to acknowledge."
	}
	return fmt.Sprintf("%s: acknowledged %d alert(s). Everyone on the list has been told.", brand, count)
}

func UnknownReplyText() string {
	return brand + ": reply OK to acknowledge an alert, HELP for help, STOP to opt out."
}

// EventText is the text one outbox event sends.
func EventText(e *repository.WatchEvent) string {
	loc, err := time.LoadLocation(e.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	clock := func(ts float64) string { return epochTime(ts).In(loc).Format("3:04 PM Jan 2") }
	what := "the base station at " + e.DeviceName
	subject := what + " has not checked in"
	since := e.RaisedTS
	if e.DeviceLastCheckin != nil {
		since = *e.DeviceLastCheckin
	}
	if e.AlertKind == KindBeaconAbsent {
		label := "the band"
		if e.BeaconLabel != nil {
			label = *e.BeaconLabel
		}
		what = label + " at " + e.DeviceName
		subject = label + " has not been detected at " + e.DeviceName
		since = e.RaisedTS
		if e.BeaconLastSeenTS != nil {
			since = *e.BeaconLastSeenTS
		}
	}
	actor := "someone"
	if e.ActorName != nil {
		actor = *e.ActorName
	}

	switch e.EventType {
	case "raised":
		return fmt.Sprintf("%s ALERT: %s since %s. Reply OK to acknowledge.", brand, subject, clock(since))
	case "reminder":
		return fmt.Sprintf("%s REMINDER: %s since %s. Reply OK to acknowledge.", brand, subject, clock(since))
	case "daily_reminder":
		return fmt.Sprintf("%s daily reminder: %s since %s%s.", brand, subject, clock(since), byClause(e))
	case "acknowledged":
		return fmt.Sprintf("%s: %s acknowledged the alert for %s.", brand, actor, what)
	case "turned_off":
		return fmt.Sprintf("%s: %s turned off the alert for %s. Alerting resumes when it is back.", brand, actor, what)
	case "reactivated":
		back := what + " is detected again"
		if e.AlertKind == KindDeviceOffline {
			back = what + " is checking in again"
		}
		return fmt.Sprintf("%s: %s; alerting is re-activated%s.", brand, back, byClause(e))
	}
	return fmt.Sprintf("%s: update on %s.", brand, what)
}

// byClause names who acknowledged or turned the alert off, for the daily
// reminder and the reactivated text.
func byClause(e *repository.WatchEvent) string {
	switch {
	case e.OffBy != nil:
		return " (turned off by " + *e.OffBy + ")"
	case e.AckBy != nil:
		return " (acknowledged by " + *e.AckBy + ")"
	}
	return ""
}
