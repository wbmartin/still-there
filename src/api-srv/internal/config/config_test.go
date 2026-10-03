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

// validBase is every required key except LOG_LEVEL, which each test adds
// or leaves out itself.
const validBase = `DATABASE_URL=postgres://u:p@h/db
JWT_SECRET=s
SERVER_NAME=n
APP_NAME=a
SERVER_PORT=8080
ENABLE_TLS=false
TOTP_ENCRYPTION_KEY=k
FAILED_LOGIN_ATTEMPT_THRESHOLD=7
FAILED_LOGIN_ATTEMPT_WINDOW_DURATION=5min
FAILED_LOGIN_LOCKOUT_DURATION=5min
`

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
				writeEnv(t, validBase+testCase.line),
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
				t, validBase+"LOG_LEVEL=info\n"+testCase.line,
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
