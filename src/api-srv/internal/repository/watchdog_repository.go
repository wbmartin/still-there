package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WatchdogRepository is the watchdog's and the inbound-text handler's view
// of the monitoring tables (migration 0008). Like the issue relay's Relay*
// methods it is a system job serving every tenant: its listings cross
// tenants by design, and every write names the row's own tenant beside
// the id. It writes as user 0. Runtime writes (notify counters, daily
// date, contact status, invite sent) leave update_ts alone; state changes
// an admin's page holds a lock on (resolve, acknowledge) bump it.
type WatchdogRepository struct {
	db *pgxpool.Pool
}

func NewWatchdogRepository(db *pgxpool.Pool) *WatchdogRepository {
	return &WatchdogRepository{db: db}
}

const watchdogUserID = 0

// Reminder settings for a tenant with no tenant_setting row yet.
const (
	DefaultTimeZone          = "America/New_York"
	DefaultRemindIntervalMin = 15
	DefaultDailyReminderTime = "09:00"
)

type WatchTenant struct {
	TimeZone          string
	RemindIntervalMin int
	DailyReminderTime string
}

type WatchBeacon struct {
	BeaconID     int
	Label        string
	Present      *bool
	LastReportTS *float64
}

type WatchDevice struct {
	DeviceID                 int
	SecurityContextID        int
	DeviceName               string
	CheckinIntervalSec       int
	MissedCheckinBeforeAlarm int
	LastCheckinTS            *float64
	Tenant                   WatchTenant
	BeaconList               []WatchBeacon
}

type WatchAlert struct {
	AlertID           int
	SecurityContextID int
	DeviceID          int
	BeaconID          *int
	AlertKind         string
	AlertState        string
	RaisedTS          float64
	LastNotifiedTS    *float64
	AckTS             *float64
	AckBy             *string
	OffBy             *string
	LastDailyDate     *string
}

// WatchEvent is one unsent outbox row with what its text needs.
type WatchEvent struct {
	AlertEventID      int
	SecurityContextID int
	AlertID           int
	EventType         string
	ActorName         *string
	ActorVia          *string
	CreateTS          float64
	AlertKind         string
	RaisedTS          float64
	AckBy             *string
	OffBy             *string
	DeviceID          int
	DeviceName        string
	DeviceLastCheckin *float64
	BeaconLabel       *string
	BeaconLastSeenTS  *float64
	TimeZone          string
}

type WatchContact struct {
	ContactID         int
	SecurityContextID int
	DeviceID          int
	DeviceName        string
	ContactName       string
	PhoneE164         string
	ContactStatus     string
	NotifyEventList   []string
}

type SMSRecord struct {
	SecurityContextID int
	Direction         string
	PhoneE164         string
	Body              string
	ContactID         *int
	AlertEventID      *int
	ProviderMessageID *string
	Status            string
	ErrorText         *string
}

// WatchdogDeviceList is every live device with its live beacons and its
// tenant's reminder settings.
func (r *WatchdogRepository) WatchdogDeviceList(ctx context.Context) ([]*WatchDevice, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.device_id, d.security_context_id, d.device_name,
		       d.checkin_interval_sec, d.missed_checkin_before_alarm,
		       d.last_checkin_ts,
		       COALESCE(t.time_zone, $1), COALESCE(t.remind_interval_min, $2),
		       COALESCE(t.daily_reminder_time, $3)
		FROM device d
		LEFT JOIN tenant_setting t
		  ON t.security_context_id = d.security_context_id AND t.delete_ts IS NULL
		WHERE d.delete_ts IS NULL
		ORDER BY d.device_id`,
		DefaultTimeZone, DefaultRemindIntervalMin, DefaultDailyReminderTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	deviceList, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*WatchDevice, error) {
		d := &WatchDevice{}
		err := row.Scan(&d.DeviceID, &d.SecurityContextID, &d.DeviceName,
			&d.CheckinIntervalSec, &d.MissedCheckinBeforeAlarm, &d.LastCheckinTS,
			&d.Tenant.TimeZone, &d.Tenant.RemindIntervalMin, &d.Tenant.DailyReminderTime)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read devices: %w", err)
	}
	byID := make(map[int]*WatchDevice, len(deviceList))
	for _, d := range deviceList {
		byID[d.DeviceID] = d
	}

	rows, err = r.db.Query(ctx, `
		SELECT b.device_id, b.security_context_id, b.beacon_id, b.label,
		       b.present, b.last_report_ts
		FROM beacon b
		WHERE b.delete_ts IS NULL
		ORDER BY b.beacon_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list beacons: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var deviceID, securityContext int
		var b WatchBeacon
		if err := rows.Scan(&deviceID, &securityContext, &b.BeaconID, &b.Label,
			&b.Present, &b.LastReportTS); err != nil {
			return nil, fmt.Errorf("failed to read beacon: %w", err)
		}
		if d, ok := byID[deviceID]; ok && d.SecurityContextID == securityContext {
			d.BeaconList = append(d.BeaconList, b)
		}
	}
	return deviceList, rows.Err()
}

