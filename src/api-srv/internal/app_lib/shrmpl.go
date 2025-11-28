package app_lib

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ShrmplKVClient represents a client for the shrmpl-kv service
type ShrmplKVClient struct {
	host string
	port int
	conn net.Conn
}

// ShrmplLogClient represents a client for the shrmpl-log service
type ShrmplLogClient struct {
	host string
	port int
	conn net.Conn
}

// NewShrmplKVClient creates a new shrmpl-kv client
func NewShrmplKVClient(host string, port int) *ShrmplKVClient {
	return &ShrmplKVClient{
		host: host,
		port: port,
	}
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

// Connect establishes connection to shrmpl-kv
func (c *ShrmplKVClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to shrmpl-kv: %w", err)
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	c.conn = conn
	return nil
}

// Get retrieves a value from shrmpl-kv
func (c *ShrmplKVClient) Get(key string) (string, error) {
	if len(key) > 100 {
		return "", fmt.Errorf("key length exceeds 100 characters")
	}

	response, err := c.sendCommand(fmt.Sprintf("GET %s", key))
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(response, "ERROR") {
		return "", fmt.Errorf(response)
	}

	return response, nil
}

// Set stores a key-value pair in shrmpl-kv
func (c *ShrmplKVClient) Set(key, value string, ttl string) error {
	if len(key) > 100 || len(value) > 100 {
		return fmt.Errorf("key or value length exceeds 100 characters")
	}

	var cmd string
	if ttl != "" {
		cmd = fmt.Sprintf("SET %s %s %s", key, value, ttl)
	} else {
		cmd = fmt.Sprintf("SET %s %s", key, value)
	}

	response, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if response != "OK" {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}

// Incr increments a counter in shrmpl-kv
func (c *ShrmplKVClient) Incr(key string, ttl string) (int, error) {
	if len(key) > 100 {
		return 0, fmt.Errorf("key length exceeds 100 characters")
	}

	var cmd string
	if ttl != "" {
		cmd = fmt.Sprintf("INCR %s %s", key, ttl)
	} else {
		cmd = fmt.Sprintf("INCR %s", key)
	}

	response, err := c.sendCommand(cmd)
	if err != nil {
		return 0, err
	}

	if strings.HasPrefix(response, "ERROR") {
		return 0, fmt.Errorf(response)
	}

	result, err := strconv.Atoi(response)
	if err != nil {
		return 0, fmt.Errorf("invalid response: %s", response)
	}

	return result, nil
}

// Ping checks if shrmpl-kv is responsive
func (c *ShrmplKVClient) Ping() error {
	response, err := c.sendCommand("PING")
	if err != nil {
		return err
	}

	if response != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", response)
	}

	return nil
}

// Close closes the connection to shrmpl-kv
func (c *ShrmplKVClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// sendCommand sends a command and returns the response
func (c *ShrmplKVClient) sendCommand(cmd string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	_, err := c.conn.Write([]byte(cmd + "\n"))
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(c.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	response = strings.TrimSpace(response)

	// Filter out heartbeats
	if response == "UPONG" {
		return "", fmt.Errorf("heartbeat received")
	}
	if response == "TERM" {
		return "", fmt.Errorf("server shutting down")
	}

	return response, nil
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
