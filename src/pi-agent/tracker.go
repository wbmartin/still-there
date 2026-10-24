package main

import (
	"math"
	"strings"
	"sync"
	"time"
)

// rssiSmoothing is the weight of a new reading in the moving average; RSSI
// jumps several dB between packets from a band that has not moved.
const rssiSmoothing = 0.3

// Tracker holds the last sighting of each configured beacon. The scanner
// calls Observe from its goroutine; the check-in loop reads Snapshot.
type Tracker struct {
	mu       sync.Mutex
	stateMap map[string]*beaconSighting
}

type beaconSighting struct {
	label    string
	lastSeen time.Time
	rssi     float64
}

type BeaconState struct {
	Address  string     `json:"address"`
	Label    string     `json:"label"`
	Present  bool       `json:"present"`
	LastSeen *time.Time `json:"lastSeen"`
	RSSI     *int       `json:"rssi"`
}

func NewTracker(beaconList []Beacon) *Tracker {
	t := &Tracker{stateMap: map[string]*beaconSighting{}}
	t.SetBeacons(beaconList)
	return t
}

// SetBeacons replaces the watched list, keeping sightings of beacons that
// stay on it.
func (t *Tracker) SetBeacons(beaconList []Beacon) {
	t.mu.Lock()
	defer t.mu.Unlock()
	next := map[string]*beaconSighting{}
	for _, b := range beaconList {
		address := strings.ToUpper(b.Address)
		s, ok := t.stateMap[address]
		if !ok {
			s = &beaconSighting{}
		}
		s.label = b.Label
		next[address] = s
	}
	t.stateMap = next
}

// Observe records one sighting; it reports whether the address is watched.
func (t *Tracker) Observe(address string, rssi int, at time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.stateMap[strings.ToUpper(address)]
	if !ok {
		return false
	}
	if s.lastSeen.IsZero() {
		s.rssi = float64(rssi)
	} else {
		s.rssi += rssiSmoothing * (float64(rssi) - s.rssi)
	}
	s.lastSeen = at
	return true
}

// Snapshot returns every watched beacon; a beacon is present when it was
// seen within absentAfter of now.
func (t *Tracker) Snapshot(now time.Time, absentAfter time.Duration) []BeaconState {
	t.mu.Lock()
	defer t.mu.Unlock()
	stateList := make([]BeaconState, 0, len(t.stateMap))
	for address, s := range t.stateMap {
		state := BeaconState{Address: address, Label: s.label}
		if !s.lastSeen.IsZero() {
			lastSeen := s.lastSeen
			rssi := int(math.Round(s.rssi))
			state.LastSeen = &lastSeen
			state.RSSI = &rssi
			state.Present = now.Sub(lastSeen) <= absentAfter
		}
		stateList = append(stateList, state)
	}
	return stateList
}