// WatchdogOpenAlertList is every alert not yet resolved.
func (r *WatchdogRepository) WatchdogOpenAlertList(ctx context.Context) ([]*WatchAlert, error) {
	rows, err := r.db.Query(ctx, `
		SELECT alert_id, security_context_id, device_id, beacon_id, alert_kind,
		       alert_state, raised_ts, last_notified_ts, ack_ts, ack_by, off_by,
		       last_daily_date
		FROM alert
		WHERE resolved_ts IS NULL AND delete_ts IS NULL
		ORDER BY alert_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list open alerts: %w", err)
	}
	alertList, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*WatchAlert, error) {
		a := &WatchAlert{}
		err := row.Scan(&a.AlertID, &a.SecurityContextID, &a.DeviceID, &a.BeaconID,
			&a.AlertKind, &a.AlertState, &a.RaisedTS, &a.LastNotifiedTS, &a.AckTS,
			&a.AckBy, &a.OffBy, &a.LastDailyDate)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read open alerts: %w", err)
	}
	return alertList, nil
}

// WatchdogRaiseAlert opens an alert and queues its "raised" event, unless
// one is already open for the same device, beacon and kind (raised false).
func (r *WatchdogRepository) WatchdogRaiseAlert(
	ctx context.Context, securityContext int, deviceID int, beaconID *int,
	kind string, now float64,
) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	var alertID int
	err = tx.QueryRow(ctx, `
		INSERT INTO alert (security_context_id, device_id, beacon_id, alert_kind,
		    alert_state, raised_ts, last_notified_ts, notify_count,
		    create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, 'raised', $5, $5, 1, $5, $6, $5, $6)
		ON CONFLICT (device_id, COALESCE(beacon_id, 0), alert_kind)
		    WHERE resolved_ts IS NULL AND delete_ts IS NULL
		DO NOTHING
		RETURNING alert_id`,
		securityContext, deviceID, beaconID, kind, now, watchdogUserID).Scan(&alertID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to raise alert: %w", err)
	}
	if err := insertAlertEvent(ctx, tx, securityContext, alertID, "raised", nil, nil, now); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// WatchdogQueueReminder queues a reminder event, claiming it by the
// alert's last_notified_ts (or, for a daily reminder, last_daily_date) so
// two ticks cannot both queue one. claimed false: another got there first.
func (r *WatchdogRepository) WatchdogQueueReminder(
	ctx context.Context, a *WatchAlert, eventType string, localDate string, now float64,
) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	if eventType == "daily_reminder" {
		ct, err := tx.Exec(ctx, `
			UPDATE alert SET last_daily_date = $1, notify_count = notify_count + 1
			WHERE alert_id = $2 AND security_context_id = $3
			  AND resolved_ts IS NULL AND last_daily_date IS NOT DISTINCT FROM $4`,
			localDate, a.AlertID, a.SecurityContextID, a.LastDailyDate)
		if err != nil {
			return false, fmt.Errorf("failed to claim daily reminder: %w", err)
		}
		if ct.RowsAffected() == 0 {
			return false, nil
		}
	} else {
		ct, err := tx.Exec(ctx, `
			UPDATE alert SET last_notified_ts = $1, notify_count = notify_count + 1
			WHERE alert_id = $2 AND security_context_id = $3
			  AND resolved_ts IS NULL AND last_notified_ts IS NOT DISTINCT FROM $4`,
			now, a.AlertID, a.SecurityContextID, a.LastNotifiedTS)
		if err != nil {
			return false, fmt.Errorf("failed to claim reminder: %w", err)
		}
		if ct.RowsAffected() == 0 {
			return false, nil
		}
	}
	if err := insertAlertEvent(ctx, tx, a.SecurityContextID, a.AlertID, eventType, nil, nil, now); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// WatchdogResolveAlert closes an open alert and queues "reactivated",
// naming whoever acknowledged or turned it off.
func (r *WatchdogRepository) WatchdogResolveAlert(ctx context.Context, a *WatchAlert, now float64) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	ct, err := tx.Exec(ctx, `
		UPDATE alert SET alert_state = 'resolved', resolved_ts = $1,
		    update_ts = $1, update_user_id = $2
		WHERE alert_id = $3 AND security_context_id = $4 AND resolved_ts IS NULL`,
		now, watchdogUserID, a.AlertID, a.SecurityContextID)
	if err != nil {
		return false, fmt.Errorf("failed to resolve alert: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return false, nil
	}
	if err := insertAlertEvent(ctx, tx, a.SecurityContextID, a.AlertID, "reactivated", nil, nil, now); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// WatchdogUnsentEventList is the outbox, oldest first.
func (r *WatchdogRepository) WatchdogUnsentEventList(ctx context.Context, limit int) ([]*WatchEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.alert_event_id, e.security_context_id, e.alert_id, e.event_type,
		       e.actor_name, e.actor_via, e.create_ts,
		       a.alert_kind, a.raised_ts, a.ack_by, a.off_by,
		       d.device_id, d.device_name, d.last_checkin_ts,
		       b.label, b.last_seen_ts,
		       COALESCE(t.time_zone, $1)
		FROM alert_event e
		JOIN alert a ON a.alert_id = e.alert_id AND a.security_context_id = e.security_context_id
		JOIN device d ON d.device_id = a.device_id AND d.security_context_id = a.security_context_id
		LEFT JOIN beacon b ON b.beacon_id = a.beacon_id AND b.security_context_id = a.security_context_id
		LEFT JOIN tenant_setting t
		  ON t.security_context_id = e.security_context_id AND t.delete_ts IS NULL
		WHERE e.sent_ts IS NULL AND e.delete_ts IS NULL
		ORDER BY e.alert_event_id
		LIMIT $2`, DefaultTimeZone, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unsent events: %w", err)
	}
	eventList, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*WatchEvent, error) {
		e := &WatchEvent{}
		err := row.Scan(&e.AlertEventID, &e.SecurityContextID, &e.AlertID, &e.EventType,
			&e.ActorName, &e.ActorVia, &e.CreateTS,
			&e.AlertKind, &e.RaisedTS, &e.AckBy, &e.OffBy,
			&e.DeviceID, &e.DeviceName, &e.DeviceLastCheckin,
			&e.BeaconLabel, &e.BeaconLastSeenTS, &e.TimeZone)
		return e, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read unsent events: %w", err)
	}
	return eventList, nil
}

