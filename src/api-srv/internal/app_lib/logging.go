package app_lib

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Logger is a thin wrapper over log/slog that keeps the call-site shape the
// codebase has always used — Debug/Info/Warn/Error(code, message, kv...) —
// and turns the code into a searchable `code=` attribute. Output is one
// logfmt line per call on stderr; systemd captures it into journald. There
// is deliberately no remote receiver and no pluggable backend (PS0003).
//
// The WithCallerSkip variants exist so that AppHelper's wrappers report the
// handler line, not the wrapper's.
type Logger struct {
	h slog.Handler
}

// NewLogger builds a Logger that writes every level to w, tagged with the
// server name. Pass os.Stderr in main; tests can pass a buffer.
func NewLogger(serverName string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stderr
	}
	h := slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		// source=file.go:NN rather than the build machine's absolute path
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					return slog.String(slog.SourceKey,
						fmt.Sprintf("%s:%d", filepath.Base(src.File), src.Line))
				}
			}
			return a
		},
	})
	return &Logger{h: h.WithAttrs([]slog.Attr{slog.String("service", serverName)})}
}

// log emits one record. skip is a runtime.Caller-style frame count with log
// itself at 0, so 1 is log's caller; the WithCallerSkip methods pass their
// argument straight through, which is the contract every call site was
// written against.
func (l *Logger) log(level slog.Level, code, message string, skip int, keyvals ...interface{}) {
	var pcs [1]uintptr
	// +1 because runtime.Callers counts itself where runtime.Caller does not.
	runtime.Callers(skip+1, pcs[:])
	r := slog.NewRecord(time.Now(), level, message, pcs[0])
	r.AddAttrs(slog.String("code", code))
	r.Add(keyvals...)
	_ = l.h.Handle(context.Background(), r)
}

// Debug logs at debug level
func (l *Logger) Debug(code, message string, keyvals ...interface{}) {
	l.log(slog.LevelDebug, code, message, 2, keyvals...)
}

// Info logs at info level
func (l *Logger) Info(code, message string, keyvals ...interface{}) {
	l.log(slog.LevelInfo, code, message, 2, keyvals...)
}

// Warn logs at warn level
func (l *Logger) Warn(code, message string, keyvals ...interface{}) {
	l.log(slog.LevelWarn, code, message, 2, keyvals...)
}

// Error logs at error level
func (l *Logger) Error(code, message string, keyvals ...interface{}) {
	l.log(slog.LevelError, code, message, 2, keyvals...)
}

// DebugWithCallerSkip logs at debug level, attributing the line to the frame
// skip levels above Logger.log (2 = the caller of this method).
func (l *Logger) DebugWithCallerSkip(code, message string, skip int, keyvals ...interface{}) {
	l.log(slog.LevelDebug, code, message, skip, keyvals...)
}

// InfoWithCallerSkip logs at info level with custom caller skip
func (l *Logger) InfoWithCallerSkip(code, message string, skip int, keyvals ...interface{}) {
	l.log(slog.LevelInfo, code, message, skip, keyvals...)
}

// WarnWithCallerSkip logs at warn level with custom caller skip
func (l *Logger) WarnWithCallerSkip(code, message string, skip int, keyvals ...interface{}) {
	l.log(slog.LevelWarn, code, message, skip, keyvals...)
}

// ErrorWithCallerSkip logs at error level with custom caller skip
func (l *Logger) ErrorWithCallerSkip(code, message string, skip int, keyvals ...interface{}) {
	l.log(slog.LevelError, code, message, skip, keyvals...)
}
