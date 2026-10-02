package cache

// In-process key-value store: rate-limit counters, login lockouts, MFA temp
// tokens and user blocks. PROCESS-LOCAL — restarting the API server clears
// it, which is acceptable for one app server.
//
// Graduation trigger: the day this app runs behind a load balancer with two
// or more API server processes, this needs a shared store (Redis or the
// like). Otherwise rate limits and lockouts become per-process and an
// attacker can spread attempts to evade them; MFA temp tokens issued by one
// server won't validate on another. Until then there is deliberately no
// pluggable backend.
//
// TTL grammar is Ns, Nmin, Nh. INCR honours the ttl argument only when it
// creates the key, so repeated INCRs cannot extend a key's lifetime.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	memKVMaxKeyLen    = 100
	memKVMaxValueLen  = 100
	memKVCleanupEvery = 60 * time.Second
)

// errKVClosed is returned by all operations after Close has been called.
var errKVClosed = errors.New("kv closed")

type memValueKind int

const (
	memKindInt memValueKind = iota
	memKindStr
)

type memStoredValue struct {
	kind      memValueKind
	intVal    int64
	strVal    string
	expiresAt time.Time // zero = no expiration
}

func (sv memStoredValue) expired(now time.Time) bool {
	return !sv.expiresAt.IsZero() && !sv.expiresAt.After(now)
}

func (sv memStoredValue) render() string {
	if sv.kind == memKindInt {
		return strconv.FormatInt(sv.intVal, 10)
	}
	return sv.strVal
}

// memParseExpiration accepts the expiration grammar: Ns, Nmin, Nh.
// An empty string means "no expiration" and is reported as (0, true) so the
// caller can distinguish it from a parse error.
func memParseExpiration(s string) (time.Duration, bool) {
	if s == "" {
		return 0, true
	}
	switch {
	case strings.HasSuffix(s, "min"):
		n, err := strconv.ParseUint(strings.TrimSuffix(s, "min"), 10, 64)
		if err != nil {
			return 0, false
		}
		return time.Duration(n) * time.Minute, true
	case strings.HasSuffix(s, "s"):
		n, err := strconv.ParseUint(strings.TrimSuffix(s, "s"), 10, 64)
		if err != nil {
			return 0, false
		}
		return time.Duration(n) * time.Second, true
	case strings.HasSuffix(s, "h"):
		n, err := strconv.ParseUint(strings.TrimSuffix(s, "h"), 10, 64)
		if err != nil {
			return 0, false
		}
		return time.Duration(n) * time.Hour, true
	default:
		return 0, false
	}
}

// KV is the in-process store. One per server, held by app.Services.
type KV struct {
	mu     sync.RWMutex
	m      map[string]memStoredValue
	closed bool
	done   chan struct{}

	// now is injectable for tests so TTL expiry can be exercised without
	// real sleeps. Default is time.Now.
	now func() time.Time
}

// NewKV constructs a KV with the background cleanup goroutine running.
func NewKV() *KV {
	kv := &KV{
		m:    make(map[string]memStoredValue),
		done: make(chan struct{}),
		now:  time.Now,
	}
	go kv.cleanupLoop()
	return kv
}

func (kv *KV) cleanupLoop() {
	ticker := time.NewTicker(memKVCleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-kv.done:
			return
		case <-ticker.C:
			kv.cleanup(kv.now())
		}
	}
}

func (kv *KV) cleanup(now time.Time) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	for k, v := range kv.m {
		if v.expired(now) {
			delete(kv.m, k)
		}
	}
}

// Get returns ("", nil) for missing or expired keys. Callers rely on this
// (see auth_handler.go and app_helper.go).
func (kv *KV) Get(key string) (string, error) {
	if len(key) > memKVMaxKeyLen {
		return "", fmt.Errorf("key length exceeds %d characters", memKVMaxKeyLen)
	}
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.closed {
		return "", errKVClosed
	}
	sv, ok := kv.m[key]
	if !ok || sv.expired(kv.now()) {
		return "", nil
	}
	return sv.render(), nil
}

// Set stores key=value with optional TTL. Empty ttl = no expiration. Values
// that parse as int64 are stored as kindInt; everything else as kindStr.
func (kv *KV) Set(key, value, ttl string) error {
	if len(key) > memKVMaxKeyLen || len(value) > memKVMaxValueLen {
		return fmt.Errorf(
			"key or value length exceeds %d characters", memKVMaxKeyLen,
		)
	}
	d, ok := memParseExpiration(ttl)
	if !ok {
		return fmt.Errorf("invalid expiration: %s", ttl)
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return errKVClosed
	}
	var expiresAt time.Time
	if d > 0 {
		expiresAt = kv.now().Add(d)
	}
	sv := memStoredValue{expiresAt: expiresAt}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		sv.kind, sv.intVal = memKindInt, i
	} else {
		sv.kind, sv.strVal = memKindStr, value
	}
	kv.m[key] = sv
	return nil
}

// Incr increments the integer at key by 1 and returns the new value:
//   - missing → new value 1, TTL taken from ttl arg
//   - existing kindInt and not expired → +1, EXISTING TTL preserved
//   - existing kindStr OR expired → reset to 1, existing expiresAt preserved
//
// Del removes a key. Not-found is not an error (so single-use tokens are
// safe to double-consume). Expired entries are
// treated as already-gone.
func (kv *KV) Del(key string) error {
	if len(key) > memKVMaxKeyLen {
		return fmt.Errorf(
			"key length exceeds %d characters", memKVMaxKeyLen,
		)
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return errKVClosed
	}
	delete(kv.m, key)
	return nil
}

// A new ttl argument is honored ONLY on first create, so callers cannot
// silently extend a key's lifetime via repeated INCRs.
func (kv *KV) Incr(key string, ttl string) (int, error) {
	if len(key) > memKVMaxKeyLen {
		return 0, fmt.Errorf(
			"key length exceeds %d characters", memKVMaxKeyLen,
		)
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return 0, errKVClosed
	}

	cur, exists := kv.m[key]
	now := kv.now()

	var newVal int64
	switch {
	case !exists:
		newVal = 1
	case cur.expired(now):
		newVal = 1
	case cur.kind == memKindInt:
		newVal = cur.intVal + 1
	default:
		// existing kindStr value: reset counter to 1
		newVal = 1
	}

	var expiresAt time.Time
	if !exists {
		d, ok := memParseExpiration(ttl)
		if !ok {
			return 0, fmt.Errorf("invalid expiration: %s", ttl)
		}
		if d > 0 {
			expiresAt = now.Add(d)
		}
	} else {
		expiresAt = cur.expiresAt
	}

	kv.m[key] = memStoredValue{
		kind:      memKindInt,
		intVal:    newVal,
		expiresAt: expiresAt,
	}
	return int(newVal), nil
}

// Close stops the cleanup goroutine. Subsequent operations return errKVClosed.
// Safe to call twice.
func (kv *KV) Close() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return
	}
	kv.closed = true
	close(kv.done)
}
