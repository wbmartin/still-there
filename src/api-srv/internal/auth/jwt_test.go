package auth

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateValidateJWT_RoundTrip(t *testing.T) {
	grants := []int{1001, 1002}
	tokenString, err := GenerateJWT(
		"stillthere-app", "alice", 42, "sess-1", 7, grants, "s3cret", 1,
	)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	claims, err := ValidateJWT(tokenString, "s3cret")
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}

	if claims.Issuer != "stillthere-app" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "stillthere-app")
	}
	if claims.Subject != "alice" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "alice")
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", claims.SessionID, "sess-1")
	}
	if claims.SecurityContext != 7 {
		t.Errorf("SecurityContext = %d, want 7", claims.SecurityContext)
	}
	if !reflect.DeepEqual(claims.Grants, grants) {
		t.Errorf("Grants = %v, want %v", claims.Grants, grants)
	}
}

func TestValidateJWT_WrongSecretFails(t *testing.T) {
	tokenString, err := GenerateJWT(
		"stillthere-app", "alice", 1, "sess", 1, nil, "right-secret", 1,
	)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	if _, err := ValidateJWT(tokenString, "wrong-secret"); err == nil {
		t.Error("ValidateJWT with wrong secret: want error, got nil")
	}
}

func TestValidateJWT_TamperedTokenFails(t *testing.T) {
	tokenString, err := GenerateJWT(
		"stillthere-app", "alice", 1, "sess", 1, nil, "s3cret", 1,
	)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	// Flip a character in the payload segment to invalidate the signature.
	tampered := []byte(tokenString)
	dot := -1
	for i, c := range tampered {
		if c == '.' {
			dot = i
			break
		}
	}
	if dot < 0 || dot+1 >= len(tampered) {
		t.Fatalf("could not locate payload segment in token %q", tokenString)
	}
	flip := dot + 1
	if tampered[flip] == 'A' {
		tampered[flip] = 'B'
	} else {
		tampered[flip] = 'A'
	}

	if _, err := ValidateJWT(string(tampered), "s3cret"); err == nil {
		t.Error("ValidateJWT with tampered payload: want error, got nil")
	}
}

func TestValidateJWT_ExpiredTokenFails(t *testing.T) {
	// expireHours of 0 sets ExpiresAt to time.Now(), which is already in
	// the past by the time ValidateJWT parses it.
	tokenString, err := GenerateJWT(
		"stillthere-app", "alice", 1, "sess", 1, nil, "s3cret", 0,
	)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	// Guarantee we are past the expiry instant regardless of clock
	// resolution.
	time.Sleep(2 * time.Millisecond)

	if _, err := ValidateJWT(tokenString, "s3cret"); err == nil {
		t.Error("ValidateJWT with expired token: want error, got nil")
	}
}

func TestValidateJWT_RejectsNonHMACMethod(t *testing.T) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "stillthere-app",
			Subject:   "alice",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing alg=none token: %v", err)
	}

	if _, err := ValidateJWT(tokenString, "s3cret"); err == nil {
		t.Error("ValidateJWT with alg=none token: want error, got nil")
	}
}
