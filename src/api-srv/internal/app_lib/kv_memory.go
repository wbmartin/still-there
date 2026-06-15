package app_lib

// In-memory implementation of ThisAppKVInterface.
//
// Default backend for single-server deployments. Counters, lockouts, MFA temp
// tokens, and rate-limit state are PROCESS-LOCAL — restarting the API server
// clears them. This is acceptable for one app server.
//
// Graduation trigger: the day this app runs behind a load balancer with two
// or more API server processes, switch KV_BACKEND=shrmpl. Otherwise rate
// limits and lockouts become per-process and an attacker can spread attempts
// to evade them; MFA temp tokens issued by one server won't validate on
// another.
//
// Storage logic (storedValue, parseExpiration, INCR lifecycle, cleanup)
// is ported from shrmpl-kv-srv/main.go with the TCP/protocol layer removed.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	memKVMaxKeyLen   = 100
	memKVMaxValueLen = 100
	memKVMaxBatchOps = 3
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

// memParseExpiration accepts the shrmpl-kv expiration grammar: Ns, Nmin, Nh.
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

// MemoryKV is an in-process implementation of ThisAppKVInterface.
type MemoryKV struct {
	mu     sync.RWMutex
	m      map[string]memStoredValue
	closed bool
	done   chan struct{}

	// now is injectable for tests so TTL expiry can be exercised without
	// real sleeps. Default is time.Now.
	now func() time.Time
}

// NewMemoryKV constructs a MemoryKV with the background cleanup goroutine
// running.
func NewMemoryKV() *MemoryKV {
	kv := &MemoryKV{
		m:    make(map[string]memStoredValue),
		done: make(chan struct{}),
		now:  time.Now,
	}
	go kv.cleanupLoop()
	return kv
}

func (kv *MemoryKV) cleanupLoop() {
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

func (kv *MemoryKV) cleanup(now time.Time) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	for k, v := range kv.m {
		if v.expired(now) {
			delete(kv.m, k)
		}
	}
}

// Get returns ("", nil) for missing or expired keys — matching shrmpl-kv's
// "*KEY NOT FOUND*" treatment translated to the Go client contract. Callers
// in this codebase rely on this (see auth_handler.go and app_helper.go).
func (kv *MemoryKV) Get(key string) (string, error) {
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
func (kv *MemoryKV) Set(key, value, ttl string) error {
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

// Incr increments the integer at key by 1 and returns the new value. The
// lifecycle matches shrmpl-kv exactly:
//   - missing → new value 1, TTL taken from ttl arg
//   - existing kindInt and not expired → +1, EXISTING TTL preserved
//   - existing kindStr OR expired → reset to 1, existing expiresAt preserved
//
// Del removes a key. Mirrors shrmpl-kv: not-found is not an error (so
// single-use tokens are safe to double-consume). Expired entries are
// treated as already-gone.
func (kv *MemoryKV) Del(key string) error {
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

// Note the deliberate quirk: a new ttl argument is honored ONLY on first
// create. This prevents callers from silently extending a key's lifetime via
// repeated INCRs and matches the comment in shrmpl-kv-srv/main.go.
func (kv *MemoryKV) Incr(key string, ttl string) (int, error) {
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

// Batch executes up to memKVMaxBatchOps shrmpl-kv commands atomically under
// one lock and returns one result string per command. Per-command errors
// surface as "ERROR <reason>" strings in the result slice, not as a Go error
// — this matches shrmpl-kv so that *KEY NOT FOUND* is treated as a value,
// not as failure.
func (kv *MemoryKV) Batch(commands []string) ([]string, error) {
	if len(commands) > memKVMaxBatchOps {
		return nil, fmt.Errorf(
			"batch cannot exceed %d commands", memKVMaxBatchOps,
		)
	}

	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return nil, errKVClosed
	}

	results := make([]string, 0, len(commands))
	for _, cmd := range commands {
		parts := strings.Fields(strings.TrimSpace(cmd))
		results = append(results, kv.execLocked(parts))
	}
	return results, nil
}

// execLocked dispatches a single parsed command while holding kv.mu. Returns
// the per-command result string (already stripped of trailing newline).
func (kv *MemoryKV) execLocked(parts []string) string {
	if len(parts) == 0 {
		return "ERROR unknown command"
	}
	switch parts[0] {
	case "GET":
		return kv.batchGetLocked(parts)
	case "SET":
		return kv.batchSetLocked(parts)
	case "INCR":
		return kv.batchIncrLocked(parts)
	default:
		return "ERROR unknown command"
	}
}

func (kv *MemoryKV) batchGetLocked(parts []string) string {
	if len(parts) != 2 {
		return "ERROR invalid arguments"
	}
	key := parts[1]
	if len(key) > memKVMaxKeyLen {
		return "ERROR invalid length"
	}
	sv, ok := kv.m[key]
	if !ok || sv.expired(kv.now()) {
		return "*KEY NOT FOUND*"
	}
	return sv.render()
}

func (kv *MemoryKV) batchSetLocked(parts []string) string {
	if len(parts) < 3 || len(parts) > 4 {
		return "ERROR invalid arguments"
	}
	key, value := parts[1], parts[2]
	if len(key) > memKVMaxKeyLen || len(value) > memKVMaxValueLen {
		return "ERROR invalid length"
	}
	var expiresAt time.Time
	if len(parts) == 4 {
		d, ok := memParseExpiration(parts[3])
		if !ok {
			return "ERROR invalid expiration"
		}
		if d > 0 {
			expiresAt = kv.now().Add(d)
		}
	}
	sv := memStoredValue{expiresAt: expiresAt}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		sv.kind, sv.intVal = memKindInt, i
	} else {
		sv.kind, sv.strVal = memKindStr, value
	}
	kv.m[key] = sv
	return "OK"
}

func (kv *MemoryKV) batchIncrLocked(parts []string) string {
	if len(parts) < 2 || len(parts) > 3 {
		return "ERROR invalid arguments"
	}
	key := parts[1]
	if len(key) > memKVMaxKeyLen {
		return "ERROR invalid length"
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
		newVal = 1
	}

	var expiresAt time.Time
	if !exists {
		if len(parts) == 3 {
			d, ok := memParseExpiration(parts[2])
			if !ok {
				return "ERROR invalid expiration"
			}
			if d > 0 {
				expiresAt = now.Add(d)
			}
		}
	} else {
		expiresAt = cur.expiresAt
	}

	kv.m[key] = memStoredValue{
		kind:      memKindInt,
		intVal:    newVal,
		expiresAt: expiresAt,
	}
	return strconv.FormatInt(newVal, 10)
}

// Close stops the cleanup goroutine. Subsequent operations return errKVClosed.
// Safe to call twice.
func (kv *MemoryKV) Close() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return
	}
	kv.closed = true
	close(kv.done)
}
