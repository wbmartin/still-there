package logging

import (
	"bytes"
	"fmt"
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
	l := NewLogger("test-svc", &buf)

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

func TestLogger_LevelsAndDebugNotDropped(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger("svc", &buf)
	l.Debug("D", "d")
	l.Warn("W", "w")
	l.Error("E", "e")
	out := buf.String()
	for _, want := range []string{"level=DEBUG", "code=D", "level=WARN", "code=W", "level=ERROR", "code=E"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
