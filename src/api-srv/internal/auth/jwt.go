package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// IssuedAtNano returns the issue time in Unix ns that GenerateJWT writes
// to jti; false when jti does not hold one.
func (c *Claims) IssuedAtNano() (int64, bool) {
	issuedNano, err := strconv.ParseInt(c.ID, 10, 64)
	if err != nil {
		return 0, false
	}
	return issuedNano, true
}

// Argon2id parameters (OWASP recommended)
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
)

// GenerateJWT creates a new JWT token. issuer is Config.AppName so a
// derived app's tokens name it.
func GenerateJWT(
	issuer string,
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
			Issuer:  issuer,
			Subject: username,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour *
				time.Duration(expireHours))),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			// The issue time in ns: iat is whole seconds, too coarse to
			// order a token against a revocation (IssuedAtNano).
			ID: fmt.Sprintf("%d", now.UnixNano()),
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
		func(token *jwt.Token) (any, error) {
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
