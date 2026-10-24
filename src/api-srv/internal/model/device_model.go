package model

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// Still There monitoring (migration 0008, tasks/todo.md Server): the Pi
// base station in a home (device) and the BLE bands it watches (beacon).
// The runtime columns (device.last_*, token_*, beacon.present / last_*)
// are written by check-ins without touching update_ts and are read-only
// through the CRUD routes. token_hash never leaves the repository.

// The lowest values a device's alarm settings take.
const (
	DeviceMinCheckinIntervalSec       = 10
	DeviceMinMissedCheckinBeforeAlarm = 1
	DeviceMinAbsentAfterSec           = 10
)

// DeviceCheckinMaxBodyBytes caps POST /deviceCheckin: a Pi reports a
// handful of bands in well under 1 KiB.
const DeviceCheckinMaxBodyBytes = 16 << 10

// DeviceModel is one device row as the API shows it.
type DeviceModel struct {
	DeviceID                 int      `json:"deviceId"`
	DeviceName               string   `json:"deviceName"`
	CheckinIntervalSec       int      `json:"checkinIntervalSec"`
	MissedCheckinBeforeAlarm int      `json:"missedCheckinBeforeAlarm"`
	AbsentAfterSec           int      `json:"absentAfterSec"`
	ConfigVersion            int      `json:"configVersion"`
	TokenIssuedTS            *float64 `json:"tokenIssuedTs,omitempty"`
	LastCheckinTS            *float64 `json:"lastCheckinTs,omitempty"`
	// The last check-in body as the Pi sent it.
	LastCheckinJSON  json.RawMessage `json:"lastCheckinJson,omitempty"`
	LastAgentVersion *string         `json:"lastAgentVersion,omitempty"`
	SecurityContext  int             `json:"securityContext"`
	CreateTS         float64         `json:"createTs"`
	CreateUserID     int             `json:"createUserId"`
	UpdateTS         float64         `json:"updateTs"`
	UpdateUserID     int             `json:"updateUserId"`
	DeleteTS         *float64        `json:"deleteTs,omitempty"`
	DeleteUserID     *int            `json:"deleteUserId,omitempty"`
}

// DeviceProblem is why a device's writable fields cannot be saved, "" when
// they can.
func DeviceProblem(device *DeviceModel) string {
	switch {
	case strings.TrimSpace(device.DeviceName) == "":
		return "deviceName is required"
	case device.CheckinIntervalSec < DeviceMinCheckinIntervalSec:
		return "checkinIntervalSec must be at least 10"
	case device.MissedCheckinBeforeAlarm < DeviceMinMissedCheckinBeforeAlarm:
		return "missedCheckinBeforeAlarm must be at least 1"
	case device.AbsentAfterSec < DeviceMinAbsentAfterSec:
		return "absentAfterSec must be at least 10"
	}
	return ""
}

// DeviceTokenRequest is the body of POST /deviceToken.
type DeviceTokenRequest struct {
	DeviceID int     `json:"deviceId"`
	UpdateTS float64 `json:"updateTs"`
}

// DeviceTokenIssuedModel answers POST /deviceToken. DeviceToken is the
// only copy of the token anywhere: it goes into the Pi's config file and
// is never shown again.
type DeviceTokenIssuedModel struct {
	DeviceID      int     `json:"deviceId"`
	DeviceToken   string  `json:"deviceToken"`
	TokenIssuedTS float64 `json:"tokenIssuedTs"`
	UpdateTS      float64 `json:"updateTs"`
}

// BeaconModel is one band a device watches.
type BeaconModel struct {
	BeaconID int `json:"beaconId"`
	DeviceID int `json:"deviceId"`
	// Upper-case MAC, AA:BB:CC:DD:EE:FF (NormaliseBeaconAddress).
	Address string `json:"address"`
	Label   string `json:"label"`
	// The Pi's last report; absent before any.
	Present         *bool    `json:"present,omitempty"`
	LastSeenTS      *float64 `json:"lastSeenTs,omitempty"`
	LastRSSI        *int     `json:"lastRssi,omitempty"`
	LastReportTS    *float64 `json:"lastReportTs,omitempty"`
	SecurityContext int      `json:"securityContext"`
	CreateTS        float64  `json:"createTs"`
	CreateUserID    int      `json:"createUserId"`
	UpdateTS        float64  `json:"updateTs"`
	UpdateUserID    int      `json:"updateUserId"`
	DeleteTS        *float64 `json:"deleteTs,omitempty"`
	DeleteUserID    *int     `json:"deleteUserId,omitempty"`
}

var beaconAddressPattern = regexp.MustCompile(
	`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)

// NormaliseBeaconAddress is address trimmed and upper-cased, and whether
// that is a MAC address (AA:BB:CC:DD:EE:FF), the form the Pi reports.
func NormaliseBeaconAddress(address string) (string, bool) {
	normal := strings.ToUpper(strings.TrimSpace(address))
	return normal, beaconAddressPattern.MatchString(normal)
}

// BeaconProblem normalises the beacon's address in place and says why the
// beacon cannot be saved, "" when it can.
func BeaconProblem(beacon *BeaconModel) string {
	address, ok := NormaliseBeaconAddress(beacon.Address)
	beacon.Address = address
	switch {
	case !ok:
		return "address must be a MAC address, AA:BB:CC:DD:EE:FF"
	case strings.TrimSpace(beacon.Label) == "":
		return "label is required"
	}
	return ""
}

// DeviceCheckinRequest is the body src/pi-agent/checkin.go sends.
type DeviceCheckinRequest struct {
	AgentVersion  string                `json:"agentVersion"`
	SentAt        *time.Time            `json:"sentAt"`
	UptimeSec     int64                 `json:"uptimeSec"`
	ConfigVersion int                   `json:"configVersion"`
	Reason        string                `json:"reason"`
	BeaconList    []DeviceCheckinBeacon `json:"beaconList"`
}

// DeviceCheckinBeacon is one band in a check-in, as the Pi sees it.
type DeviceCheckinBeacon struct {
	Address  string     `json:"address"`
	Label    string     `json:"label"`
	Present  bool       `json:"present"`
	LastSeen *time.Time `json:"lastSeen"`
	RSSI     *int       `json:"rssi"`
}

// DeviceCheckinRecord is what one check-in writes: the device_checkin row
// and the device's last_* columns.
type DeviceCheckinRecord struct {
	ReceivedTS   float64
	SentTS       *float64
	Reason       string
	BodyJSON     json.RawMessage
	SourceIP     string
	AgentVersion string
}

// DeviceCheckinReply answers POST /deviceCheckin.
type DeviceCheckinReply struct {
	ConfigVersion int `json:"configVersion"`
}
