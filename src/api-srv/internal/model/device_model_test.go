package model

import "testing"

func TestNormaliseBeaconAddress(t *testing.T) {
	for address, want := range map[string]string{
		"c3:00:00:98:f3:19":   "C3:00:00:98:F3:19",
		" C3:00:00:98:F3:19 ": "C3:00:00:98:F3:19",
		"c3-00-00-98-f3-19":   "",
		"C3:00:00:98:F3":      "",
		"C3:00:00:98:F3:1G":   "",
		"":                    "",
	} {
		got, ok := NormaliseBeaconAddress(address)
		if want == "" && ok {
			t.Errorf("%q accepted as %q", address, got)
		}
		if want != "" && (!ok || got != want) {
			t.Errorf("%q: got %q ok=%v, want %q", address, got, ok, want)
		}
	}
}

func TestDeviceProblem(t *testing.T) {
	good := DeviceModel{DeviceName: "Pi", CheckinIntervalSec: 10,
		MissedCheckinBeforeAlarm: 1, AbsentAfterSec: 10}
	if problem := DeviceProblem(&good); problem != "" {
		t.Fatalf("the floor values refused: %s", problem)
	}
	for name, change := range map[string]func(*DeviceModel){
		"blank name":  func(d *DeviceModel) { d.DeviceName = "  " },
		"interval 9":  func(d *DeviceModel) { d.CheckinIntervalSec = 9 },
		"missed 0":    func(d *DeviceModel) { d.MissedCheckinBeforeAlarm = 0 },
		"absent 9":    func(d *DeviceModel) { d.AbsentAfterSec = 9 },
		"interval -1": func(d *DeviceModel) { d.CheckinIntervalSec = -1 },
	} {
		device := good
		change(&device)
		if DeviceProblem(&device) == "" {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestContactProblem(t *testing.T) {
	good := ContactModel{ContactName: "Sam", PhoneE164: " +15551234567 "}
	if problem := ContactProblem(&good); problem != "" {
		t.Fatalf("a good contact refused: %s", problem)
	}
	if good.PhoneE164 != "+15551234567" {
		t.Errorf("phone not trimmed: %q", good.PhoneE164)
	}
	good.NotifyEventList = append([]string{}, ContactNotifyEventList...)
	if problem := ContactProblem(&good); problem != "" {
		t.Fatalf("every event refused: %s", problem)
	}
	for name, change := range map[string]func(*ContactModel){
		"blank name":    func(c *ContactModel) { c.ContactName = "" },
		"no plus":       func(c *ContactModel) { c.PhoneE164 = "15551234567" },
		"leading zero":  func(c *ContactModel) { c.PhoneE164 = "+05551234567" },
		"too short":     func(c *ContactModel) { c.PhoneE164 = "+1234567" },
		"too long":      func(c *ContactModel) { c.PhoneE164 = "+1234567890123456" },
		"dashes":        func(c *ContactModel) { c.PhoneE164 = "+1-555-123-4567" },
		"unknown event": func(c *ContactModel) { c.NotifyEventList = []string{"raised", "sms"} },
		"status case":   func(c *ContactModel) { c.NotifyEventList = []string{"Raised"} },
	} {
		contact := good
		change(&contact)
		if ContactProblem(&contact) == "" {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestBeaconProblem(t *testing.T) {
	beacon := BeaconModel{Address: "c3:00:00:98:f3:19", Label: "Band"}
	if problem := BeaconProblem(&beacon); problem != "" {
		t.Fatalf("a good beacon refused: %s", problem)
	}
	if beacon.Address != "C3:00:00:98:F3:19" {
		t.Errorf("address not normalised: %q", beacon.Address)
	}
	for _, bad := range []BeaconModel{
		{Address: "C3:00:00:98:F3:19", Label: " "},
		{Address: "not-a-mac", Label: "Band"},
	} {
		if BeaconProblem(&bad) == "" {
			t.Errorf("%+v accepted", bad)
		}
	}
}

func TestTenantSettingProblem(t *testing.T) {
	good := TenantSettingModel{TimeZone: "America/New_York",
		RemindIntervalMin: 1, DailyReminderTime: "00:00"}
	if problem := TenantSettingProblem(&good); problem != "" {
		t.Fatalf("a good setting refused: %s", problem)
	}
	for name, change := range map[string]func(*TenantSettingModel){
		"empty zone":   func(s *TenantSettingModel) { s.TimeZone = "" },
		"Local zone":   func(s *TenantSettingModel) { s.TimeZone = "Local" },
		"unknown zone": func(s *TenantSettingModel) { s.TimeZone = "Mars/Base" },
		"interval 0":   func(s *TenantSettingModel) { s.RemindIntervalMin = 0 },
		"interval max": func(s *TenantSettingModel) { s.RemindIntervalMin = 1441 },
		"24:00":        func(s *TenantSettingModel) { s.DailyReminderTime = "24:00" },
		"7:30":         func(s *TenantSettingModel) { s.DailyReminderTime = "7:30" },
		"07:60":        func(s *TenantSettingModel) { s.DailyReminderTime = "07:60" },
	} {
		setting := good
		change(&setting)
		if TenantSettingProblem(&setting) == "" {
			t.Errorf("%s accepted", name)
		}
	}
	good.RemindIntervalMin, good.DailyReminderTime = 1440, "23:59"
	if problem := TenantSettingProblem(&good); problem != "" {
		t.Errorf("the top values refused: %s", problem)
	}
}
