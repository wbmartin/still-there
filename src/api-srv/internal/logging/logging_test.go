package logging

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"testing"
)

// viaWrapper stands in for AppHelper.Info: one frame between the call site
// and the Logger, so skip=3 must land on viaWrapper's caller.
func viaWrapper(l *Logger) {
	l.InfoWithCallerSkip("WRAP", "through a wrapper", 3, "username", "bob")
}

func TestLogger_CodeAttrAndSource(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger("test-svc", &buf, slog.LevelDebug)

	_, _, here, _ := runtime.Caller(0)
	l.Info("EX_CR_START", "example created", "username", "alice", "exampleId", 7)
	line := buf.String()
	buf.Reset()

	for _, want := range []string{
		"level=INFO", `msg="example created"`, "code=EX_CR_START",
		"service=test-svc", "username=alice", "exampleId=7",
		fmt.Sprintf("logging_test.go:%d", here+1),
	} {
		if !strings.Contains(line, want) {
			t.Errorf("log line missing %q:\n%s", want, line)
		}
	}

	_, _, here, _ = runtime.Caller(0)
	viaWrapper(l)
	line = buf.String()
	if !strings.Contains(line, fmt.Sprintf("logging_test.go:%d", here+1)) ||
		!strings.Contains(line, "code=WRAP") {
		t.Errorf("wrapper skip did not attribute to the call site:\n%s", line)
	}
}

func TestLogger_DebugLevelWritesEveryLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger("svc", &buf, slog.LevelDebug)
	l.Debug("D", "d")
	l.Warn("W", "w")
	l.Error("E", "e")
	out := buf.String()
	for _, want := range []string{
		"level=DEBUG", "code=D", "level=WARN", "code=W",
		"level=ERROR", "code=E",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestLogger_InfoLevelDropsDebug(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger("svc", &buf, slog.LevelInfo)
	l.Debug("D", "d")
	l.DebugWithCallerSkip("DS", "ds", 2)
	l.Info("I", "i")
	l.Warn("W", "w")
	out := buf.String()
	if strings.Contains(out, "level=DEBUG") {
		t.Errorf("debug line written at info:\n%s", out)
	}
	for _, want := range []string{"code=I", "code=W"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestParseLevel(t *testing.T) {
	wantByName := map[string]slog.Level{
		"debug": slog.LevelDebug, "info": slog.LevelInfo,
		"warn": slog.LevelWarn, "error": slog.LevelError,
	}
	for name, want := range wantByName {
		got, ok := ParseLevel(name)
		if !ok || got != want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v, true",
				name, got, ok, want)
		}
	}
	for _, name := range []string{"", "INFO", "verbose", "warning", " info"} {
		if _, ok := ParseLevel(name); ok {
			t.Errorf("ParseLevel(%q) accepted", name)
		}
	}
}
