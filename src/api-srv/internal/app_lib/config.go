package app_lib

import (
	"encoding/json"
	"fmt"
	"os"
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

// LoadConfig loads configuration from a JSON file
func LoadConfig(configFilePath string) (*Config, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("unable to parse configuration file: %w", err)
	}

	// Validate required configuration
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("database_url is required")
	}
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("jwt_secret is required")
	}
	if config.LogReceiverHostPort == "" {
		return nil, fmt.Errorf("log_receiver_host_port is required")
	}
	if config.KVHostPort == "" {
		return nil, fmt.Errorf("kv_host_port is required")
	}
	if config.ServerName == "" {
		return nil, fmt.Errorf("server_name is required")
	}
	if config.TLSCertPath == "" {
		return nil, fmt.Errorf("tls_cert_path is required")
	}
	if config.TLSKeyPath == "" {
		return nil, fmt.Errorf("tls_key_path is required")
	}

	return &config, nil
}
