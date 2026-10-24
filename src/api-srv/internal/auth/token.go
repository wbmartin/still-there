package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// NewInviteToken returns a hex-encoded 32-byte random token for an invite
// link (B035). It is a bearer credential for one login's first password, so
// only HashInviteToken of it is stored.
func NewInviteToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

// HashInviteToken is the value user_invite.token_hash holds for a token: its
// hex SHA-256. A fast hash is enough here, unlike a password: the token is
// 256 random bits, so there is nothing to brute-force from the hash.
func HashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewDeviceToken returns a token for a Pi base station: the same 256
// random bits as an invite, kept in the Pi's config file and sent as a
// Bearer header on every check-in. Only HashDeviceToken of it is stored.
func NewDeviceToken() (string, error) {
	return NewInviteToken()
}

// HashDeviceToken is the value device.token_hash holds for a token: its
// hex SHA-256, as HashInviteToken.
func HashDeviceToken(token string) string {
	return HashInviteToken(token)
}
