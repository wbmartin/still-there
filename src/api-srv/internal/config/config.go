package config

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/wbmartin/starter-api/internal/cache"
	"github.com/wbmartin/starter-api/internal/logging"
)

// Config holds the application configuration. It is loaded from a flat
// KEY=VALUE file (doc/905-config-guidance.md) and is never serialized, so
// its fields carry no json tags. Never marshal, log or print it whole:
// JWTSecret and TOTPEncryptionKey are secrets.
type Config struct {
	DatabaseURL                      string
	JWTSecret                        string
	JWTSessionExpireHours            int
	MaxUserReqPerMin                 int
	MaxUnauthReqPerMin               int
	ServerName                       string
	AppName                          string
	ServerPort                       int
	ServerBindAddress                string
	EnableTLS                        bool
	TLSCertPath                      string
	TLSKeyPath                       string
	TOTPEncryptionKey                string
	FailedLoginAttemptThreshold      int
	FailedLoginAttemptWindowDuration string
	FailedLoginLockoutDuration       string
	CorsAllowedOrigins               []string
	TrustedProxyCount                int
	LogLevel                         slog.Level
	InviteTTLHours                   int
	AllowMFAExemptViaAPI             bool
}

// ReadEnvFile parses a KEY=VALUE file -- blank lines and # comments ignored,
// no interpolation -- into a map. A value wrapped in matching double or
// single quotes has the quotes removed (so APP_NAME="Proto App" is
// Proto App, not "Proto App"); anything else is taken verbatim after
// trimming whitespace. Shared by LoadConfig and by cmd/migrate, which needs
// only DATABASE_URL from a file of the same shape.
func ReadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			env[key] = unquote(strings.TrimSpace(parts[1]))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}
	return env, nil
}

// unquote strips one pair of matching surrounding quotes. An unmatched or
// lone quote is content, not syntax.
func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// minJWTSecretLength is the shortest JWT_SECRET LoadConfig accepts; every
// host gets 64 hex characters (bin/610, doc/330, doc/331).
const minJWTSecretLength = 32

// totpKeyByteLength is the AES-256 key size TOTP_ENCRYPTION_KEY decodes to
// (auth.EncryptTOTPSeed hex-decodes it for aes.NewCipher).
const totpKeyByteLength = 32

