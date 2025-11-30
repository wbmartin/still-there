package app_lib

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ThisAppKVInterface defines the key-value store interface for this application
type ThisAppKVInterface interface {
	Get(key string) (string, error)
	Set(key, value, ttl string) error
	Incr(key, ttl string) (int, error)
	Close()
}

// KV wraps shrmpl-kv client for key-value operations
type KV struct {
	shrmplKVClient *ShrmplKVClient
	hostPort       string
}

// parseHostPort parses a "host:port" string into separate host and port components
func parseHostPort(hostPort string) (string, string, error) {
	parts := strings.Split(hostPort, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid host:port format: %s", hostPort)
	}
	return parts[0], parts[1], nil
}

// NewKV creates a key-value store client
func NewKV(config *Config) ThisAppKVInterface {
	// Parse the combined host:port string
	host, portStr, err := parseHostPort(config.KVHostPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse kv_host_port: %s\n", err.Error())
		return &KV{shrmplKVClient: nil, hostPort: config.KVHostPort}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid port in kv_host_port: %s\n", err.Error())
		return &KV{shrmplKVClient: nil, hostPort: config.KVHostPort}
	}

	shrmplKV := NewShrmplKVClient(host, port)
	if err := shrmplKV.Connect(); err != nil {
		// If we can't connect, we'll return a client that logs errors
		// The operations will fail gracefully
		fmt.Fprintf(os.Stderr, "Failed to connect to shrmpl-kv: %s\n", err.Error())
		return &KV{shrmplKVClient: nil, hostPort: config.KVHostPort}
	}

	return &KV{
		shrmplKVClient: shrmplKV,
		hostPort:       config.KVHostPort,
	}
}

// tryReconnect attempts to reconnect to the KV server
func (kv *KV) tryReconnect() {
	host, portStr, err := parseHostPort(kv.hostPort)
	if err != nil {
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return
	}
	client := NewShrmplKVClient(host, port)
	if err := client.Connect(); err == nil {
		kv.shrmplKVClient = client
	}
}

// Get retrieves a value from the key-value store
func (kv *KV) Get(key string) (string, error) {
	if kv.shrmplKVClient == nil {
		kv.tryReconnect()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		return "", fmt.Errorf("key-value store not available")
	}
	val, err := kv.shrmplKVClient.Get(key)
	if err != nil {
		kv.shrmplKVClient.Close()
		kv.shrmplKVClient = nil
		return "", err
	}
	return val, nil
}

// Set stores a key-value pair with optional TTL
func (kv *KV) Set(key, value, ttl string) error {
	if kv.shrmplKVClient == nil {
		kv.tryReconnect()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		return fmt.Errorf("key-value store not available")
	}
	err := kv.shrmplKVClient.Set(key, value, ttl)
	if err != nil {
		kv.shrmplKVClient.Close()
		kv.shrmplKVClient = nil
		return err
	}
	return nil
}

// Incr increments a counter and returns the new value
func (kv *KV) Incr(key string, ttl string) (int, error) {
	if kv.shrmplKVClient == nil {
		kv.tryReconnect()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		return 0, fmt.Errorf("key-value store not available")
	}
	val, err := kv.shrmplKVClient.Incr(key, ttl)
	if err != nil {
		kv.shrmplKVClient.Close()
		kv.shrmplKVClient = nil
		return 0, err
	}
	return val, nil
}

// ShrmplKVClient represents a client for the shrmpl-kv service
type ShrmplKVClient struct {
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

// Close closes the underlying KV client connection
func (kv *KV) Close() {
	if kv.shrmplKVClient != nil {
		kv.shrmplKVClient.Close()
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
