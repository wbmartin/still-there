package app_lib

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL           string `json:"database_url"`
	JWTSecret             string `json:"jwt_secret"`
	JWTSessionExpireHours int    `json:"jwt_session_expire_hours"`
	MaxUserReqPerMin      int    `json:"max_user_req_per_min"`
	MaxUnauthReqPerMin    int    `json:"max_unauth_req_per_min"`
	LogReceiverHostPort   string `json:"log_receiver_host_port"`
	KVHostPort            string `json:"kv_host_port"`
	ServerName            string `json:"server_name"`
	ServerPort            int    `json:"server_port"`
	TLSCertPath           string `json:"tls_cert_path"`
	TLSKeyPath            string `json:"tls_key_path"`
}

// LoadConfig loads configuration from a .env file
func LoadConfig(configFilePath string) (*Config, error) {
	// Open the .env file
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()

	// Parse the .env file
	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			env[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	var config Config

	// Populate from parsed env map
	config.DatabaseURL = env["DATABASE_URL"]
	config.JWTSecret = env["JWT_SECRET"]

	if jwtHoursStr := env["JWT_SESSION_EXPIRE_HOURS"]; jwtHoursStr != "" {
		if val, err := strconv.Atoi(jwtHoursStr); err == nil {
			config.JWTSessionExpireHours = val
		}
	}

	if maxUserStr := env["MAX_USER_REQ_PER_MIN"]; maxUserStr != "" {
		if val, err := strconv.Atoi(maxUserStr); err == nil {
			config.MaxUserReqPerMin = val
		}
	}

	if maxUnauthStr := env["MAX_UNAUTH_REQ_PER_MIN"]; maxUnauthStr != "" {
		if val, err := strconv.Atoi(maxUnauthStr); err == nil {
			config.MaxUnauthReqPerMin = val
		}
	}

	config.LogReceiverHostPort = env["LOG_RECEIVER_HOST_PORT"]
	config.KVHostPort = env["KV_HOST_PORT"]
	config.ServerName = env["SERVER_NAME"]

	if serverPortStr := env["SERVER_PORT"]; serverPortStr != "" {
		if val, err := strconv.Atoi(serverPortStr); err == nil {
			config.ServerPort = val
		}
	}

	config.TLSCertPath = env["TLS_CERT_PATH"]
	config.TLSKeyPath = env["TLS_KEY_PATH"]

	// Validate required configuration
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if config.LogReceiverHostPort == "" {
		return nil, fmt.Errorf("LOG_RECEIVER_HOST_PORT is required")
	}
	if config.KVHostPort == "" {
		return nil, fmt.Errorf("KV_HOST_PORT is required")
	}
	if config.ServerName == "" {
		return nil, fmt.Errorf("SERVER_NAME is required")
	}
	if config.TLSCertPath == "" {
		return nil, fmt.Errorf("TLS_CERT_PATH is required")
	}
	if config.TLSKeyPath == "" {
		return nil, fmt.Errorf("TLS_KEY_PATH is required")
	}

	return &config, nil
}
