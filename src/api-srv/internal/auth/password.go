package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, argon2Time,
		argon2Memory, argon2Threads, argon2KeyLen)

	// Encode as base64 for storage
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	result := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads, saltB64, hashB64)

	return result, nil
}

// CheckPasswordHash verifies a password against an Argon2id hash
func CheckPasswordHash(password, encodedHash string) bool {
	// Parse the encoded hash format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	// Parse parameters from parts[3]: m=65536,t=3,p=4
	paramStr := parts[3]
	params := strings.Split(paramStr, ",")
	var memory, time, threads uint32
	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) != 2 {
			return false
		}
		val, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return false
		}
		switch kv[0] {
		case "m":
			memory = uint32(val)
		case "t":
			time = uint32(val)
		case "p":
			if val > 255 {
				return false
			}
			threads = uint32(val)
		default:
			return false
		}
	}

	// Validate parameters have reasonable ranges
	if memory == 0 || time == 0 || threads == 0 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Hash with parsed parameters
	hash := argon2.IDKey([]byte(password), salt, time, memory, uint8(threads), argon2KeyLen)

	// Compare hashes using constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}
