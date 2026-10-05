package security

import (
	"testing"
	"time"
)

// TestRateWindowCatchesTheMinuteBoundaryBurst: a full allowance spent at
// 12:00:59 still counts at 12:01:00, so a second full allowance there is
// over the limit -- the fixed per-minute buckets let it through. A minute
// later the old bucket has slid out.
func TestRateWindowCatchesTheMinuteBoundaryBurst(t *testing.T) {
	const limit = 20
	noon := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

	atEndOfMinute := newRateWindow(
		"ip:198.51.100.7", noon.Add(59*time.Second),
	)
	atNextMinute := newRateWindow(
		"ip:198.51.100.7", noon.Add(60*time.Second),
	)
	if atNextMinute.previousKey != atEndOfMinute.currentKey {
		t.Fatalf("12:01's previous bucket %q is not 12:00's %q",
			atNextMinute.previousKey, atEndOfMinute.currentKey)
	}

	count, err := atNextMinute.count(limit, "20")
	if err != nil {
		t.Fatal(err)
	}
	if count <= limit {
		t.Errorf("20 at 12:00:59 + 20 at 12:01:00 counts %d, want over %d",
			count, limit)
	}

	lateInMinute := newRateWindow(
		"ip:198.51.100.7", noon.Add(119*time.Second),
	)
	count, err = lateInMinute.count(1, "20")
	if err != nil {
		t.Fatal(err)
	}
	if count > 2 {
		t.Errorf("at 12:01:59 the 12:00 bucket weighs %d, want almost "+
			"nothing", count-1)
	}

	if count, _ = atNextMinute.count(3, ""); count != 3 {
		t.Errorf("an absent previous bucket counts %d, want 0", count-3)
	}
	if _, err = atNextMinute.count(3, "not-a-count"); err == nil {
		t.Error("an unreadable previous bucket must fail closed")
	}
}

// TestRateWindowKeysOnAbsoluteMinute: the same minute of different hours
// is a different bucket, so an hour-old bucket is never counted again.
func TestRateWindowKeysOnAbsoluteMinute(t *testing.T) {
	noon := time.Date(2026, 10, 2, 12, 5, 0, 0, time.UTC)
	anHourLater := newRateWindow("alice", noon.Add(time.Hour))
	if anHourLater.currentKey == newRateWindow("alice", noon).currentKey {
		t.Errorf("12:05 and 13:05 share bucket %q", anHourLater.currentKey)
	}
}
