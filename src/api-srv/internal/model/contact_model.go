package model

import (
	"regexp"
	"strings"
)

// The contact.contact_status values (migration 0008). A contact is created
// invited; only their own replies by text move it (the watchdog's job), so
// the API never writes these but invited.
const (
	ContactStatusInvited  = "invited"
	ContactStatusActive   = "active"
	ContactStatusOptedOut = "opted_out"
)

// ContactNotifyEventList is every event a contact can be texted about, the
// notify_event_list a create without one gets.
var ContactNotifyEventList = []string{
	"raised", "reminder", "acknowledged", "turned_off", "reactivated",
	"daily_reminder",
}

var contactPhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// ContactModel is one person texted about one device. contact_status,
// invite_sent_ts, opted_in_ts and opted_out_ts are the watchdog's: the
// API reads them and never takes them from a client.
type ContactModel struct {
	ContactID       int      `json:"contactId"`
	DeviceID        int      `json:"deviceId"`
	ContactName     string   `json:"contactName"`
	PhoneE164       string   `json:"phoneE164"`
	ContactStatus   string   `json:"contactStatus"`
	NotifyEventList []string `json:"notifyEventList"`
	InviteSentTS    *float64 `json:"inviteSentTs,omitempty"`
	OptedInTS       *float64 `json:"optedInTs,omitempty"`
	OptedOutTS      *float64 `json:"optedOutTs,omitempty"`
	SecurityContext int      `json:"securityContext"`
	CreateTS        float64  `json:"createTs"`
	CreateUserID    int      `json:"createUserId"`
	UpdateTS        float64  `json:"updateTs"`
	UpdateUserID    int      `json:"updateUserId"`
	DeleteTS        *float64 `json:"deleteTs,omitempty"`
	DeleteUserID    *int     `json:"deleteUserId,omitempty"`
}

// ContactProblem says why the contact's writable fields cannot be saved,
// "" when they can. A nil notifyEventList passes: create fills the
// default, update keeps the stored list.
func ContactProblem(contact *ContactModel) string {
	contact.PhoneE164 = strings.TrimSpace(contact.PhoneE164)
	switch {
	case strings.TrimSpace(contact.ContactName) == "":
		return "contactName is required"
	case !contactPhonePattern.MatchString(contact.PhoneE164):
		return "phoneE164 must be an E.164 number, + and 8 to 15 digits"
	}
	for _, event := range contact.NotifyEventList {
		if !isContactNotifyEvent(event) {
			return "notifyEventList takes only " +
				strings.Join(ContactNotifyEventList, ", ")
		}
	}
	return ""
}

func isContactNotifyEvent(event string) bool {
	for _, known := range ContactNotifyEventList {
		if event == known {
			return true
		}
	}
	return false
}
