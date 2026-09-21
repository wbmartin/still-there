package model

import (
	"crypto/rand"
	"fmt"
)

// NewUUID returns a random (version 4) UUID in canonical text form. It is the
// server-assigned identity for rows with a uuid column (example.example_uuid);
// handlers set it on create and it never changes. No library needed.
func NewUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is not recoverable; the OS entropy source is gone.
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
