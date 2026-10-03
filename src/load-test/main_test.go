package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadEnvKey(t *testing.T) {
	caseList := []struct {
		name    string
		content string
		want    string
	}{
		{"plain", "TEST_PERSONA_PASSWORD=basic123!\n", "basic123!"},
		{"spaces", "  TEST_PERSONA_PASSWORD = basic123! \n", "basic123!"},
		{"double quotes", "TEST_PERSONA_PASSWORD=\"a b#c\"\n", "a b#c"},
		{"single quotes", "TEST_PERSONA_PASSWORD='x=y'\n", "x=y"},
		{"unmatched quote kept", "TEST_PERSONA_PASSWORD=\"abc\n", "\"abc"},
		{"export prefix", "export TEST_PERSONA_PASSWORD='p'\n", "p"},
		{"equals in value", "TEST_PERSONA_PASSWORD=a=b\n", "a=b"},
		{
			"comments, other keys, prefix names skipped",
			"# TEST_PERSONA_PASSWORD=commented\n" +
				"TEST_PERSONA_PASSWORD_OLD=old\n" +
				"OTHER=1\n\n" +
				"TEST_PERSONA_PASSWORD=right\n",
			"right",
		},
		{
			"last one wins, as in the API config",
			"TEST_PERSONA_PASSWORD=first\nTEST_PERSONA_PASSWORD=second\n",
			"second",
		},
	}
	for _, testCase := range caseList {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeEnvFile(t, testCase.content)
			got, err := readEnvKey(path, passwordKey)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReadEnvKeyErrors(t *testing.T) {
	for _, content := range []string{
		"",
		"OTHER=1\n",
		"TEST_PERSONA_PASSWORD=\n",
		"TEST_PERSONA_PASSWORD=''\n",
		"# TEST_PERSONA_PASSWORD=commented\n",
	} {
		path := writeEnvFile(t, content)
		_, err := readEnvKey(path, passwordKey)
		if err == nil || !strings.Contains(err.Error(), "not set") {
			t.Fatalf("content %q: want a not-set error, got %v", content, err)
		}
	}
	_, err := readEnvKey(
		filepath.Join(t.TempDir(), "missing.env"), passwordKey,
	)
	if err == nil || !strings.Contains(err.Error(), "cannot read") {
		t.Fatalf("missing file: want a read error, got %v", err)
	}
}
