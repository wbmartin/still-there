package model

import (
	"regexp"
	"time"
)

// TenantSettingModel is the one row per tenant that says when and how
// often reminders go out (migration 0008).
type TenantSettingModel struct {
	TenantSettingID int `json:"tenantSettingId"`
	// An IANA name, as time.LoadLocation takes it.
	TimeZone          string `json:"timeZone"`
	RemindIntervalMin int    `json:"remindIntervalMin"`
	// HH:MM in TimeZone.
	DailyReminderTime string   `json:"dailyReminderTime"`
	SecurityContext   int      `json:"securityContext"`
	CreateTS          float64  `json:"createTs"`
	CreateUserID      int      `json:"createUserId"`
	UpdateTS          float64  `json:"updateTs"`
	UpdateUserID      int      `json:"updateUserId"`
	DeleteTS          *float64 `json:"deleteTs,omitempty"`
	DeleteUserID      *int     `json:"deleteUserId,omitempty"`
}

var dailyReminderTimePattern = regexp.MustCompile(
	`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// TenantSettingProblem says why the setting cannot be saved, "" when it
// can.
func TenantSettingProblem(setting *TenantSettingModel) string {
	// LoadLocation takes "" and "UTC" as UTC and "Local" as the server's
	// zone; only a real name is stored.
	if _, err := time.LoadLocation(setting.TimeZone); err != nil ||
		setting.TimeZone == "" || setting.TimeZone == "Local" {
		return "timeZone must be an IANA time zone name"
	}
	if setting.RemindIntervalMin < 1 || setting.RemindIntervalMin > 1440 {
		return "remindIntervalMin must be from 1 to 1440"
	}
	if !dailyReminderTimePattern.MatchString(setting.DailyReminderTime) {
		return "dailyReminderTime must be HH:MM"
	}
	return ""
}
