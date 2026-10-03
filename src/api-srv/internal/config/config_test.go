package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadEnvFile_QuotesAndEdges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.env")
	content := `# comment
APP_NAME="Proto App"
SINGLE='single quoted'
PLAIN=plain value
URL=postgres://u:p@h:5432/db?sslmode=disable
INNER_EQ=a=b=c
EMPTY=
LONE_QUOTE="unterminated
JUST_QUOTES=""
SPACED   =   padded
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := ReadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"APP_NAME":    "Proto App",
		"SINGLE":      "single quoted",
		"PLAIN":       "plain value",
		"URL":         "postgres://u:p@h:5432/db?sslmode=disable",
		"INNER_EQ":    "a=b=c",
		"EMPTY":       "",
		"LONE_QUOTE":  `"unterminated`,
		"JUST_QUOTES": "",
		"SPACED":      "padded",
	}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("%s = %q, want %q", k, env[k], v)
		}
	}
	if _, ok := env["# comment"]; ok {
		t.Errorf("comment line was parsed as a key")
	}
}

// validBase is every required key; tests drop or append lines.
const validBase = `DATABASE_URL=postgres://u:p@h/db
JWT_SECRET=s
SERVER_NAME=n
APP_NAME=a
SERVER_PORT=8080
SERVER_BIND_ADDRESS=127.0.0.1
ENABLE_TLS=false
TOTP_ENCRYPTION_KEY=k
JWT_SESSION_EXPIRE_HOURS=12
MAX_USER_REQ_PER_MIN=200
MAX_UNAUTH_REQ_PER_MIN=20
FAILED_LOGIN_ATTEMPT_THRESHOLD=7
FAILED_LOGIN_ATTEMPT_WINDOW_DURATION=5min
FAILED_LOGIN_LOCKOUT_DURATION=5min
LOG_LEVEL=info
INVITE_TTL_HOURS=72
`

// withoutKey returns validBase minus the line that sets key.
func withoutKey(key string) string {
	var kept []string
	for _, line := range strings.SplitAfter(validBase, "\n") {
		if !strings.HasPrefix(line, key+"=") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "")
}

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "api.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// LOG_LEVEL is required and must be one of debug, info, warn, error.
func TestLoadConfig_LogLevel(t *testing.T) {
	testCaseList := []struct {
		name    string
		line    string
		want    slog.Level
		wantErr string
	}{
		{"absent", "", 0, "LOG_LEVEL is required"},
		{"empty", "LOG_LEVEL=\n", 0, "LOG_LEVEL is required"},
		{"debug", "LOG_LEVEL=debug\n", slog.LevelDebug, ""},
		{"info", "LOG_LEVEL=info\n", slog.LevelInfo, ""},
		{"warn", "LOG_LEVEL=warn\n", slog.LevelWarn, ""},
		{"error", "LOG_LEVEL=error\n", slog.LevelError, ""},
		{"quoted", "LOG_LEVEL=\"warn\"\n", slog.LevelWarn, ""},
		{"unknown", "LOG_LEVEL=verbose\n", 0, "LOG_LEVEL must be"},
		{"upper case", "LOG_LEVEL=INFO\n", 0, "LOG_LEVEL must be"},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			config, err := LoadConfig(
				writeEnv(t, withoutKey("LOG_LEVEL")+testCase.line),
			)
			if testCase.wantErr != "" {
				if err == nil ||
					!strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("err = %v, want %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.LogLevel != testCase.want {
				t.Errorf("LogLevel = %v, want %v",
					config.LogLevel, testCase.want)
			}
		})
	}
}

// SERVER_BIND_ADDRESS is required: an empty bind used to listen on every
// interface, beside nginx rather than behind it.
func TestLoadConfig_ServerBindAddress(t *testing.T) {
	testCaseList := []struct {
		name    string
		line    string
		want    string
		wantErr bool
	}{
		{"absent", "", "", true},
		{"empty", "SERVER_BIND_ADDRESS=\n", "", true},
		{"loopback", "SERVER_BIND_ADDRESS=127.0.0.1\n", "127.0.0.1", false},
		{"every interface", "SERVER_BIND_ADDRESS=0.0.0.0\n", "0.0.0.0", false},
		{"ipv6 loopback", "SERVER_BIND_ADDRESS=::1\n", "::1", false},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			config, err := LoadConfig(writeEnv(
				t, withoutKey("SERVER_BIND_ADDRESS")+testCase.line,
			))
			if testCase.wantErr {
				wantErr := "SERVER_BIND_ADDRESS is required"
				if err == nil || !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("err = %v, want %q", err, wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.ServerBindAddress != testCase.want {
				t.Errorf("ServerBindAddress = %q, want %q",
					config.ServerBindAddress, testCase.want)
			}
		})
	}
}

