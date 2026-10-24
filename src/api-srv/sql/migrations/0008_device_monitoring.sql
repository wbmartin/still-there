-- Still There monitoring: the Pi base station in a home (device), the BLE
-- bands it watches (beacon), the people texted about it (contact), the
-- alarms (alert) and the texts they cause (alert_event outbox,
-- sms_message log). tasks/todo.md, Server, holds the agreed design.
--
-- Runtime columns (device.last_*, beacon.present / last_*) are written by
-- check-ins and the watchdog without touching update_ts, so an admin's
-- edit of the same row does not 409 on every check-in. Timestamps are
-- epoch seconds; the system writes as user 0.

-- One row per tenant: when and how often reminders go out.
-- daily_reminder_time is 'HH:MM' in time_zone (an IANA name).
CREATE TABLE tenant_setting (
    tenant_setting_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    time_zone TEXT NOT NULL,
    remind_interval_min INTEGER NOT NULL,
    daily_reminder_time TEXT NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE UNIQUE INDEX idx_tenant_setting_sc
    ON tenant_setting(security_context_id) WHERE delete_ts IS NULL;

-- The Pi. token_hash is the hex SHA-256 of the device token, shown once
-- at issue (auth.NewInviteToken); NULL until one is issued. The check-in
-- route looks a device up by it before there is a tenant to scope by.
CREATE TABLE device (
    device_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    device_name TEXT NOT NULL,
    checkin_interval_sec INTEGER NOT NULL,
    missed_checkin_before_alarm INTEGER NOT NULL,
    absent_after_sec INTEGER NOT NULL,
    config_version INTEGER NOT NULL,
    token_hash TEXT,
    token_issued_ts NUMERIC(17,7),
    last_checkin_ts NUMERIC(17,7),
    last_checkin_json JSONB,
    last_agent_version TEXT,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE UNIQUE INDEX idx_device_token_hash ON device(token_hash);
CREATE INDEX idx_device_sc_update_ts ON device(security_context_id, update_ts);

-- A band a device watches, by BLE address (upper case). present is the
-- Pi's last report (NULL before any); last_seen_ts the Pi's last
-- sighting; last_report_ts the check-in that carried it.
CREATE TABLE beacon (
    beacon_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    address TEXT NOT NULL,
    label TEXT NOT NULL,
    present BOOLEAN,
    last_seen_ts NUMERIC(17,7),
    last_rssi INTEGER,
    last_report_ts NUMERIC(17,7),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE UNIQUE INDEX idx_beacon_sc_device_address
    ON beacon(security_context_id, device_id, address) WHERE delete_ts IS NULL;
CREATE INDEX idx_beacon_sc_update_ts ON beacon(security_context_id, update_ts);

-- Every check-in as received. Insert-only: update_* repeat the create
-- pair. Kept in full for the PoC.
CREATE TABLE device_checkin (
    device_checkin_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    received_ts NUMERIC(17,7) NOT NULL,
    sent_ts NUMERIC(17,7),
    reason TEXT NOT NULL,
    body_json JSONB NOT NULL,
    source_ip TEXT,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_device_checkin_sc_device_received
    ON device_checkin(security_context_id, device_id, received_ts);

-- A person texted about one device. contact_status: invited (invite text
-- due or sent, waiting for Y) | active (replied Y) | opted_out (replied
-- STOP; only their START re-enables). invite_sent_ts NULL = the watchdog
-- owes the invite text. notify_event_list is the events this contact
-- wants: raised, reminder, acknowledged, turned_off, reactivated,
-- daily_reminder.
CREATE TABLE contact (
    contact_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    contact_name TEXT NOT NULL,
    phone_e164 TEXT NOT NULL,
    contact_status TEXT NOT NULL,
    notify_event_list TEXT[] NOT NULL,
    invite_sent_ts NUMERIC(17,7),
    opted_in_ts NUMERIC(17,7),
    opted_out_ts NUMERIC(17,7),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE UNIQUE INDEX idx_contact_sc_device_phone
    ON contact(security_context_id, device_id, phone_e164) WHERE delete_ts IS NULL;
-- Inbound texts find their contacts by number alone, across tenants.
CREATE INDEX idx_contact_phone ON contact(phone_e164) WHERE delete_ts IS NULL;
CREATE INDEX idx_contact_sc_update_ts ON contact(security_context_id, update_ts);

-- One alarm. alert_kind: device_offline (beacon_id NULL) | beacon_absent.
-- alert_state: raised | acknowledged | turned_off | resolved. Open while
-- resolved_ts IS NULL; at most one open alarm per device, beacon and kind,
-- so a band raises again only after it was seen. ack_by / off_by name the
-- person (contact name or user name), *_via is sms or ui.
-- last_daily_date is the tenant-local date of the last daily reminder.
CREATE TABLE alert (
    alert_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    beacon_id INTEGER,
    alert_kind TEXT NOT NULL,
    alert_state TEXT NOT NULL,
    raised_ts NUMERIC(17,7) NOT NULL,
    last_notified_ts NUMERIC(17,7),
    notify_count INTEGER NOT NULL,
    ack_ts NUMERIC(17,7),
    ack_by TEXT,
    ack_via TEXT,
    off_ts NUMERIC(17,7),
    off_by TEXT,
    off_via TEXT,
    resolved_ts NUMERIC(17,7),
    last_daily_date TEXT,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE UNIQUE INDEX idx_alert_open
    ON alert(device_id, COALESCE(beacon_id, 0), alert_kind)
    WHERE resolved_ts IS NULL AND delete_ts IS NULL;
CREATE INDEX idx_alert_sc_update_ts ON alert(security_context_id, update_ts);

-- Outbox: one row per thing to tell an alert's contacts. event_type is a
-- notify_event_list value. Written by the watchdog (raised, reminder,
-- daily_reminder, reactivated) and by an acknowledge or turn-off from a
-- text or the UI; the watchdog sends rows with sent_ts NULL. actor_name /
-- actor_via say who did it, for acknowledged / turned_off / reactivated.
-- Insert, then sent_ts once; update_* repeat the create pair.
CREATE TABLE alert_event (
    alert_event_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    alert_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    actor_name TEXT,
    actor_via TEXT,
    sent_ts NUMERIC(17,7),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_alert_event_unsent ON alert_event(alert_event_id) WHERE sent_ts IS NULL;
CREATE INDEX idx_alert_event_sc_alert ON alert_event(security_context_id, alert_id);

-- Every text in or out. direction: out | in. A text in from a number no
-- contact has is kept with security_context_id 0 and contact_id NULL.
-- status: sent | failed | logged (SMS_PROVIDER=log) | received.
-- body is what was texted and may hold anything a person typed.
CREATE TABLE sms_message (
    sms_message_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    direction TEXT NOT NULL,
    phone_e164 TEXT NOT NULL,
    body TEXT NOT NULL,
    contact_id INTEGER,
    alert_event_id INTEGER,
    provider_message_id TEXT,
    status TEXT NOT NULL,
    error_text TEXT,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_sms_message_sc_create_ts ON sms_message(security_context_id, create_ts);
CREATE INDEX idx_sms_message_phone_create_ts ON sms_message(phone_e164, create_ts);

-- New tables need their own grants (doc/900).
GRANT SELECT ON tenant_setting, device, beacon, device_checkin, contact, alert, alert_event, sms_message TO db_reader;
GRANT ALL PRIVILEGES ON tenant_setting, device, beacon, device_checkin, contact, alert, alert_event, sms_message TO db_writer;
GRANT ALL ON SEQUENCE tenant_setting_tenant_setting_id_seq, device_device_id_seq, beacon_beacon_id_seq,
    device_checkin_device_checkin_id_seq, contact_contact_id_seq, alert_alert_id_seq,
    alert_event_alert_event_id_seq, sms_message_sms_message_id_seq TO db_writer;
