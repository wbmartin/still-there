package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"stillthere-api/internal/model"
)

// The device-monitoring tables (migration 0008): device, device_checkin,
// beacon, contact, alert, alert_event and tenant_setting. Every row a
// test makes hangs off a device named "RepoTest ..." and is removed with
// it; tenant_setting rows of the two test tenants are removed outright.
// Isolation is proved per table: testTenant (2) never reads, changes or
// deletes an otherTenant (3) row, and the row is left as it was.

func TestDeviceMonitoringRepository(t *testing.T) {
	suite.Run(t, &DeviceMonitoringRepositoryTestSuite{})
}

type DeviceMonitoringRepositoryTestSuite struct {
	suite.Suite
	db                      *pgxpool.Pool
	deviceRepository        *DeviceRepository
	beaconRepository        *BeaconRepository
	contactRepository       *ContactRepository
	alertRepository         *AlertRepository
	tenantSettingRepository *TenantSettingRepository
}

func (s *DeviceMonitoringRepositoryTestSuite) SetupSuite() {
	s.db = connectTestDB(s.T())
	s.deviceRepository = NewDeviceRepository(s.db)
	s.beaconRepository = NewBeaconRepository(s.db)
	s.contactRepository = NewContactRepository(s.db)
	s.alertRepository = NewAlertRepository(s.db)
	s.tenantSettingRepository = NewTenantSettingRepository(s.db)
	s.cleanup()
}

