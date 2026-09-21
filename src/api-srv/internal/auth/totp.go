package auth

// TOTP secrets, otpauth URIs and the AES-GCM at-rest encryption of the
// seed. Handlers call these; nothing else in the tree touches the crypto.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret generates a valid Base32 TOTP secret (no padding, uppercase)
func GenerateTOTPSecret() (string, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", err
	}

	// Encode to Base32, strip padding, ensure uppercase
	secret := base32.StdEncoding.EncodeToString(secretBytes)
	secret = strings.TrimRight(secret, "=")
	secret = strings.ToUpper(secret)

	return secret, nil
}

// ValidateTOTPSecret validates Base32 secret format (no padding allowed)
func ValidateTOTPSecret(secret string) bool {
	if len(secret) == 0 {
		return false
	}

	// Check for valid Base32 characters only (A-Z, 2-7, no padding)
	for _, r := range secret {
		if !((r >= 'A' && r <= 'Z') || (r >= '2' && r <= '7')) {
			return false
		}
	}

	return true
}

// formatTOTPLabel creates a properly formatted TOTP label
func formatTOTPLabel(issuer, account string) string {
	// Clean issuer for label (allow alphanumeric, underscore, dash)
	cleanIssuer := regexp.MustCompile(`[^a-zA-Z0-9_-]`).
		ReplaceAllString(issuer, "")
	return cleanIssuer + ":" + account
}

// GenerateTOTPURI generates a fully compliant otpauth:// URI
func GenerateTOTPURI(issuer, account, secret string) (string, error) {
	label := formatTOTPLabel(issuer, account)

	// Build URI with minimal required parameters (SHA1, 6 digits, 30s period are defaults)
	uri := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s",
		url.PathEscape(label), secret, url.QueryEscape(issuer))

	return uri, nil
}

// EncryptTOTPSeed encrypts a TOTP seed using AES-GCM
func EncryptTOTPSeed(plainSeed, keyHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainSeed), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSeed decrypts a TOTP seed using AES-GCM
func DecryptTOTPSeed(encryptedSeed, keyHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSeed)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ValidateTOTPCode checks a six-digit code against the plaintext secret
func ValidateTOTPCode(code, secret string) bool {
	return totp.Validate(code, secret)
}