func (r *WatchdogRepository) WatchdogMarkEventSent(
	ctx context.Context, alertEventID int, securityContext int, now float64,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE alert_event SET sent_ts = $1
		WHERE alert_event_id = $2 AND security_context_id = $3`,
		now, alertEventID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to mark event sent: %w", err)
	}
	return nil
}

const watchContactColumnList = `
	c.contact_id, c.security_context_id, c.device_id, d.device_name,
	c.contact_name, c.phone_e164, c.contact_status, c.notify_event_list`

func collectWatchContactList(rows pgx.Rows) ([]*WatchContact, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*WatchContact, error) {
		c := &WatchContact{}
		err := row.Scan(&c.ContactID, &c.SecurityContextID, &c.DeviceID, &c.DeviceName,
			&c.ContactName, &c.PhoneE164, &c.ContactStatus, &c.NotifyEventList)
		return c, err
	})
}

// WatchdogDeviceContactList is a device's live contacts, any status.
func (r *WatchdogRepository) WatchdogDeviceContactList(
	ctx context.Context, securityContext int, deviceID int,
) ([]*WatchContact, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+watchContactColumnList+`
		FROM contact c
		JOIN device d ON d.device_id = c.device_id AND d.security_context_id = c.security_context_id
		WHERE c.security_context_id = $1 AND c.device_id = $2 AND c.delete_ts IS NULL
		ORDER BY c.contact_id`, securityContext, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list device contacts: %w", err)
	}
	contactList, err := collectWatchContactList(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read device contacts: %w", err)
	}
	return contactList, nil
}

// WatchdogPendingInviteList is every invited contact whose invite text is
// still owed.
func (r *WatchdogRepository) WatchdogPendingInviteList(ctx context.Context) ([]*WatchContact, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+watchContactColumnList+`
		FROM contact c
		JOIN device d ON d.device_id = c.device_id AND d.security_context_id = c.security_context_id
		WHERE c.contact_status = 'invited' AND c.invite_sent_ts IS NULL
		  AND c.delete_ts IS NULL AND d.delete_ts IS NULL
		ORDER BY c.contact_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending invites: %w", err)
	}
	contactList, err := collectWatchContactList(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read pending invites: %w", err)
	}
	return contactList, nil
}