// LoadConfig loads configuration from a .env file
func LoadConfig(configFilePath string) (*Config, error) {
	env, err := ReadEnvFile(configFilePath)
	if err != nil {
		return nil, err
	}

	var config Config

	// Populate from parsed env map
	config.DatabaseURL = env["DATABASE_URL"]
	config.JWTSecret = env["JWT_SECRET"]

	// 0 here would mint every JWT already expired or 429 every request.
	if config.JWTSessionExpireHours, err = requirePositiveInt(
		env, "JWT_SESSION_EXPIRE_HOURS",
	); err != nil {
		return nil, err
	}
	if config.MaxUserReqPerMin, err = requirePositiveInt(
		env, "MAX_USER_REQ_PER_MIN",
	); err != nil {
		return nil, err
	}
	if config.MaxUnauthReqPerMin, err = requirePositiveInt(
		env, "MAX_UNAUTH_REQ_PER_MIN",
	); err != nil {
		return nil, err
	}

	// How long an invite link works (B035); 0 would issue dead links.
	if config.InviteTTLHours, err = requirePositiveInt(
		env, "INVITE_TTL_HOURS",
	); err != nil {
		return nil, err
	}

	// Whether POST/PUT /user and /userInvite may turn mfa_exempt on (B097).
	// Required and spelled out, so no host allows it by omission: true
	// only for the test harness, false on every server.
	switch env["ALLOW_MFA_EXEMPT_VIA_API"] {
	case "true":
		config.AllowMFAExemptViaAPI = true
	case "false":
		config.AllowMFAExemptViaAPI = false
	case "":
		return nil, fmt.Errorf("ALLOW_MFA_EXEMPT_VIA_API is required " +
			"(true or false; false on every server)")
	default:
		return nil, fmt.Errorf(
			"ALLOW_MFA_EXEMPT_VIA_API must be true or false, got %q",
			env["ALLOW_MFA_EXEMPT_VIA_API"],
		)
	}

	logLevelStr := env["LOG_LEVEL"]
	if logLevelStr == "" {
		return nil, fmt.Errorf("LOG_LEVEL is required")
	}
	logLevel, ok := logging.ParseLevel(logLevelStr)
	if !ok {
		return nil, fmt.Errorf(
			"LOG_LEVEL must be debug, info, warn or error, got %q",
			logLevelStr,
		)
	}
	config.LogLevel = logLevel

	config.ServerName = env["SERVER_NAME"]
	config.AppName = env["APP_NAME"]
	// Required, so a forgotten key fails closed instead of listening on
	// every interface. 127.0.0.1 is right behind nginx, for dev and for the
	// test harness, whose instance carries a well-known password.
	config.ServerBindAddress = env["SERVER_BIND_ADDRESS"]
	if config.ServerBindAddress == "" {
		return nil, fmt.Errorf(
			"SERVER_BIND_ADDRESS is required (the interface to listen " +
				"on, e.g. 127.0.0.1; 0.0.0.0 for every interface)",
		)
	}

	if config.ServerPort, err = requirePositiveInt(
		env, "SERVER_PORT",
	); err != nil {
		return nil, err
	}

	if enableTLSStr := env["ENABLE_TLS"]; enableTLSStr != "" {
		if val, err := strconv.ParseBool(enableTLSStr); err == nil {
			config.EnableTLS = val
		} else {
			return nil, fmt.Errorf("ENABLE_TLS must be a valid boolean (true/false)")
		}
	} else {
		return nil, fmt.Errorf("ENABLE_TLS is required")
	}

	config.TLSCertPath = env["TLS_CERT_PATH"]
	config.TLSKeyPath = env["TLS_KEY_PATH"]
	config.TOTPEncryptionKey = env["TOTP_ENCRYPTION_KEY"]

	if config.FailedLoginAttemptThreshold, err = requirePositiveInt(
		env, "FAILED_LOGIN_ATTEMPT_THRESHOLD",
	); err != nil {
		return nil, err
	}
	// Checked here so a bad value fails start-up, not every failed login.
	if config.FailedLoginAttemptWindowDuration, err = requireDuration(
		env, "FAILED_LOGIN_ATTEMPT_WINDOW_DURATION",
	); err != nil {
		return nil, err
	}
	if config.FailedLoginLockoutDuration, err = requireDuration(
		env, "FAILED_LOGIN_LOCKOUT_DURATION",
	); err != nil {
		return nil, err
	}

	// Optional, default 0: no proxy, so X-Forwarded-For is ignored. Behind
	// the host's nginx it is 1 (security.ClientIP).
	if proxyCountStr := env["TRUSTED_PROXY_COUNT"]; proxyCountStr != "" {
		val, err := strconv.Atoi(proxyCountStr)
		if err != nil || val < 0 {
			return nil, fmt.Errorf(
				"TRUSTED_PROXY_COUNT must be a non-negative integer",
			)
		}
		config.TrustedProxyCount = val
	}

	if corsOriginsStr := env["CORS_ALLOWED_ORIGINS"]; corsOriginsStr != "" {
		origins := strings.Split(corsOriginsStr, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		config.CorsAllowedOrigins = origins
	} else {
		config.CorsAllowedOrigins = []string{} // default to no CORS
	}

	// Validate required configuration
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(config.JWTSecret) < minJWTSecretLength {
		return nil, fmt.Errorf(
			"JWT_SECRET must be at least %d characters "+
				"(openssl rand -hex 32)", minJWTSecretLength,
		)
	}
	if config.ServerName == "" {
		return nil, fmt.Errorf("SERVER_NAME is required")
	}
	if config.AppName == "" {
		return nil, fmt.Errorf("APP_NAME is required")
	}
	if config.EnableTLS {
		if config.TLSCertPath == "" {
			return nil, fmt.Errorf(
				"TLS_CERT_PATH is required when ENABLE_TLS is true",
			)
		}
		if config.TLSKeyPath == "" {
			return nil, fmt.Errorf(
				"TLS_KEY_PATH is required when ENABLE_TLS is true",
			)
		}
	}
	if config.TOTPEncryptionKey == "" {
		return nil, fmt.Errorf("TOTP_ENCRYPTION_KEY is required")
	}
	// A key that is not AES-256 hex used to load and then answer 500 at
	// the first MFA enrolment or login.
	totpKey, err := hex.DecodeString(config.TOTPEncryptionKey)
	if err != nil || len(totpKey) != totpKeyByteLength {
		return nil, fmt.Errorf(
			"TOTP_ENCRYPTION_KEY must be %d hex characters "+
				"(%d bytes, openssl rand -hex 32)",
			2*totpKeyByteLength, totpKeyByteLength,
		)
	}

	return &config, nil
}

// requirePositiveInt reads a key that must be a whole number above 0.
func requirePositiveInt(env map[string]string, key string) (int, error) {
	raw := env[key]
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf(
			"%s must be a positive integer, got %q", key, raw,
		)
	}
	return val, nil
}

// requireDuration reads a key that must be a cache.KV TTL above 0 (30s,
// 5min, 1h); the string is kept because KV takes it in that form.
func requireDuration(env map[string]string, key string) (string, error) {
	raw := env[key]
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	if d, ok := cache.ParseTTL(raw); !ok || d <= 0 {
		return "", fmt.Errorf(
			"%s must be a duration such as 30s, 5min or 1h, got %q",
			key, raw,
		)
	}
	return raw, nil
}