func (s *DeviceMonitoringRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

// cleanup removes the suite's rows: test data in a scratch database.
func (s *DeviceMonitoringRepositoryTestSuite) cleanup() {
	ctx := context.Background()
	const testDevice = `SELECT device_id FROM device
		WHERE device_name LIKE 'RepoTest%'`
	for _, statement := range []string{
		`DELETE FROM alert_event WHERE alert_id IN (SELECT alert_id
			FROM alert WHERE device_id IN (` + testDevice + `))`,
		`DELETE FROM alert WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM device_checkin WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM beacon WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM contact WHERE device_id IN (` + testDevice + `)`,
		`DELETE FROM device WHERE device_name LIKE 'RepoTest%'`,
		`DELETE FROM tenant_setting WHERE security_context_id IN (2, 3)`,
	} {
		_, err := s.db.Exec(ctx, statement)
		s.Require().NoError(err)
	}
}

const repoTestTS = 1234567890.123

func (s *DeviceMonitoringRepositoryTestSuite) createDevice(
	name string, tenant int,
) *model.DeviceModel {
	created, err := s.deviceRepository.CreateDevice(context.Background(),
		&model.DeviceModel{
			DeviceName: "RepoTest " + name, CheckinIntervalSec: 60,
			MissedCheckinBeforeAlarm: 3, AbsentAfterSec: 600,
			CreateTS: repoTestTS, CreateUserID: 1,
			UpdateTS: repoTestTS, UpdateUserID: 1,
		}, tenant)
	s.Require().NoError(err)
	return created
}

func (s *DeviceMonitoringRepositoryTestSuite) createBeacon(
	device *model.DeviceModel, address string,
) *model.BeaconModel {
	created, err := s.beaconRepository.CreateBeacon(context.Background(),
		&model.BeaconModel{
			DeviceID: device.DeviceID, Address: address, Label: "Band",
			CreateTS: repoTestTS, CreateUserID: 1,
			UpdateTS: repoTestTS, UpdateUserID: 1,
		}, device.SecurityContext)
	s.Require().NoError(err)
	return created
}

func (s *DeviceMonitoringRepositoryTestSuite) createContact(
	device *model.DeviceModel, phone string,
) *model.ContactModel {
	created, err := s.contactRepository.CreateContact(context.Background(),
		&model.ContactModel{
			DeviceID: device.DeviceID, ContactName: "Sam",
			PhoneE164: phone, NotifyEventList: []string{"raised"},
			CreateTS: repoTestTS, CreateUserID: 1,
			UpdateTS: repoTestTS, UpdateUserID: 1,
		}, device.SecurityContext)
	s.Require().NoError(err)
	return created
}

// createAlert inserts an alarm as the watchdog would; the API makes none.
func (s *DeviceMonitoringRepositoryTestSuite) createAlert(
	device *model.DeviceModel, state string, resolvedTS *float64,
) *model.AlertModel {
	var alertID int
	err := s.db.QueryRow(context.Background(), `
		INSERT INTO alert (security_context_id, device_id, alert_kind,
			alert_state, raised_ts, notify_count, resolved_ts, create_ts,
			create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, 'device_offline', $3, $4, 1, $5, $4, 0, $4, 0)
		RETURNING alert_id`,
		device.SecurityContext, device.DeviceID, state, repoTestTS,
		resolvedTS).Scan(&alertID)
	s.Require().NoError(err)
	return &model.AlertModel{AlertID: alertID, UpdateTS: repoTestTS}
}

func (s *DeviceMonitoringRepositoryTestSuite) readDevice(
	device *model.DeviceModel,
) *model.DeviceModel {
	deviceList, err := s.deviceRepository.RetrieveDeviceList(
		context.Background(), []int{device.DeviceID},
		device.SecurityContext, 0)
	s.Require().NoError(err)
	s.Require().Len(deviceList, 1)
	return deviceList[0]
}

func (s *DeviceMonitoringRepositoryTestSuite) readBeacon(
	beacon *model.BeaconModel,
) *model.BeaconModel {
	beaconList, err := s.beaconRepository.RetrieveBeaconList(
		context.Background(), []int{beacon.BeaconID}, 0,
		beacon.SecurityContext, 0)
	s.Require().NoError(err)
	s.Require().Len(beaconList, 1)
	return beaconList[0]
}

func (s *DeviceMonitoringRepositoryTestSuite) TestDeviceUpdateBumpsConfigOnSettingsOnly() {
	ctx := context.Background()
	device := s.createDevice("Update", testTenant)
	s.Equal(1, device.ConfigVersion)

	rename := *device
	rename.DeviceName = "RepoTest Renamed"
	rename.UpdateTS = repoTestTS + 1
	updated, err := s.deviceRepository.UpdateDevice(ctx, &rename, repoTestTS)
	s.Require().NoError(err)
	s.Equal(1, updated.ConfigVersion, "a rename is not config")
	s.Equal("RepoTest Renamed", updated.DeviceName)

	settings := *updated
	settings.AbsentAfterSec = 300
	settings.UpdateTS = repoTestTS + 2
	updated, err = s.deviceRepository.UpdateDevice(
		ctx, &settings, repoTestTS+1)
	s.Require().NoError(err)
	s.Equal(2, updated.ConfigVersion)
	s.Equal(300, updated.AbsentAfterSec)

	// A stale lock is a conflict and changes nothing.
	settings.AbsentAfterSec = 30
	_, err = s.deviceRepository.UpdateDevice(ctx, &settings, repoTestTS)
	s.True(errors.Is(err, ErrConflict), "stale: %v", err)
	s.Equal(300, s.readDevice(device).AbsentAfterSec)

	s.Require().NoError(s.deviceRepository.DeleteDevice(ctx,
		device.DeviceID, testTenant, repoTestTS+2, repoTestTS+3, 1))
	list, err := s.deviceRepository.RetrieveDeviceList(
		ctx, []int{device.DeviceID}, testTenant, 0)
	s.Require().NoError(err)
	s.Empty(list)
}

func (s *DeviceMonitoringRepositoryTestSuite) TestDeviceTokenReissueAndDelete() {
	ctx := context.Background()
	device := s.createDevice("Token", testTenant)
	s.Require().NoError(s.deviceRepository.IssueDeviceToken(ctx,
		device.DeviceID, testTenant, repoTestTS, "repotest-hash-1",
		repoTestTS+1, 1))
	found, err := s.deviceRepository.RetrieveDeviceByTokenHash(
		ctx, "repotest-hash-1")
	s.Require().NoError(err)
	s.Equal(device.DeviceID, found.DeviceID)
	s.Equal(testTenant, found.SecurityContext)
	s.Require().NotNil(found.TokenIssuedTS)

	// The lock is the row's update_ts, which the issue moved.
	err = s.deviceRepository.IssueDeviceToken(ctx, device.DeviceID,
		testTenant, repoTestTS, "repotest-hash-2", repoTestTS+2, 1)
	s.True(errors.Is(err, ErrConflict), "stale: %v", err)

	s.Require().NoError(s.deviceRepository.IssueDeviceToken(ctx,
		device.DeviceID, testTenant, repoTestTS+1, "repotest-hash-2",
		repoTestTS+2, 1))
	_, err = s.deviceRepository.RetrieveDeviceByTokenHash(
		ctx, "repotest-hash-1")
	s.True(errors.Is(err, ErrNotFound), "old token: %v", err)
	_, err = s.deviceRepository.RetrieveDeviceByTokenHash(
		ctx, "repotest-hash-2")
	s.Require().NoError(err)

	s.Require().NoError(s.deviceRepository.DeleteDevice(ctx,
		device.DeviceID, testTenant, repoTestTS+2, repoTestTS+3, 1))
	_, err = s.deviceRepository.RetrieveDeviceByTokenHash(
		ctx, "repotest-hash-2")
	s.True(errors.Is(err, ErrNotFound), "deleted device: %v", err)
}

// A check-in records the row, the device's and the band's runtime columns
// and touches neither update_ts; an address the device does not watch is
// answered, a deleted band is one of those.
func (s *DeviceMonitoringRepositoryTestSuite) TestRecordDeviceCheckin() {
	ctx := context.Background()
	device := s.createDevice("Checkin", testTenant)
	watched := s.createBeacon(device, "C3:00:00:98:F3:19")
	gone := s.createBeacon(device, "C3:00:00:98:F3:20")
	s.Require().NoError(s.beaconRepository.DeleteBeacon(ctx, gone.BeaconID,
		testTenant, repoTestTS, repoTestTS+1, 1))
	device = s.readDevice(device)
	configBefore := device.ConfigVersion

	seen := time.Unix(1792833460, 0)
	rssi := -61
	body := json.RawMessage(`{"reason":"interval","agentVersion":"0.1.0"}`)
	sentTS := 1792833464.0
	configVersion, unknownList, err := s.deviceRepository.RecordDeviceCheckin(
		ctx, device.DeviceID, testTenant, &model.DeviceCheckinRecord{
			ReceivedTS: 1792833465.5, SentTS: &sentTS, Reason: "interval",
			BodyJSON: body, SourceIP: "192.0.2.10", AgentVersion: "0.1.0",
		}, []model.DeviceCheckinBeacon{
			{Address: "c3:00:00:98:f3:19", Present: true, LastSeen: &seen,
				RSSI: &rssi},
			{Address: "C3:00:00:98:F3:20", Present: true},
			{Address: "AA:BB:CC:DD:EE:FF", Present: false},
		})
	s.Require().NoError(err)
	s.Equal(configBefore, configVersion)
	s.Equal([]string{"C3:00:00:98:F3:20", "AA:BB:CC:DD:EE:FF"}, unknownList)

	after := s.readDevice(device)
	s.Equal(device.UpdateTS, after.UpdateTS, "device update_ts moved")
	s.Require().NotNil(after.LastCheckinTS)
	s.Equal(1792833465.5, *after.LastCheckinTS)
	s.Equal("0.1.0", *after.LastAgentVersion)
	s.JSONEq(string(body), string(after.LastCheckinJSON))

	band := s.readBeacon(watched)
	s.Equal(watched.UpdateTS, band.UpdateTS, "beacon update_ts moved")
	s.Require().NotNil(band.Present)
	s.True(*band.Present)
	s.Equal(1792833460.0, *band.LastSeenTS)
	s.Equal(-61, *band.LastRSSI)
	s.Equal(1792833465.5, *band.LastReportTS)

	var count int
	var storedReason, storedIP string
	var storedSent float64
	s.Require().NoError(s.db.QueryRow(ctx, `
		SELECT count(*), max(reason), max(source_ip), max(sent_ts)
		FROM device_checkin
		WHERE device_id = $1 AND security_context_id = $2
		  AND create_user_id = 0`, device.DeviceID, testTenant).
		Scan(&count, &storedReason, &storedIP, &storedSent))
	s.Equal(1, count)
	s.Equal("interval", storedReason)
	s.Equal("192.0.2.10", storedIP)
	s.Equal(sentTS, storedSent)

	// A later report without a sighting keeps the earlier one.
	_, _, err = s.deviceRepository.RecordDeviceCheckin(ctx, device.DeviceID,
		testTenant, &model.DeviceCheckinRecord{
			ReceivedTS: 1792833525.0, Reason: "change", BodyJSON: body,
		}, []model.DeviceCheckinBeacon{
			{Address: "C3:00:00:98:F3:19", Present: false}})
	s.Require().NoError(err)
	band = s.readBeacon(watched)
	s.False(*band.Present)
	s.Equal(1792833460.0, *band.LastSeenTS)
	s.Equal(1792833525.0, *band.LastReportTS)
}

// Every band write moves the device's config version and leaves its
// update_ts; an address the device watches already is a duplicate.
func (s *DeviceMonitoringRepositoryTestSuite) TestBeaconWritesBumpTheDeviceConfig() {
	ctx := context.Background()
	device := s.createDevice("Beacon", testTenant)
	beacon := s.createBeacon(device, "C3:00:00:98:F3:19")
	s.Equal(2, s.readDevice(device).ConfigVersion)

	_, err := s.beaconRepository.CreateBeacon(ctx, &model.BeaconModel{
		DeviceID: device.DeviceID, Address: "C3:00:00:98:F3:19",
		Label: "Again", CreateTS: 1, UpdateTS: 1,
	}, testTenant)
	s.True(errors.Is(err, ErrDuplicate), "duplicate: %v", err)
	s.Equal(2, s.readDevice(device).ConfigVersion, "refused write bumped")

	change := *beacon
	change.Label = "Wrist"
	change.UpdateTS = repoTestTS + 1
	updated, err := s.beaconRepository.UpdateBeacon(ctx, &change, repoTestTS)
	s.Require().NoError(err)
	s.Equal("Wrist", updated.Label)
	s.Equal(3, s.readDevice(device).ConfigVersion)

	s.Require().NoError(s.beaconRepository.DeleteBeacon(ctx,
		beacon.BeaconID, testTenant, repoTestTS+1, repoTestTS+2, 1))
	after := s.readDevice(device)
	s.Equal(4, after.ConfigVersion)
	s.Equal(device.UpdateTS, after.UpdateTS)
}

// A contact starts invited with no invite sent; only an invited contact's
// number can change, and a change owes a new invite.
func (s *DeviceMonitoringRepositoryTestSuite) TestContactNumberRules() {
	ctx := context.Background()
	device := s.createDevice("Contact", testTenant)
	contact := s.createContact(device, "+15551230001")
	s.Equal(model.ContactStatusInvited, contact.ContactStatus)
	s.Nil(contact.InviteSentTS)
	s.Equal([]string{"raised"}, contact.NotifyEventList)

	_, err := s.db.Exec(ctx, `UPDATE contact SET invite_sent_ts = 5
		WHERE contact_id = $1`, contact.ContactID)
	s.Require().NoError(err)

	// nil keeps the list; a new number resets the invite.
	change := *contact
	change.PhoneE164 = "+15551230002"
	change.NotifyEventList = nil
	change.UpdateTS = repoTestTS + 1
	updated, err := s.contactRepository.UpdateContact(ctx, &change, repoTestTS)
	s.Require().NoError(err)
	s.Equal("+15551230002", updated.PhoneE164)
	s.Equal([]string{"raised"}, updated.NotifyEventList)
	s.Nil(updated.InviteSentTS)

	// Once they replied, the number is theirs.
	_, err = s.db.Exec(ctx, `UPDATE contact SET contact_status = 'active'
		WHERE contact_id = $1`, contact.ContactID)
	s.Require().NoError(err)
	change.PhoneE164 = "+15551230003"
	change.UpdateTS = repoTestTS + 2
	_, err = s.contactRepository.UpdateContact(ctx, &change, repoTestTS+1)
	var clientErr *ClientMessageError
	s.Require().True(errors.As(err, &clientErr), "number change: %v", err)
	s.True(errors.Is(err, ErrConflict))
	s.Contains(clientErr.Message, "cannot change")

	// The name can still change; a stale lock is the plain conflict.
	change.PhoneE164 = "+15551230002"
	change.ContactName = "Samantha"
	_, err = s.contactRepository.UpdateContact(ctx, &change, repoTestTS)
	s.True(errors.Is(err, ErrConflict))
	s.False(errors.As(err, &clientErr), "stale is not the number message")
	updated, err = s.contactRepository.UpdateContact(ctx, &change,
		repoTestTS+1)
	s.Require().NoError(err)
	s.Equal("Samantha", updated.ContactName)
	s.Equal(model.ContactStatusActive, updated.ContactStatus)

	// One number per device.
	_, err = s.contactRepository.CreateContact(ctx, &model.ContactModel{
		DeviceID: device.DeviceID, ContactName: "Twin",
		PhoneE164: "+15551230002", NotifyEventList: []string{},
		CreateTS: 1, UpdateTS: 1,
	}, testTenant)
	s.True(errors.Is(err, ErrDuplicate), "duplicate: %v", err)
}

// Acknowledge from raised, turn off from raised or acknowledged; each
// writes its outbox row in the same transaction; anything else is 409.
func (s *DeviceMonitoringRepositoryTestSuite) TestAlertStateRules() {
	ctx := context.Background()
	device := s.createDevice("Alert", testTenant)
	alert := s.createAlert(device, model.AlertStateRaised, nil)

	acked, err := s.alertRepository.AcknowledgeAlert(ctx, alert.AlertID,
		testTenant, repoTestTS, "Alice Smith", repoTestTS+1, 7)
	s.Require().NoError(err)
	s.Equal(model.AlertStateAcknowledged, acked.AlertState)
	s.Equal("Alice Smith", *acked.AckBy)
	s.Equal("ui", *acked.AckVia)
	s.Equal(repoTestTS+1, *acked.AckTS)
	s.Equal(repoTestTS+1, acked.UpdateTS)

	_, err = s.alertRepository.AcknowledgeAlert(ctx, alert.AlertID,
		testTenant, repoTestTS+1, "Alice Smith", repoTestTS+2, 7)
	var clientErr *ClientMessageError
	s.Require().True(errors.As(err, &clientErr), "second ack: %v", err)
	s.True(errors.Is(err, ErrConflict))
	s.Equal("This alert is acknowledged and cannot be acknowledged",
		clientErr.Message)

	_, err = s.alertRepository.TurnOffAlert(ctx, alert.AlertID, testTenant,
		repoTestTS, "Bob", repoTestTS+2, 8)
	s.True(errors.Is(err, ErrConflict), "stale: %v", err)
	s.False(errors.As(err, &clientErr))

	off, err := s.alertRepository.TurnOffAlert(ctx, alert.AlertID,
		testTenant, repoTestTS+1, "Bob", repoTestTS+2, 8)
	s.Require().NoError(err)
	s.Equal(model.AlertStateTurnedOff, off.AlertState)
	s.Equal("Bob", *off.OffBy)
	s.Equal("Alice Smith", *off.AckBy, "the acknowledgement stays")

	rows, err := s.db.Query(ctx, `
		SELECT event_type, actor_name, actor_via, sent_ts IS NULL,
		       create_user_id
		FROM alert_event WHERE alert_id = $1 AND security_context_id = $2
		ORDER BY alert_event_id`, alert.AlertID, testTenant)
	s.Require().NoError(err)
	type event struct {
		eventType, actorName, actorVia string
		unsent                         bool
		createUserID                   int
	}
	var eventList []event
	for rows.Next() {
		var e event
		s.Require().NoError(rows.Scan(&e.eventType, &e.actorName,
			&e.actorVia, &e.unsent, &e.createUserID))
		eventList = append(eventList, e)
	}
	s.Require().NoError(rows.Err())
	s.Equal([]event{
		{"acknowledged", "Alice Smith", "ui", true, 7},
		{"turned_off", "Bob", "ui", true, 8},
	}, eventList, "one outbox row per change, none for a refusal")

	resolvedTS := repoTestTS + 5
	resolved := s.createAlert(device, model.AlertStateResolved, &resolvedTS)
	for _, change := range []func() (*model.AlertModel, error){
		func() (*model.AlertModel, error) {
			return s.alertRepository.AcknowledgeAlert(ctx, resolved.AlertID,
				testTenant, repoTestTS, "Alice", repoTestTS+6, 7)
		},
		func() (*model.AlertModel, error) {
			return s.alertRepository.TurnOffAlert(ctx, resolved.AlertID,
				testTenant, repoTestTS, "Alice", repoTestTS+6, 7)
		},
	} {
		_, err := change()
		s.True(errors.Is(err, ErrConflict), "resolved: %v", err)
		s.Require().True(errors.As(err, &clientErr))
		s.Contains(clientErr.Message, "is resolved")
	}
}

// The list: open first, then resolved within the window; ?deviceId narrows.
func (s *DeviceMonitoringRepositoryTestSuite) TestAlertListOrder() {
	ctx := context.Background()
	device := s.createDevice("AlertList", testTenant)
	other := s.createDevice("AlertListOther", testTenant)
	oldTS, recentTS := repoTestTS-10, repoTestTS+10
	old := s.createAlert(device, model.AlertStateResolved, &oldTS)
	recent := s.createAlert(device, model.AlertStateResolved, &recentTS)
	open := s.createAlert(device, model.AlertStateRaised, nil)
	elsewhere := s.createAlert(other, model.AlertStateRaised, nil)

	alertList, err := s.alertRepository.RetrieveAlertList(
		ctx, device.DeviceID, testTenant, repoTestTS)
	s.Require().NoError(err)
	var idList []int
	for _, alert := range alertList {
		idList = append(idList, alert.AlertID)
	}
	s.Equal([]int{open.AlertID, recent.AlertID}, idList)
	s.NotContains(idList, old.AlertID)
	s.NotContains(idList, elsewhere.AlertID)
}

// clearTenantSetting starts a tenant-setting test with neither test
// tenant holding a row.
func (s *DeviceMonitoringRepositoryTestSuite) clearTenantSetting() {
	_, err := s.db.Exec(context.Background(), `DELETE FROM tenant_setting
		WHERE security_context_id IN (2, 3)`)
	s.Require().NoError(err)
}

func (s *DeviceMonitoringRepositoryTestSuite) TestTenantSettingSave() {
	ctx := context.Background()
	s.clearTenantSetting()
	_, err := s.tenantSettingRepository.RetrieveTenantSetting(ctx, testTenant)
	s.True(errors.Is(err, ErrNotFound), "none yet: %v", err)

	setting := &model.TenantSettingModel{
		TimeZone: "America/New_York", RemindIntervalMin: 15,
		DailyReminderTime: "09:00", SecurityContext: testTenant,
		UpdateTS: repoTestTS, UpdateUserID: 7,
	}
	created, err := s.tenantSettingRepository.SaveTenantSetting(
		ctx, setting, 0)
	s.Require().NoError(err)
	s.Equal(repoTestTS, created.CreateTS)
	s.Equal(7, created.CreateUserID)

	setting.RemindIntervalMin = 30
	setting.UpdateTS = repoTestTS + 1
	_, err = s.tenantSettingRepository.SaveTenantSetting(ctx, setting, 0)
	s.True(errors.Is(err, ErrConflict), "stale: %v", err)
	saved, err := s.tenantSettingRepository.SaveTenantSetting(
		ctx, setting, repoTestTS)
	s.Require().NoError(err)
	s.Equal(30, saved.RemindIntervalMin)
	s.Equal(created.TenantSettingID, saved.TenantSettingID)
}

// --- cross-tenant isolation, one test per table (doc/110) ---

func (s *DeviceMonitoringRepositoryTestSuite) TestDeviceCrossTenantIsolation() {
	ctx := context.Background()
	mine := s.createDevice("Tenant Mine", testTenant)
	theirs := s.createDevice("Tenant Theirs", otherTenant)
	s.Require().NoError(s.deviceRepository.IssueDeviceToken(ctx,
		theirs.DeviceID, otherTenant, repoTestTS, "repotest-theirs",
		repoTestTS+1, 1))
	theirs = s.readDevice(theirs)

	byID, err := s.deviceRepository.RetrieveDeviceList(ctx,
		[]int{mine.DeviceID, theirs.DeviceID}, testTenant, 0)
	s.Require().NoError(err)
	s.Require().Len(byID, 1)
	s.Equal(mine.DeviceID, byID[0].DeviceID)
	full, err := s.deviceRepository.RetrieveDeviceList(
		ctx, nil, testTenant, 0)
	s.Require().NoError(err)
	for _, device := range full {
		s.Equal(testTenant, device.SecurityContext)
	}

	attempt := *theirs
	attempt.SecurityContext = testTenant
	attempt.DeviceName = "RepoTest Hijacked"
	attempt.CheckinIntervalSec = 999
	_, err = s.deviceRepository.UpdateDevice(ctx, &attempt, theirs.UpdateTS)
	s.True(errors.Is(err, ErrNotFound), "update: %v", err)
	err = s.deviceRepository.IssueDeviceToken(ctx, theirs.DeviceID,
		testTenant, theirs.UpdateTS, "repotest-hijack", repoTestTS+2, 1)
	s.True(errors.Is(err, ErrNotFound), "token: %v", err)
	err = s.deviceRepository.DeleteDevice(ctx, theirs.DeviceID, testTenant,
		theirs.UpdateTS, repoTestTS+2, 1)
	s.True(errors.Is(err, ErrNotFound), "delete: %v", err)
	_, _, err = s.deviceRepository.RecordDeviceCheckin(ctx, theirs.DeviceID,
		testTenant, &model.DeviceCheckinRecord{ReceivedTS: 1,
			Reason: "x", BodyJSON: json.RawMessage(`{}`)}, nil)
	s.True(errors.Is(err, ErrNotFound), "check-in: %v", err)

	after := s.readDevice(theirs)
	s.Equal(theirs.DeviceName, after.DeviceName)
	s.Equal(theirs.UpdateTS, after.UpdateTS)
	s.Equal(theirs.ConfigVersion, after.ConfigVersion)
	s.Nil(after.LastCheckinTS)
	found, err := s.deviceRepository.RetrieveDeviceByTokenHash(
		ctx, "repotest-theirs")
	s.Require().NoError(err, "their token still works")
	s.Equal(otherTenant, found.SecurityContext)
}

func (s *DeviceMonitoringRepositoryTestSuite) TestBeaconCrossTenantIsolation() {
	ctx := context.Background()
	mine := s.createDevice("Beacon Mine", testTenant)
	theirDevice := s.createDevice("Beacon Theirs", otherTenant)
	theirs := s.createBeacon(theirDevice, "C3:00:00:98:F3:19")
	theirDevice = s.readDevice(theirDevice)

	list, err := s.beaconRepository.RetrieveBeaconList(ctx,
		[]int{theirs.BeaconID}, 0, testTenant, 0)
	s.Require().NoError(err)
	s.Empty(list)
	list, err = s.beaconRepository.RetrieveBeaconList(ctx, nil,
		theirDevice.DeviceID, testTenant, 0)
	s.Require().NoError(err)
	s.Empty(list)

	// A band cannot be hung on their device, nor moved there.
	_, err = s.beaconRepository.CreateBeacon(ctx, &model.BeaconModel{
		DeviceID: theirDevice.DeviceID, Address: "C3:00:00:98:F3:21",
		Label: "x", CreateTS: 1, UpdateTS: 1,
	}, testTenant)
	s.True(errors.Is(err, ErrNotFound), "create: %v", err)

	attempt := *theirs
	attempt.SecurityContext = testTenant
	attempt.DeviceID = mine.DeviceID
	attempt.Label = "Hijacked"
	_, err = s.beaconRepository.UpdateBeacon(ctx, &attempt, theirs.UpdateTS)
	s.True(errors.Is(err, ErrNotFound), "update: %v", err)
	err = s.beaconRepository.DeleteBeacon(ctx, theirs.BeaconID, testTenant,
		theirs.UpdateTS, repoTestTS+1, 1)
	s.True(errors.Is(err, ErrNotFound), "delete: %v", err)

	after := s.readBeacon(theirs)
	s.Equal("Band", after.Label)
	s.Nil(after.DeleteUserID)
	s.Equal(theirDevice.ConfigVersion,
		s.readDevice(theirDevice).ConfigVersion, "their config bumped")
}

func (s *DeviceMonitoringRepositoryTestSuite) TestContactCrossTenantIsolation() {
	ctx := context.Background()
	theirDevice := s.createDevice("Contact Theirs", otherTenant)
	theirs := s.createContact(theirDevice, "+15551239999")

	list, err := s.contactRepository.RetrieveContactList(ctx,
		[]int{theirs.ContactID}, 0, testTenant, 0)
	s.Require().NoError(err)
	s.Empty(list)
	list, err = s.contactRepository.RetrieveContactList(ctx, nil, 0,
		testTenant, 0)
	s.Require().NoError(err)
	for _, contact := range list {
		s.Equal(testTenant, contact.SecurityContext)
	}

	_, err = s.contactRepository.CreateContact(ctx, &model.ContactModel{
		DeviceID: theirDevice.DeviceID, ContactName: "x",
		PhoneE164: "+15551238888", NotifyEventList: []string{},
		CreateTS: 1, UpdateTS: 1,
	}, testTenant)
	s.True(errors.Is(err, ErrNotFound), "create: %v", err)

	attempt := *theirs
	attempt.SecurityContext = testTenant
	attempt.ContactName = "Hijacked"
	_, err = s.contactRepository.UpdateContact(ctx, &attempt, theirs.UpdateTS)
	s.True(errors.Is(err, ErrNotFound), "update: %v", err)
	err = s.contactRepository.DeleteContact(ctx, theirs.ContactID,
		testTenant, theirs.UpdateTS, repoTestTS+1, 1)
	s.True(errors.Is(err, ErrNotFound), "delete: %v", err)

	list, err = s.contactRepository.RetrieveContactList(ctx,
		[]int{theirs.ContactID}, 0, otherTenant, 0)
	s.Require().NoError(err)
	s.Require().Len(list, 1)
	s.Equal("Sam", list[0].ContactName)
	s.Nil(list[0].DeleteUserID)
}

func (s *DeviceMonitoringRepositoryTestSuite) TestAlertCrossTenantIsolation() {
	ctx := context.Background()
	theirDevice := s.createDevice("Alert Theirs", otherTenant)
	theirs := s.createAlert(theirDevice, model.AlertStateRaised, nil)

	list, err := s.alertRepository.RetrieveAlertList(ctx, 0, testTenant, 0)
	s.Require().NoError(err)
	for _, alert := range list {
		s.NotEqual(theirs.AlertID, alert.AlertID)
	}
	list, err = s.alertRepository.RetrieveAlertList(ctx,
		theirDevice.DeviceID, testTenant, 0)
	s.Require().NoError(err)
	s.Empty(list)

	_, err = s.alertRepository.AcknowledgeAlert(ctx, theirs.AlertID,
		testTenant, repoTestTS, "Mallory", repoTestTS+1, 7)
	s.True(errors.Is(err, ErrNotFound), "acknowledge: %v", err)
	_, err = s.alertRepository.TurnOffAlert(ctx, theirs.AlertID,
		testTenant, repoTestTS, "Mallory", repoTestTS+1, 7)
	s.True(errors.Is(err, ErrNotFound), "turn off: %v", err)

	var state string
	var eventCount int
	s.Require().NoError(s.db.QueryRow(ctx, `
		SELECT alert_state, (SELECT count(*) FROM alert_event
			WHERE alert_id = $1)
		FROM alert WHERE alert_id = $1`, theirs.AlertID).
		Scan(&state, &eventCount))
	s.Equal(model.AlertStateRaised, state)
	s.Zero(eventCount)
}

func (s *DeviceMonitoringRepositoryTestSuite) TestTenantSettingCrossTenantIsolation() {
	ctx := context.Background()
	s.clearTenantSetting()
	theirs, err := s.tenantSettingRepository.SaveTenantSetting(ctx,
		&model.TenantSettingModel{
			TimeZone: "Europe/London", RemindIntervalMin: 5,
			DailyReminderTime: "08:00", SecurityContext: otherTenant,
			UpdateTS: repoTestTS, UpdateUserID: 1,
		}, 0)
	s.Require().NoError(err)

	_, err = s.tenantSettingRepository.RetrieveTenantSetting(ctx, testTenant)
	s.True(errors.Is(err, ErrNotFound), "read: %v", err)

	// Tenant 2's first save is its own row, even with their lock.
	mine, err := s.tenantSettingRepository.SaveTenantSetting(ctx,
		&model.TenantSettingModel{
			TimeZone: "America/Chicago", RemindIntervalMin: 60,
			DailyReminderTime: "10:00", SecurityContext: testTenant,
			UpdateTS: repoTestTS + 1, UpdateUserID: 7,
		}, theirs.UpdateTS)
	s.Require().NoError(err)
	s.NotEqual(theirs.TenantSettingID, mine.TenantSettingID)

	after, err := s.tenantSettingRepository.RetrieveTenantSetting(
		ctx, otherTenant)
	s.Require().NoError(err)
	s.Equal("Europe/London", after.TimeZone)
	s.Equal(theirs.UpdateTS, after.UpdateTS)
}