func (r *WatchdogRepository) WatchdogMarkInviteSent(
	ctx context.Context, contactID int, securityContext int, now float64,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE contact SET invite_sent_ts = $1
		WHERE contact_id = $2 AND security_context_id = $3`,
		now, contactID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to mark invite sent: %w", err)
	}
	return nil
}

// ContactListByPhone is every live contact with this number, across
// tenants: an inbound text names no tenant, only its sender.
func (r *WatchdogRepository) ContactListByPhone(ctx context.Context, phone string) ([]*WatchContact, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+watchContactColumnList+`
		FROM contact c
		JOIN device d ON d.device_id = c.device_id AND d.security_context_id = c.security_context_id
		WHERE c.phone_e164 = $1 AND c.delete_ts IS NULL AND d.delete_ts IS NULL
		ORDER BY c.contact_id`, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts by phone: %w", err)
	}
	contactList, err := collectWatchContactList(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read contacts by phone: %w", err)
	}
	return contactList, nil
}

// SetContactStatus moves a contact to active or opted_out and stamps when.
func (r *WatchdogRepository) SetContactStatus(
	ctx context.Context, contactID int, securityContext int, status string, now float64,
) error {
	column := "opted_in_ts"
	if status == "opted_out" {
		column = "opted_out_ts"
	}
	_, err := r.db.Exec(ctx, `
		UPDATE contact SET contact_status = $1, `+column+` = $2
		WHERE contact_id = $3 AND security_context_id = $4 AND delete_ts IS NULL`,
		status, now, contactID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to set contact status: %w", err)
	}
	return nil
}

// AcknowledgeDeviceAlerts acknowledges every raised alert of one device
// for an inbound OK and queues an "acknowledged" event for each. It
// returns how many it acknowledged.
func (r *WatchdogRepository) AcknowledgeDeviceAlerts(
	ctx context.Context, securityContext int, deviceID int, actorName string, now float64,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	rows, err := tx.Query(ctx, `
		UPDATE alert SET alert_state = 'acknowledged', ack_ts = $1, ack_by = $2,
		    ack_via = 'sms', update_ts = $1, update_user_id = $3
		WHERE security_context_id = $4 AND device_id = $5
		  AND alert_state = 'raised' AND resolved_ts IS NULL AND delete_ts IS NULL
		RETURNING alert_id`,
		now, actorName, watchdogUserID, securityContext, deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to acknowledge alerts: %w", err)
	}
	alertIDList, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("failed to read acknowledged alerts: %w", err)
	}
	via := "sms"
	for _, alertID := range alertIDList {
		if err := insertAlertEvent(ctx, tx, securityContext, alertID, "acknowledged", &actorName, &via, now); err != nil {
			return 0, err
		}
	}
	return len(alertIDList), tx.Commit(ctx)
}

func (r *WatchdogRepository) InsertSMS(ctx context.Context, m SMSRecord, now float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sms_message (security_context_id, direction, phone_e164, body,
		    contact_id, alert_event_id, provider_message_id, status, error_text,
		    create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $10, $11)`,
		m.SecurityContextID, m.Direction, m.PhoneE164, m.Body, m.ContactID,
		m.AlertEventID, m.ProviderMessageID, m.Status, m.ErrorText, now, watchdogUserID)
	if err != nil {
		return fmt.Errorf("failed to record text: %w", err)
	}
	return nil
}

func insertAlertEvent(
	ctx context.Context, tx pgx.Tx, securityContext int, alertID int,
	eventType string, actorName *string, actorVia *string, now float64,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO alert_event (security_context_id, alert_id, event_type,
		    actor_name, actor_via, create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $6, $7)`,
		securityContext, alertID, eventType, actorName, actorVia, now, watchdogUserID)
	if err != nil {
		return fmt.Errorf("failed to queue %s event: %w", eventType, err)
	}
	return nil
}
