package config

import (
	"os"
	"path/filepath"
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

// TRUSTED_PROXY_COUNT is optional (0 when absent) and must be a
// non-negative integer when given.
func TestLoadConfig_TrustedProxyCount(t *testing.T) {
	base := `DATABASE_URL=postgres://u:p@h/db
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
			path := filepath.Join(t.TempDir(), "api.env")
			content := base + testCase.line
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			config, err := LoadConfig(path)
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
