package watchdog

import (
	"context"
	"strings"
	"testing"
	"time"

	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// fakeStore keeps alerts, events and texts in memory with the same claim
// rules as the SQL: one open alert per key, compare-and-set reminders.
type fakeStore struct {
	deviceList  []*repository.WatchDevice
	alertList   []*repository.WatchAlert
	eventList   []*repository.WatchEvent
	contactList []*repository.WatchContact
	smsList     []repository.SMSRecord
	nextID      int
}

func (f *fakeStore) id() int { f.nextID++; return f.nextID }

func (f *fakeStore) WatchdogDeviceList(context.Context) ([]*repository.WatchDevice, error) {
	return f.deviceList, nil
}

func (f *fakeStore) WatchdogOpenAlertList(context.Context) ([]*repository.WatchAlert, error) {
	var open []*repository.WatchAlert
	for _, a := range f.alertList {
		if a.AlertState != "resolved" {
			c := *a
			open = append(open, &c)
		}
	}
	return open, nil
}

func (f *fakeStore) find(alertID int) *repository.WatchAlert {
	for _, a := range f.alertList {
		if a.AlertID == alertID {
			return a
		}
	}
	return nil
}

func (f *fakeStore) addEvent(a *repository.WatchAlert, eventType string, actor *string, now float64) {
	d := f.deviceList[0]
	e := &repository.WatchEvent{
		AlertEventID: f.id(), SecurityContextID: a.SecurityContextID, AlertID: a.AlertID,
		EventType: eventType, ActorName: actor, CreateTS: now, AlertKind: a.AlertKind,
		RaisedTS: a.RaisedTS, AckBy: a.AckBy, OffBy: a.OffBy, DeviceID: a.DeviceID,
		DeviceName: d.DeviceName, TimeZone: d.Tenant.TimeZone,
	}
	if a.BeaconID != nil {
		label := "B7"
		e.BeaconLabel = &label
	}
	f.eventList = append(f.eventList, e)
}

func (f *fakeStore) WatchdogRaiseAlert(_ context.Context, sc int, deviceID int, beaconID *int, kind string, now float64) (bool, error) {
	for _, a := range f.alertList {
		if a.AlertState != "resolved" && a.DeviceID == deviceID && a.AlertKind == kind &&
			((a.BeaconID == nil && beaconID == nil) || (a.BeaconID != nil && beaconID != nil && *a.BeaconID == *beaconID)) {
			return false, nil
		}
	}
	n := now
	a := &repository.WatchAlert{AlertID: f.id(), SecurityContextID: sc, DeviceID: deviceID,
		BeaconID: beaconID, AlertKind: kind, AlertState: "raised", RaisedTS: now, LastNotifiedTS: &n}
	f.alertList = append(f.alertList, a)
	f.addEvent(a, "raised", nil, now)
	return true, nil
}

func (f *fakeStore) WatchdogQueueReminder(_ context.Context, held *repository.WatchAlert, eventType string, localDate string, now float64) (bool, error) {
	a := f.find(held.AlertID)
	if a == nil || a.AlertState == "resolved" {
		return false, nil
	}
	if eventType == "daily_reminder" {
		if !eqStr(a.LastDailyDate, held.LastDailyDate) {
			return false, nil
		}
		d := localDate
		a.LastDailyDate = &d
	} else {
		if !eqFloat(a.LastNotifiedTS, held.LastNotifiedTS) {
			return false, nil
		}
		n := now
		a.LastNotifiedTS = &n
	}
	f.addEvent(a, eventType, nil, now)
	return true, nil
}

func (f *fakeStore) WatchdogResolveAlert(_ context.Context, held *repository.WatchAlert, now float64) (bool, error) {
	a := f.find(held.AlertID)
	if a == nil || a.AlertState == "resolved" {
		return false, nil
	}
	a.AlertState = "resolved"
	f.addEvent(a, "reactivated", nil, now)
	return true, nil
}

func (f *fakeStore) WatchdogUnsentEventList(context.Context, int) ([]*repository.WatchEvent, error) {
	var unsent []*repository.WatchEvent
	for _, e := range f.eventList {
		if e.CreateTS >= 0 {
			unsent = append(unsent, e)
		}
	}
	return unsent, nil
}

func (f *fakeStore) WatchdogMarkEventSent(_ context.Context, id int, _ int, _ float64) error {
	for _, e := range f.eventList {
		if e.AlertEventID == id {
			e.CreateTS = -1 // sent
		}
	}
	return nil
}

func (f *fakeStore) WatchdogDeviceContactList(context.Context, int, int) ([]*repository.WatchContact, error) {
	return f.contactList, nil
}

func (f *fakeStore) WatchdogPendingInviteList(context.Context) ([]*repository.WatchContact, error) {
	return nil, nil
}

func (f *fakeStore) WatchdogMarkInviteSent(context.Context, int, int, float64) error { return nil }

func (f *fakeStore) InsertSMS(_ context.Context, m repository.SMSRecord, _ float64) error {
	f.smsList = append(f.smsList, m)
	return nil
}

func eqStr(a, b *string) bool    { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func eqFloat(a, b *float64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

type fakeSender struct{ sent []string }

func (s *fakeSender) Send(_ context.Context, to string, body string) (string, string, error) {
	s.sent = append(s.sent, to+": "+body)
	return "", "logged", nil
}

type nopLog struct{}

func (nopLog) Info(string, string, ...any)  {}
func (nopLog) Error(string, string, ...any) {}

func ptrBool(b bool) *bool        { return &b }
func ptrFloat(f float64) *float64 { return &f }

var allEvents = []string{"raised", "reminder", "acknowledged", "turned_off", "reactivated", "daily_reminder"}

// t0 is 2026-10-24 08:00 in New York.
var t0 = model.EpochSeconds(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))

func setup() (*fakeStore, *fakeSender, *Watchdog, *float64) {
	clock := t0
	store := &fakeStore{
		deviceList: []*repository.WatchDevice{{
			DeviceID: 1, SecurityContextID: 2, DeviceName: "Mom's house",
			CheckinIntervalSec: 60, MissedCheckinBeforeAlarm: 3,
			LastCheckinTS: ptrFloat(t0),
			Tenant:        repository.WatchTenant{TimeZone: "America/New_York", RemindIntervalMin: 15, DailyReminderTime: "09:00"},
			BeaconList:    []repository.WatchBeacon{{BeaconID: 7, Label: "B7", Present: ptrBool(true), LastReportTS: ptrFloat(t0)}},
		}},
		contactList: []*repository.WatchContact{
			{ContactID: 1, SecurityContextID: 2, DeviceID: 1, ContactName: "Ann", PhoneE164: "+15550000001", ContactStatus: "active", NotifyEventList: allEvents},
			{ContactID: 2, SecurityContextID: 2, DeviceID: 1, ContactName: "Bob", PhoneE164: "+15550000002", ContactStatus: "invited", NotifyEventList: allEvents},
			{ContactID: 3, SecurityContextID: 2, DeviceID: 1, ContactName: "Cy", PhoneE164: "+15550000003", ContactStatus: "active", NotifyEventList: []string{"raised"}},
		},
	}
	sender := &fakeSender{}
	w := &Watchdog{Store: store, Sender: sender, Log: nopLog{}, Now: func() float64 { return clock }, StartTS: t0}
	return store, sender, w, &clock
}

func TestBeaconAbsentRaisesOnceAndTextsActiveContacts(t *testing.T) {
	store, sender, w, _ := setup()
	store.deviceList[0].BeaconList[0].Present = ptrBool(false)
	w.Tick(context.Background())
	w.Tick(context.Background())
	if len(store.alertList) != 1 {
		t.Fatalf("want 1 alert, got %d", len(store.alertList))
	}
	if len(sender.sent) != 2 { // Ann and Cy; Bob is only invited
		t.Fatalf("want 2 texts, got %v", sender.sent)
	}
	if !strings.Contains(sender.sent[0], "ALERT: B7 has not been detected at Mom's house") {
		t.Fatalf("raised text: %q", sender.sent[0])
	}
}

func TestReminderEveryIntervalUntilAcknowledged(t *testing.T) {
	store, sender, w, clock := setup()
	store.deviceList[0].BeaconList[0].Present = ptrBool(false)
	w.Tick(context.Background())
	*clock += 14 * 60
	store.deviceList[0].LastCheckinTS = ptrFloat(*clock)
	w.Tick(context.Background())
	if len(sender.sent) != 2 {
		t.Fatalf("14 min: want no reminder yet, got %v", sender.sent)
	}
	*clock += 60
	store.deviceList[0].LastCheckinTS = ptrFloat(*clock)
	w.Tick(context.Background())
	if len(sender.sent) != 3 || !strings.Contains(sender.sent[2], "REMINDER") {
		t.Fatalf("15 min: want a reminder to Ann only (Cy wants raised only), got %v", sender.sent)
	}
}

func TestAcknowledgedGetsDailyReminderAtTenantTime(t *testing.T) {
	store, sender, w, clock := setup()
	store.deviceList[0].BeaconList[0].Present = ptrBool(false)
	w.Tick(context.Background())
	a := store.alertList[0]
	a.AlertState = "acknowledged"
	ann := "Ann"
	a.AckBy, a.AckTS = &ann, ptrFloat(t0+60)
	before := len(sender.sent)

	*clock = t0 + 59*60 // 08:59 local
	store.deviceList[0].LastCheckinTS = ptrFloat(*clock)
	w.Tick(context.Background())
	if len(sender.sent) != before {
		t.Fatalf("acknowledged, before 09:00: want nothing, got %v", sender.sent[before:])
	}
	*clock = t0 + 61*60 // 09:01 local
	store.deviceList[0].LastCheckinTS = ptrFloat(*clock)
	w.Tick(context.Background())
	w.Tick(context.Background())
	got := sender.sent[before:]
	if len(got) != 1 || !strings.Contains(got[0], "daily reminder") || !strings.Contains(got[0], "acknowledged by Ann") {
		t.Fatalf("09:01: want one daily reminder naming Ann, got %v", got)
	}
}

func TestAcknowledgedAfterDailyTimeWaitsForTomorrow(t *testing.T) {
	a := &repository.WatchAlert{AlertState: "acknowledged", RaisedTS: t0, AckTS: ptrFloat(t0 + 2*3600)} // acked 10:00
	tenant := repository.WatchTenant{TimeZone: "America/New_York", DailyReminderTime: "09:00", RemindIntervalMin: 15}
	if ev, _ := reminderDue(a, tenant, t0+3*3600); ev != "" {
		t.Fatalf("acked at 10:00: no daily reminder at 11:00 the same day, got %q", ev)
	}
	if ev, date := reminderDue(a, tenant, t0+25*3600); ev != "daily_reminder" || date != "2026-10-25" {
		t.Fatalf("next day 09:00: want daily_reminder 2026-10-25, got %q %q", ev, date)
	}
}

func TestTurnedOffSendsNothingUntilResolved(t *testing.T) {
	store, sender, w, clock := setup()
	store.deviceList[0].BeaconList[0].Present = ptrBool(false)
	w.Tick(context.Background())
	a := store.alertList[0]
	a.AlertState = "turned_off"
	ann := "Ann"
	a.OffBy = &ann
	before := len(sender.sent)
	*clock += 26 * 3600
	store.deviceList[0].LastCheckinTS = ptrFloat(*clock)
	w.Tick(context.Background())
	if len(sender.sent) != before {
		t.Fatalf("turned off: want no texts, got %v", sender.sent[before:])
	}
	store.deviceList[0].BeaconList[0].Present = ptrBool(true)
	w.Tick(context.Background())
	got := sender.sent[before:]
	if len(got) != 1 || !strings.Contains(got[0], "B7 at Mom's house is detected again; alerting is re-activated (turned off by Ann)") {
		t.Fatalf("band back: want one reactivated text to Ann, got %v", got)
	}
}

func TestDeviceOfflineAndRestartGrace(t *testing.T) {
	store, sender, w, clock := setup()
	store.deviceList[0].LastCheckinTS = ptrFloat(t0 - 3600) // an hour before the server started
	*clock = t0 + 179
	w.Tick(context.Background())
	if len(store.alertList) != 0 {
		t.Fatal("within 3 intervals of start: restart grace should hold the alarm")
	}
	store.deviceList[0].BeaconList[0].Present = ptrBool(false) // stale while offline
	*clock = t0 + 181
	w.Tick(context.Background())
	if len(store.alertList) != 1 || store.alertList[0].AlertKind != KindDeviceOffline {
		t.Fatalf("want one device_offline alert and no beacon alert, got %+v", store.alertList)
	}
	if !strings.Contains(sender.sent[0], "base station at Mom's house has not checked in") {
		t.Fatalf("offline text: %q", sender.sent[0])
	}
}

func TestNeverCheckedInIsIgnored(t *testing.T) {
	store, _, w, clock := setup()
	store.deviceList[0].LastCheckinTS = nil
	*clock += 86400
	w.Tick(context.Background())
	if len(store.alertList) != 0 {
		t.Fatal("a device that never checked in raised an alarm")
	}
}
