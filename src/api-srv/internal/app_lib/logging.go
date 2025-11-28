package app_lib

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ThisAppLoggerInterface defines the logging interface for this application
type ThisAppLoggerInterface interface {
	Debug(host, code, message string)
	Info(host, code, message string)
	Warn(host, code, message string)
	Error(host, code, message string)
}

// Logger wraps shrmpl-log client for structured logging
type Logger struct {
	shrmplLog *ShrmplLogClient
	service   string
}

// NewLogger creates a logger that uses shrmpl-log
func NewLogger(config *Config) *Logger {
	// Create shrmpl-log client internally
	shrmplLog, err := NewShrmplLogClient(config.LogReceiverHostPort)
	if err != nil {
		// If we can't create the client, we'll log to console and continue
		// The Log method will handle the case where shrmplLog is nil
		fmt.Fprintf(os.Stderr, "Failed to create shrmpl-log client: %s\n", err.Error())
		return &Logger{
			shrmplLog: nil,
			service:   config.ServerName,
		}
	}

	if err := shrmplLog.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to shrmpl-log: %s\n", err.Error())
		return &Logger{
			shrmplLog: nil,
			service:   config.ServerName,
		}
	}

	return &Logger{
		shrmplLog: shrmplLog,
		service:   config.ServerName,
	}
}

// log sends a log message to shrmpl-log with caller information
func (l *Logger) log(level string, msg string, args ...any) {
	// Format message with structured args
	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}

	// Add caller information (skip 2 frames: log() -> Debug/Info/Warn/Error() -> caller)
	_, file, line, ok := runtime.Caller(2)
	callerInfo := ""
	if ok {
		// Extract just the filename from the full path
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]
		callerInfo = fmt.Sprintf(" %s:%d", filename, line)
	}

	// Append caller info to message
	fullMessage := formattedMsg + callerInfo

	// Send to shrmpl-log (ignore errors - if it fails, logging fails)
	_ = l.shrmplLog.Log(level, l.service, "0000", fullMessage)
}

// Debug logs at debug level
func (l *Logger) Debug(host, code, message string) {
	l.log("DEBG", message)
}

// Info logs at info level
func (l *Logger) Info(host, code, message string) {
	l.log("INFO", message)
}

// Warn logs at warn level
func (l *Logger) Warn(host, code, message string) {
	l.log("WARN", message)
}

// Error logs at error level
func (l *Logger) Error(host, code, message string) {
	l.log("ERRO", message)
}

// ShrmplLogClient represents a client for the shrmpl-log service
type ShrmplLogClient struct {
	host string
	port int
	conn net.Conn
}

// NewShrmplLogClient creates a new shrmpl-log client
func NewShrmplLogClient(logDest string) (*ShrmplLogClient, error) {
	parts := strings.Split(logDest, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid log destination format: %s", logDest)
	}

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port in log destination: %w", err)
	}

	return &ShrmplLogClient{
		host: host,
		port: port,
	}, nil
}

// Connect establishes connection to shrmpl-log
func (c *ShrmplLogClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to shrmpl-log: %w", err)
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	c.conn = conn
	return nil
}

// Log sends a log message to shrmpl-log
func (c *ShrmplLogClient) Log(level, host, code, message string) error {
	// Validate inputs
	if len(level) != 4 {
		return fmt.Errorf("level must be exactly 4 characters")
	}
	if len(host) > 32 {
		return fmt.Errorf("host must be <= 32 characters")
	}
	if len(code) != 4 {
		return fmt.Errorf("code must be exactly 4 characters")
	}
	if len(message) > 4096 {
		return fmt.Errorf("message must be <= 4096 characters")
	}

	// Format: [LVL(4)] [HOST(32)] [CODE(4)] [LEN(4)]: [MSG]\n
	paddedHost := fmt.Sprintf("%-32s", host[:min(32, len(host))])
	paddedLevel := fmt.Sprintf("%-4s", level[:4])
	paddedCode := fmt.Sprintf("%-4s", code[:4])
	msgLen := fmt.Sprintf("%04d", len(message))

	logLine := fmt.Sprintf("[%s] [%s] [%s] [%s]: %s\n", paddedLevel, paddedHost, paddedCode, msgLen, message)

	_, err := c.conn.Write([]byte(logLine))
	return err
}

// Close closes the connection to shrmpl-log
func (c *ShrmplLogClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
