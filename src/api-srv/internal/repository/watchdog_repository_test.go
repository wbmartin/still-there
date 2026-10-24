package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

// The watchdog's SQL against the scratch database: the claims that keep
// a tick from doubling a text (one open alert per key, compare-and-set
// reminders), the outbox read, and the inbound-text writes. Rows hang off
// devices named "WatchdogTest ..." in tenant 2 and are removed with them.

func TestWatchdogRepository(t *testing.T) {
	suite.Run(t, &WatchdogRepositoryTestSuite{})
}

type WatchdogRepositoryTestSuite struct {
	suite.Suite
	db   *pgxpool.Pool
	repo *WatchdogRepository
}

func (s *WatchdogRepositoryTestSuite) SetupSuite() {
	s.db = connectTestDB(s.T())
	s.repo = NewWatchdogRepository(s.db)
	s.cleanup()
}

func (s *WatchdogRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *WatchdogRepositoryTestSuite) cleanup() {
	ctx := context.Background()
	const testDevice = `SELECT device_id FROM device WHERE device_name LIKE 'WatchdogTest%'`
	for _, statement := range []string{
		`DELETE FROM sms_message WHERE phone_e164 LIKE '+1555777%'`,
		`DELETE FROM alert_event WHERE alert_id IN (SELECT alert_id
			FROM alert WHERE device_id IN (` + testDevice + `))`,
		`DELETE FROM alert WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM beacon WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM contact WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM device WHERE device_name LIKE 'WatchdogTest%'`,
	} {
		_, err := s.db.Exec(ctx, statement)
		s.Require().NoError(err)
	}
}

const wdTS = 1800000000.0

// fixture makes a device with one band, an active and an invited contact.
func (s *WatchdogRepositoryTestSuite) fixture(name string) (deviceID, beaconID int) {
	ctx := context.Background()
	s.Require().NoError(s.db.QueryRow(ctx, `
		INSERT INTO device (security_context_id, device_name, checkin_interval_sec,
		    missed_checkin_before_alarm, absent_after_sec, config_version,
		    last_checkin_ts, create_ts, create_user_id, update_ts, update_user_id)
		VALUES (2, $1, 60, 3, 600, 1, $2, $2, 1, $2, 1)
		RETURNING device_id`, "WatchdogTest "+name, wdTS).Scan(&deviceID))
	s.Require().NoError(s.db.QueryRow(ctx, `
		INSERT INTO beacon (security_context_id, device_id, address, label, present,
		    last_report_ts, create_ts, create_user_id, update_ts, update_user_id)
		VALUES (2, $1, 'C3:00:00:98:F3:19', 'B7', false, $2, $2, 1, $2, 1)
		RETURNING beacon_id`, deviceID, wdTS).Scan(&beaconID))
	_, err := s.db.Exec(ctx, `
		INSERT INTO contact (security_context_id, device_id, contact_name, phone_e164,
		    contact_status, notify_event_list, create_ts, create_user_id, update_ts, update_user_id)
		VALUES (2, $1, 'Ann', '+15557770001', 'active', ARRAY['raised','acknowledged'], $2, 1, $2, 1),
		       (2, $1, 'Bob', '+15557770002', 'invited', ARRAY['raised'], $2, 1, $2, 1)`,
		deviceID, wdTS)
	s.Require().NoError(err)
	return deviceID, beaconID
}

func (s *WatchdogRepositoryTestSuite) openAlert(deviceID int) *WatchAlert {
	list, err := s.repo.WatchdogOpenAlertList(context.Background())
	s.Require().NoError(err)
	for _, a := range list {
		if a.DeviceID == deviceID {
			return a
		}
	}
	return nil
}

// eventTypeList is the device's unsent outbox, oldest first.
func (s *WatchdogRepositoryTestSuite) eventTypeList(deviceID int) []string {
	list, err := s.repo.WatchdogUnsentEventList(context.Background(), 1000)
	s.Require().NoError(err)
	var typeList []string
	for _, e := range list {
		if e.DeviceID == deviceID {
			typeList = append(typeList, e.EventType)
		}
	}
	return typeList
}

func (s *WatchdogRepositoryTestSuite) TestDeviceListCarriesBeaconsAndTenantDefaults() {
	deviceID, beaconID := s.fixture("list")
	list, err := s.repo.WatchdogDeviceList(context.Background())
	s.Require().NoError(err)
	for _, d := range list {
		if d.DeviceID != deviceID {
			continue
		}
		s.Require().Len(d.BeaconList, 1)
		s.Equal(beaconID, d.BeaconList[0].BeaconID)
		s.False(*d.BeaconList[0].Present)
		s.NotEmpty(d.Tenant.TimeZone)
		s.Positive(d.Tenant.RemindIntervalMin)
		return
	}
	s.Fail("device not listed")
}