// TRUSTED_PROXY_COUNT is optional (0 when absent) and must be a
// non-negative integer when given.
func TestLoadConfig_TrustedProxyCount(t *testing.T) {
	testCaseList := []struct {
		name    string
		line    string
		want    int
		wantErr bool
	}{
		{"absent", "", 0, false},
		{"zero", "TRUSTED_PROXY_COUNT=0\n", 0, false},
		{"one", "TRUSTED_PROXY_COUNT=1\n", 1, false},
		{"negative", "TRUSTED_PROXY_COUNT=-1\n", 0, true},
		{"not a number", "TRUSTED_PROXY_COUNT=yes\n", 0, true},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			config, err := LoadConfig(writeEnv(
				t, validBase+testCase.line,
			))
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %+v", config)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.TrustedProxyCount != testCase.want {
				t.Errorf(
					"TrustedProxyCount = %d, want %d",
					config.TrustedProxyCount, testCase.want,
				)
			}
		})
	}
}

// A missing or non-positive integer used to load as 0, minting every JWT
// already expired and 429ing every request. Each must now fail by name.
func TestLoadConfig_PositiveIntKeys(t *testing.T) {
	keyList := []string{
		"JWT_SESSION_EXPIRE_HOURS", "MAX_USER_REQ_PER_MIN",
		"MAX_UNAUTH_REQ_PER_MIN", "SERVER_PORT",
		"FAILED_LOGIN_ATTEMPT_THRESHOLD", "INVITE_TTL_HOURS",
	}
	testCaseList := []struct {
		name    string
		value   *string
		wantErr string
	}{
		{"absent", nil, "is required"},
		{"empty", ptr(""), "is required"},
		{"zero", ptr("0"), "must be a positive integer"},
		{"negative", ptr("-1"), "must be a positive integer"},
		{"not a number", ptr("abc"), "must be a positive integer"},
		{"unit suffix", ptr("200/min"), "must be a positive integer"},
		{"valid", ptr("200"), ""},
	}
	for _, key := range keyList {
		for _, testCase := range testCaseList {
			t.Run(key+"/"+testCase.name, func(t *testing.T) {
				content := withoutKey(key)
				if testCase.value != nil {
					content += key + "=" + *testCase.value + "\n"
				}
				config, err := LoadConfig(writeEnv(t, content))
				if testCase.wantErr == "" {
					if err != nil {
						t.Fatal(err)
					}
					if got := intField(config, key); got != 200 {
						t.Errorf("%s = %d, want 200", key, got)
					}
					return
				}
				wantErr := key + " " + testCase.wantErr
				if err == nil || !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("err = %v, want %q", err, wantErr)
				}
			})
		}
	}
}

// The FAILED_LOGIN_* durations are KV TTLs; a bad one used to load and
// then fail every failed login.
func TestLoadConfig_DurationKeys(t *testing.T) {
	keyList := []string{
		"FAILED_LOGIN_ATTEMPT_WINDOW_DURATION",
		"FAILED_LOGIN_LOCKOUT_DURATION",
	}
	testCaseList := []struct {
		name    string
		value   *string
		wantErr string
	}{
		{"absent", nil, "is required"},
		{"empty", ptr(""), "is required"},
		{"no unit", ptr("5"), "must be a duration"},
		{"zero", ptr("0min"), "must be a duration"},
		{"word", ptr("never"), "must be a duration"},
		{"go syntax", ptr("5m"), "must be a duration"},
		{"seconds", ptr("30s"), ""},
		{"minutes", ptr("5min"), ""},
		{"hours", ptr("1h"), ""},
	}
	for _, key := range keyList {
		for _, testCase := range testCaseList {
			t.Run(key+"/"+testCase.name, func(t *testing.T) {
				content := withoutKey(key)
				if testCase.value != nil {
					content += key + "=" + *testCase.value + "\n"
				}
				_, err := LoadConfig(writeEnv(t, content))
				if testCase.wantErr == "" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				wantErr := key + " " + testCase.wantErr
				if err == nil || !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("err = %v, want %q", err, wantErr)
				}
			})
		}
	}
}

// Every API env file the repo ships loads through the real loader. Infra
// copies hold SERVERONLY and the UAT template __PLACEHOLDERS__; LoadConfig
// checks presence and shape, not secret content, so they load as-is.
func TestLoadConfig_RepoEnvFiles(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..", "..")
	var pathList []string
	for _, pattern := range []string{
		"etc/*-api-srv.env",
		"etc/*-api-srv.env.template",
		"infra/*/etc/*-api.env",
	} {
		matchList, err := filepath.Glob(filepath.Join(repoRoot, pattern))
		if err != nil {
			t.Fatal(err)
		}
		pathList = append(pathList, matchList...)
	}
	if len(pathList) < 6 {
		t.Fatalf("found %d API env files, want at least 6: %v",
			len(pathList), pathList)
	}
	for _, path := range pathList {
		if _, err := LoadConfig(path); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

func ptr(s string) *string { return &s }

func intField(config *Config, key string) int {
	return map[string]int{
		"JWT_SESSION_EXPIRE_HOURS":       config.JWTSessionExpireHours,
		"MAX_USER_REQ_PER_MIN":           config.MaxUserReqPerMin,
		"MAX_UNAUTH_REQ_PER_MIN":         config.MaxUnauthReqPerMin,
		"SERVER_PORT":                    config.ServerPort,
		"FAILED_LOGIN_ATTEMPT_THRESHOLD": config.FailedLoginAttemptThreshold,
		"INVITE_TTL_HOURS":               config.InviteTTLHours,
	}[key]
}
