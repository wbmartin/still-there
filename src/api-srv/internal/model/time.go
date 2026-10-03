package model

import "time"

// The one timestamp formula. Every create_ts / update_ts is epoch seconds
// in a NUMERIC(17,7) column, and update_ts is the optimistic lock, so every
// writer must spell the clock the same way. It lives in model because both
// repository and handler already import it, and the timestamp is part of
// the row contract (CreateTS / UpdateTS on every model).
//
// float64(t.Unix()) + float64(t.Nanosecond())/1e9 is used rather than
// float64(t.UnixNano())/1e9: at present-day nanosecond magnitudes (~1.8e18)
// a float64 cannot hold the integer exactly, so the second form is off by
// up to ~128 ns and cannot represent a whole second exactly. The time is
// read once: reading time.Now() twice (once for Unix, once for Nanosecond)
// can straddle a second boundary and come out up to a second early.

// EpochSeconds is t as epoch seconds with sub-second precision.
func EpochSeconds(t time.Time) float64 {
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}

// Now is the current instant as epoch seconds with sub-second precision,
// what a row's CreateTS / UpdateTS is set to.
func Now() float64 {
	return EpochSeconds(time.Now())
}
