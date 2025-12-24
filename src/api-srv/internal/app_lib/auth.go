package app_lib

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Claims represents the JWT claims structure
type Claims struct {
	jwt.RegisteredClaims
	UserID          int    `json:"uid"`
	SessionID       string `json:"ses"`
	SecurityContext int    `json:"sctx"`
	Grants          []int  `json:"grants"`
	// []grant_id for current security context
}

// Argon2id parameters (OWASP recommended)
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
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

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Extract parameters from the format
	// For now, assume fixed parameters as defined in HashPassword
	hash := argon2.IDKey([]byte(password), salt, argon2Time,
		argon2Memory, argon2Threads, argon2KeyLen)

	// Compare hashes using constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

// GenerateJWT creates a new JWT token
func GenerateJWT(
	username string,
	userID int,
	sessionID string,
	securityContext int,
	grants []int,
	jwtSecret string,
	expireHours int,
) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  "proto-go-app",
			Subject: username,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour *
				time.Duration(expireHours))),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("%d", now.UnixNano()),
		},
		UserID:          userID,
		SessionID:       sessionID,
		SecurityContext: securityContext,
		Grants:          grants,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf(
					"unexpected signing method: %v",
					token.Header["alg"],
				)
			}
			return []byte(jwtSecret), nil
		})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid JWT token")
}
