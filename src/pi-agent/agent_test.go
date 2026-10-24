package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrackerPresence(t *testing.T) {
	tr := NewTracker([]Beacon{{Address: "c3:00:00:98:f3:19", Label: "band"}})
	t0 := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

	state := tr.Snapshot(t0, time.Minute)[0]
	if state.Present || state.LastSeen != nil || state.RSSI != nil {
		t.Fatalf("never seen: got %+v", state)
	}

	if tr.Observe("AA:BB:CC:DD:EE:FF", -40, t0) {
		t.Fatal("unwatched address reported as watched")
	}
	if !tr.Observe("C3:00:00:98:F3:19", -50, t0) {
		t.Fatal("watched address not matched case-insensitively")
	}
	tr.Observe("C3:00:00:98:F3:19", -60, t0.Add(time.Second))

	state = tr.Snapshot(t0.Add(time.Minute), time.Minute)[0]
	if !state.Present {
		t.Fatalf("seen 59s ago with absentAfter 60s: want present, got %+v", state)
	}
	if *state.RSSI != -53 { // -50 + 0.3*(-60 - -50)
		t.Fatalf("smoothed rssi: want -53, got %d", *state.RSSI)
	}
	state = tr.Snapshot(t0.Add(time.Minute+2*time.Second), time.Minute)[0]
	if state.Present {
		t.Fatalf("seen 61s ago: want absent, got %+v", state)
	}
}

func TestTrackerSetBeaconsKeepsSightings(t *testing.T) {
	tr := NewTracker([]Beacon{{Address: "AA:00:00:00:00:01"}})
	now := time.Now()
	tr.Observe("AA:00:00:00:00:01", -40, now)
	tr.SetBeacons([]Beacon{{Address: "AA:00:00:00:00:01", Label: "renamed"}, {Address: "AA:00:00:00:00:02"}})
	stateList := tr.Snapshot(now, time.Minute)
	if len(stateList) != 2 {
		t.Fatalf("want 2 beacons, got %d", len(stateList))
	}
	for _, s := range stateList {
		if s.Address == "AA:00:00:00:00:01" && (!s.Present || s.Label != "renamed") {
			t.Fatalf("kept beacon lost its sighting or label: %+v", s)
		}
	}
}

func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	cfg := Config{
		ServerURL:          "https://example.test",
		DeviceToken:        "tok",
		ConfigVersion:      3,
		CheckinIntervalSec: 60,
		AbsentAfterSec:     600,
		BeaconList:         []Beacon{{Address: "c3:00:00:98:f3:19", Label: "band"}},
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config holds the device token: want mode 0600, got %v", info.Mode().Perm())
	}
	got, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.BeaconList[0].Address != "C3:00:00:98:F3:19" || got.ConfigVersion != 3 {
		t.Fatalf("round trip: got %+v", got)
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatal("empty config passed validation")
	}
}