func (s *WatchdogRepositoryTestSuite) TestAlertLifecycle() {
	ctx := context.Background()
	deviceID, beaconID := s.fixture("lifecycle")

	raised, err := s.repo.WatchdogRaiseAlert(ctx, 2, deviceID, &beaconID, "beacon_absent", wdTS)
	s.Require().NoError(err)
	s.True(raised)
	raised, err = s.repo.WatchdogRaiseAlert(ctx, 2, deviceID, &beaconID, "beacon_absent", wdTS+1)
	s.Require().NoError(err)
	s.False(raised, "a second open alert for the same band and kind")
	s.Equal([]string{"raised"}, s.eventTypeList(deviceID))

	a := s.openAlert(deviceID)
	s.Require().NotNil(a)
	var updateBefore float64
	s.Require().NoError(s.db.QueryRow(ctx, `SELECT update_ts FROM alert WHERE alert_id = $1`, a.AlertID).Scan(&updateBefore))

	stale := *a
	stale.LastNotifiedTS = nil
	claimed, err := s.repo.WatchdogQueueReminder(ctx, &stale, "reminder", "", wdTS+900)
	s.Require().NoError(err)
	s.False(claimed, "a reminder claimed against a stale last_notified_ts")
	claimed, err = s.repo.WatchdogQueueReminder(ctx, a, "reminder", "", wdTS+900)
	s.Require().NoError(err)
	s.True(claimed)
	var updateAfter float64
	s.Require().NoError(s.db.QueryRow(ctx, `SELECT update_ts FROM alert WHERE alert_id = $1`, a.AlertID).Scan(&updateAfter))
	s.Equal(updateBefore, updateAfter, "a reminder must not move the alert's lock")

	n, err := s.repo.AcknowledgeDeviceAlerts(ctx, 2, deviceID, "Ann", wdTS+1000)
	s.Require().NoError(err)
	s.Equal(1, n)
	n, err = s.repo.AcknowledgeDeviceAlerts(ctx, 2, deviceID, "Ann", wdTS+1001)
	s.Require().NoError(err)
	s.Equal(0, n, "an acknowledged alert acknowledged again")
	a = s.openAlert(deviceID)
	s.Equal("acknowledged", a.AlertState)
	s.Equal("Ann", *a.AckBy)

	claimed, err = s.repo.WatchdogQueueReminder(ctx, a, "daily_reminder", "2027-01-15", wdTS+2000)
	s.Require().NoError(err)
	s.True(claimed)
	claimed, err = s.repo.WatchdogQueueReminder(ctx, a, "daily_reminder", "2027-01-15", wdTS+2001)
	s.Require().NoError(err)
	s.False(claimed, "two daily reminders for one date")

	resolved, err := s.repo.WatchdogResolveAlert(ctx, a, wdTS+3000)
	s.Require().NoError(err)
	s.True(resolved)
	s.Nil(s.openAlert(deviceID))
	s.Equal([]string{"raised", "reminder", "acknowledged", "daily_reminder", "reactivated"},
		s.eventTypeList(deviceID))

	raised, err = s.repo.WatchdogRaiseAlert(ctx, 2, deviceID, &beaconID, "beacon_absent", wdTS+4000)
	s.Require().NoError(err)
	s.True(raised, "after resolve the band may raise again")
}

func (s *WatchdogRepositoryTestSuite) TestOutboxReadAndMarkSent() {
	ctx := context.Background()
	deviceID, beaconID := s.fixture("outbox")
	_, err := s.repo.WatchdogRaiseAlert(ctx, 2, deviceID, &beaconID, "beacon_absent", wdTS)
	s.Require().NoError(err)

	list, err := s.repo.WatchdogUnsentEventList(ctx, 1000)
	s.Require().NoError(err)
	var mine *WatchEvent
	for _, e := range list {
		if e.DeviceID == deviceID {
			mine = e
		}
	}
	s.Require().NotNil(mine)
	s.Equal("WatchdogTest outbox", mine.DeviceName)
	s.Equal("B7", *mine.BeaconLabel)
	s.Equal("beacon_absent", mine.AlertKind)

	s.Require().NoError(s.repo.WatchdogMarkEventSent(ctx, mine.AlertEventID, 2, wdTS+1))
	s.Empty(s.eventTypeList(deviceID))

	contactList, err := s.repo.WatchdogDeviceContactList(ctx, 2, deviceID)
	s.Require().NoError(err)
	s.Len(contactList, 2)
	otherTenant, err := s.repo.WatchdogDeviceContactList(ctx, 3, deviceID)
	s.Require().NoError(err)
	s.Empty(otherTenant, "another tenant read this device's contacts")
}

func (s *WatchdogRepositoryTestSuite) TestInviteAndContactStatus() {
	ctx := context.Background()
	deviceID, _ := s.fixture("invite")

	pending, err := s.repo.WatchdogPendingInviteList(ctx)
	s.Require().NoError(err)
	var bob *WatchContact
	for _, c := range pending {
		if c.DeviceID == deviceID {
			s.Equal("Bob", c.ContactName)
			bob = c
		}
	}
	s.Require().NotNil(bob)
	s.Require().NoError(s.repo.WatchdogMarkInviteSent(ctx, bob.ContactID, 2, wdTS))
	pending, err = s.repo.WatchdogPendingInviteList(ctx)
	s.Require().NoError(err)
	for _, c := range pending {
		s.NotEqual(bob.ContactID, c.ContactID, "invite still owed after it was sent")
	}

	// Bob is on every fixture's household; this test's row is the one.
	bobRow := func() *WatchContact {
		byPhone, err := s.repo.ContactListByPhone(ctx, "+15557770002")
		s.Require().NoError(err)
		for _, c := range byPhone {
			if c.ContactID == bob.ContactID {
				return c
			}
		}
		s.FailNow("contact not found by phone")
		return nil
	}
	s.Equal("invited", bobRow().ContactStatus)
	s.Require().NoError(s.repo.SetContactStatus(ctx, bob.ContactID, 2, "active", wdTS+1))
	s.Require().NoError(s.repo.SetContactStatus(ctx, bob.ContactID, 3, "opted_out", wdTS+2))
	s.Equal("active", bobRow().ContactStatus, "a write naming another tenant changed the row")

	contactID := bob.ContactID
	s.Require().NoError(s.repo.InsertSMS(ctx, SMSRecord{
		SecurityContextID: 2, Direction: "in", PhoneE164: "+15557770002",
		Body: "Y", ContactID: &contactID, Status: "received",
	}, wdTS+3))
}
