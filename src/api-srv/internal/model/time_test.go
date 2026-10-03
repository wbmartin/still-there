package model

import (
	"testing"
	"time"
)

// EpochSeconds is the one timestamp formula, so the two properties callers
// rely on are pinned here: a whole second converts exactly, and sub-second
// detail survives (optimistic locking on update_ts depends on it).
func TestEpochSecondsIsExactForAWholeSecond(t *testing.T) {
	whole := time.Date(2026, 9, 22, 13, 45, 6, 0, time.UTC)

	got := EpochSeconds(whole)
	if want := float64(whole.Unix()); got != want {
		t.Fatalf("EpochSeconds(%v) = %v, want exactly %v", whole, got, want)
	}
}

func TestEpochSecondsKeepsSubSecondDetail(t *testing.T) {
	base := time.Date(2026, 9, 22, 13, 45, 6, 0, time.UTC)

	if EpochSeconds(base.Add(time.Millisecond)) <= EpochSeconds(base) {
		t.Fatal("a millisecond later did not compare greater")
	}
	if EpochSeconds(base.Add(time.Microsecond)) <= EpochSeconds(base) {
		t.Fatal("a microsecond later did not compare greater")
	}
}

func TestNowMovesForward(t *testing.T) {
	first := Now()
	if first <= 0 {
		t.Fatalf("Now() = %v, want a positive epoch", first)
	}

	time.Sleep(2 * time.Millisecond)
	if second := Now(); second <= first {
		t.Fatalf("Now() = %v then %v, want the second to be later",
			first, second)
	}
}
