package app_lib

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ThisAppKVInterface defines the key-value store interface for this application
type ThisAppKVInterface interface {
	Get(key string) (string, error)
	Set(key, value, ttl string) error
	Incr(key string, ttl string) (int, error)
	Batch(commands []string) ([]string, error)
	Close()
}

// KV wraps shrmpl-kv client for key-value operations
type KV struct {
	shrmplKVClient *ShrmplKVClient
	hostPort       string
	mu             sync.Mutex
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
	kv.mu.Lock()
	defer kv.mu.Unlock()
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
	kv.mu.Lock()
	if kv.shrmplKVClient == nil {
		kv.mu.Unlock()
		kv.tryReconnect()
		kv.mu.Lock()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		kv.mu.Unlock()
		return "", fmt.Errorf("key-value store not available")
	}
	client := kv.shrmplKVClient
	kv.mu.Unlock()
	val, err := client.Get(key)
	if err != nil {
		kv.mu.Lock()
		if kv.shrmplKVClient == client {
			kv.shrmplKVClient.Close()
			kv.shrmplKVClient = nil
		}
		kv.mu.Unlock()
		return "", err
	}
	return val, nil
}

// Set stores a key-value pair with optional TTL
func (kv *KV) Set(key, value, ttl string) error {
	kv.mu.Lock()
	if kv.shrmplKVClient == nil {
		kv.mu.Unlock()
		kv.tryReconnect()
		kv.mu.Lock()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		kv.mu.Unlock()
		return fmt.Errorf("key-value store not available")
	}
	client := kv.shrmplKVClient
	kv.mu.Unlock()
	err := client.Set(key, value, ttl)
	if err != nil {
		kv.mu.Lock()
		if kv.shrmplKVClient == client {
			kv.shrmplKVClient.Close()
			kv.shrmplKVClient = nil
		}
		kv.mu.Unlock()
		return err
	}
	return nil
}

// Incr increments a counter and returns the new value
func (kv *KV) Incr(key string, ttl string) (int, error) {
	kv.mu.Lock()
	if kv.shrmplKVClient == nil {
		kv.mu.Unlock()
		kv.tryReconnect()
		kv.mu.Lock()
	}
	if kv.shrmplKVClient == nil { //try again after reconnect
		kv.mu.Unlock()
		return 0, fmt.Errorf("key-value store not available")
	}
	client := kv.shrmplKVClient
	kv.mu.Unlock()
	val, err := client.Incr(key, ttl)
	if err != nil {
		kv.mu.Lock()
		if kv.shrmplKVClient == client {
			kv.shrmplKVClient.Close()
			kv.shrmplKVClient = nil
		}
		kv.mu.Unlock()
		return 0, err
	}
	return val, nil
}

// Batch executes multiple commands in a single call
func (kv *KV) Batch(commands []string) ([]string, error) {
	if len(commands) > 3 {
		return nil, fmt.Errorf("batch cannot exceed 3 commands")
	}
	kv.mu.Lock()
	if kv.shrmplKVClient == nil {
		kv.mu.Unlock()
		kv.tryReconnect()
		kv.mu.Lock()
	}
	if kv.shrmplKVClient == nil {
		kv.mu.Unlock()
		return nil, fmt.Errorf("key-value store not available")
	}
	client := kv.shrmplKVClient
	kv.mu.Unlock()

	batchCmd := "BATCH " + strings.Join(commands, ";")
	fmt.Printf("DEBUG: Sending batchCmd: %s\n", batchCmd)
	response, err := client.sendCommand(batchCmd)
	fmt.Printf("DEBUG: Received response: %s\n", response)
	if err != nil {
		kv.mu.Lock()
		if kv.shrmplKVClient == client {
			kv.shrmplKVClient.Close()
			kv.shrmplKVClient = nil
		}
		kv.mu.Unlock()
		return nil, err
	}

	if strings.HasPrefix(response, "ERROR") {
		return nil, errors.New(response)
	}

	results := strings.Split(strings.TrimSpace(response), ";")
	return results, nil
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
		return "", errors.New(response)
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
		return 0, errors.New(response)
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
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
	c.conn = nil
}

// Close closes the underlying KV client connection
func (kv *KV) Close() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shrmplKVClient != nil {
		kv.shrmplKVClient.Close()
		kv.shrmplKVClient = nil
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
	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		response = strings.TrimSpace(response)

		// Skip heartbeats
		if response == "UPONG" {
			continue
		}
		if response == "TERM" {
			return "", fmt.Errorf("server shutting down")
		}

		return response, nil
	}
}
