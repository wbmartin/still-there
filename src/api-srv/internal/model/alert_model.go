package model

// Alarms (migration 0008). The watchdog raises and resolves them; through
// the API a person acknowledges one or turns it off, and each of those
// writes an alert_event row (the outbox) for the watchdog to text.

// The alert.alert_state values.
const (
	AlertStateRaised       = "raised"
	AlertStateAcknowledged = "acknowledged"
	AlertStateTurnedOff    = "turned_off"
	AlertStateResolved     = "resolved"
)

// AlertViaUI is ack_via / off_via / actor_via for a person in the web app.
const AlertViaUI = "ui"

// AlertRecentResolvedSec is how far back GET /alertList shows resolved
// alarms: a week.
const AlertRecentResolvedSec = 7 * 86400

// AlertModel is one alarm. beacon_id is absent for device_offline.
type AlertModel struct {
	AlertID         int      `json:"alertId"`
	DeviceID        int      `json:"deviceId"`
	BeaconID        *int     `json:"beaconId,omitempty"`
	AlertKind       string   `json:"alertKind"`
	AlertState      string   `json:"alertState"`
	RaisedTS        float64  `json:"raisedTs"`
	LastNotifiedTS  *float64 `json:"lastNotifiedTs,omitempty"`
	NotifyCount     int      `json:"notifyCount"`
	AckTS           *float64 `json:"ackTs,omitempty"`
	AckBy           *string  `json:"ackBy,omitempty"`
	AckVia          *string  `json:"ackVia,omitempty"`
	OffTS           *float64 `json:"offTs,omitempty"`
	OffBy           *string  `json:"offBy,omitempty"`
	OffVia          *string  `json:"offVia,omitempty"`
	ResolvedTS      *float64 `json:"resolvedTs,omitempty"`
	SecurityContext int      `json:"securityContext"`
	CreateTS        float64  `json:"createTs"`
	CreateUserID    int      `json:"createUserId"`
	UpdateTS        float64  `json:"updateTs"`
	UpdateUserID    int      `json:"updateUserId"`
}

// AlertActionRequest is the body of POST /alertAcknowledge and
// /alertTurnOff.
type AlertActionRequest struct {
	AlertID  int     `json:"alertId"`
	UpdateTS float64 `json:"updateTs"`
}
