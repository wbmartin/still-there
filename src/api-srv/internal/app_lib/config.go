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
	ShrmplLogDest         string `json:"shrmpl_log_dest"`
	ShrmplKVHost          string `json:"shrmpl_kv_host"`
	ShrmplKVPort          int    `json:"shrmpl_kv_port"`
	ServerName            string `json:"server_name"`
	ServerPort            int    `json:"server_port"`
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
	if config.ShrmplLogDest == "" {
		return nil, fmt.Errorf("shrmpl_log_dest is required")
	}
	if config.ServerName == "" {
		return nil, fmt.Errorf("server_name is required")
	}

	return &config, nil
}
