package app_lib

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

// newTestKV builds a KV with a manually advanced clock. Lets tests
// exercise expiration without real sleeps.
func newTestKV() (*KV, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := &now
	kv := &KV{
		m:    make(map[string]memStoredValue),
		done: make(chan struct{}),
		now:  func() time.Time { return *clock },
	}
	// no cleanup goroutine — tests trigger it explicitly if needed
	return kv, clock
}

func TestKV_GetMissingReturnsEmpty(t *testing.T) {
	kv, _ := newTestKV()
	val, err := kv.Get("nope")
	if err != nil {
		t.Fatalf("expected nil error for missing key, got %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string for missing key, got %q", val)
	}
}

func TestKV_SetGet(t *testing.T) {
	kv, _ := newTestKV()
	if err := kv.Set("foo", "bar", ""); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, err := kv.Get("foo")
	if err != nil || val != "bar" {
		t.Fatalf("Get returned (%q, %v), want (\"bar\", nil)", val, err)
	}
}

func TestKV_SetWithTTLExpires(t *testing.T) {
	kv, clock := newTestKV()
	if err := kv.Set("foo", "bar", "10min"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Just before expiry.
	*clock = clock.Add(9 * time.Minute)
	val, _ := kv.Get("foo")
	if val != "bar" {
		t.Fatalf("expected value to still exist before expiry, got %q", val)
	}
	// After expiry.
	*clock = clock.Add(2 * time.Minute)
	val, err := kv.Get("foo")
	if err != nil {
		t.Fatalf("Get after expiry: unexpected err %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty after expiry, got %q", val)
	}
}

func TestKV_IncrInitialAndIncrementPreserveTTL(t *testing.T) {
	kv, clock := newTestKV()

	// First INCR creates key with TTL.
	v1, err := kv.Incr("counter", "1min")
	if err != nil || v1 != 1 {
		t.Fatalf("first Incr = (%d, %v), want (1, nil)", v1, err)
	}

	// Advance 30s and INCR again — should NOT reset the TTL.
	*clock = clock.Add(30 * time.Second)
	v2, _ := kv.Incr("counter", "1min")
	if v2 != 2 {
		t.Fatalf("second Incr = %d, want 2", v2)
	}

	// Advance another 31s. Original TTL was 1 minute from t0, so we are
	// now 1s past expiry. The next Incr must observe the key as expired
	// and reset to 1.
	*clock = clock.Add(31 * time.Second)
	v3, _ := kv.Incr("counter", "1min")
	if v3 != 1 {
		t.Fatalf("Incr after original TTL expiry = %d, want reset to 1", v3)
	}
}

func TestKV_IncrOnKindStrResetsButPreservesTTL(t *testing.T) {
	kv, clock := newTestKV()

	// Set as a non-numeric string with a TTL.
	if err := kv.Set("k", "notanumber", "1h"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	originalExpiry := clock.Add(time.Hour)

	v, err := kv.Incr("k", "10min") // ttl arg should be ignored (exists)
	if err != nil {
		t.Fatalf("Incr: %v", err)
	}
	if v != 1 {
		t.Fatalf("Incr on kindStr = %d, want reset to 1", v)
	}

	// TTL should still match the original Set, not the new arg.
	sv := kv.m["k"]
	if !sv.expiresAt.Equal(originalExpiry) {
		t.Fatalf(
			"expiresAt = %v, want preserved original %v",
			sv.expiresAt, originalExpiry,
		)
	}
}

func TestKV_ConcurrentIncrIsAtomic(t *testing.T) {
	kv := NewKV()
	defer kv.Close()

	const goroutines = 50
	const incsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range incsPerGoroutine {
				if _, err := kv.Incr("hot", ""); err != nil {
					t.Errorf("Incr: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	val, err := kv.Get("hot")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := strconv.Atoi(val)
	want := goroutines * incsPerGoroutine
	if got != want {
		t.Fatalf("counter = %d, want %d", got, want)
	}
}

func TestKV_CloseReturnsErrOnSubsequentOps(t *testing.T) {
	kv := NewKV()
	kv.Close()

	if _, err := kv.Get("any"); !errors.Is(err, errKVClosed) {
		t.Fatalf("Get after Close: err=%v, want errKVClosed", err)
	}
	if err := kv.Set("a", "b", ""); !errors.Is(err, errKVClosed) {
		t.Fatalf("Set after Close: err=%v, want errKVClosed", err)
	}
	if _, err := kv.Incr("a", ""); !errors.Is(err, errKVClosed) {
		t.Fatalf("Incr after Close: err=%v, want errKVClosed", err)
	}

	// Calling Close twice must not panic.
	kv.Close()
}

func TestKV_SetDetectsIntKind(t *testing.T) {
	kv, _ := newTestKV()
	_ = kv.Set("k", "42", "")
	sv := kv.m["k"]
	if sv.kind != memKindInt || sv.intVal != 42 {
		t.Fatalf("expected kindInt=42, got kind=%d intVal=%d strVal=%q",
			sv.kind, sv.intVal, sv.strVal)
	}
}

func TestKV_ParseExpirationGrammar(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, true},
		{"30s", 30 * time.Second, true},
		{"10min", 10 * time.Minute, true},
		{"2h", 2 * time.Hour, true},
		{"5", 0, false},
		{"abc", 0, false},
		{"1d", 0, false},
	}
	for _, c := range cases {
		got, ok := memParseExpiration(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf(
				"memParseExpiration(%q) = (%v, %v), want (%v, %v)",
				c.in, got, ok, c.want, c.ok,
			)
		}
	}
}

func TestKV_KeyLengthLimit(t *testing.T) {
	kv, _ := newTestKV()
	longKey := make([]byte, memKVMaxKeyLen+1)
	for i := range longKey {
		longKey[i] = 'a'
	}
	if _, err := kv.Get(string(longKey)); err == nil {
		t.Fatalf("expected key-length error on Get, got nil")
	}
	if err := kv.Set(string(longKey), "v", ""); err == nil {
		t.Fatalf("expected key-length error on Set, got nil")
	}
}
