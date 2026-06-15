package app_lib

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL                      string   `json:"database_url"`
	JWTSecret                        string   `json:"jwt_secret"`
	JWTSessionExpireHours            int      `json:"jwt_session_expire_hours"`
	MaxUserReqPerMin                 int      `json:"max_user_req_per_min"`
	MaxUnauthReqPerMin               int      `json:"max_unauth_req_per_min"`
	LogReceiverHostPort              string   `json:"log_receiver_host_port"`
	KVHostPort                       string   `json:"kv_host_port"`
	KVBackend                        string   `json:"kv_backend"`
	ServerName                       string   `json:"server_name"`
	AppName                          string   `json:"app_name"`
	ServerPort                       int      `json:"server_port"`
	EnableTLS                        bool     `json:"enable_tls"`
	TLSCertPath                      string   `json:"tls_cert_path"`
	TLSKeyPath                       string   `json:"tls_key_path"`
	TOTPEncryptionKey                string   `json:"totp_encryption_key"`
	FailedLoginAttemptThreshold      int      `json:"failed_login_attempt_threshold"`
	FailedLoginAttemptWindowDuration string   `json:"failed_login_attempt_window_duration"`
	FailedLoginLockoutDuration       string   `json:"failed_login_lockout_duration"`
	CorsAllowedOrigins               []string `json:"cors_allowed_origins"`
}

// VaultClient represents a client for the shrmpl-vault service
type VaultClient struct {
	serverURL string
	certPath  string
	keyPath   string
	secret    string
	client    *http.Client
}

// NewVaultClient creates a new vault client
func NewVaultClient(serverURL, certPath, keyPath, secret string) *VaultClient {
	return &VaultClient{
		serverURL: strings.TrimRight(serverURL, "/"),
		certPath:  certPath,
		keyPath:   keyPath,
		secret:    secret,
	}
}

// Connect establishes TLS connection to shrmpl-vault
func (c *VaultClient) Connect() (bool, error) {
	// Load client certificates
	cert, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
	if err != nil {
		return false, fmt.Errorf("failed to load certificates: %v", err)
	}

	// Load vault server certificate for verification
	serverCertPath := "/Users/brandon/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_fullchain.pem"
	serverCertPEM, err := os.ReadFile(serverCertPath)
	if err != nil {
		return false, fmt.Errorf("failed to load server certificate: %v", err)
	}
	block, _ := pem.Decode(serverCertPEM)
	if block == nil {
		return false, fmt.Errorf(
			"failed to decode PEM block from server certificate",
		)
	}
	serverCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse server certificate: %v", err)
	}

	// Create cert pool with server cert as CA (assuming self-signed, will break otherwise)
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(serverCert)

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	// Create HTTP client
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	c.client = &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Connection setup successful - actual testing happens during GetConfig calls
	return true, nil
}

// GetConfig retrieves a configuration file from shrmpl-vault
func (c *VaultClient) GetConfig(filename string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("not connected")
	}

	url := fmt.Sprintf("%s/%s?secret=%s", c.serverURL, filename, c.secret)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		content, err := io.ReadAll(resp.Body)
		return string(content), err
	case 404:
		return "", fmt.Errorf("file not found")
	case 401:
		return "", fmt.Errorf("unauthorized - invalid certificate or secret")
	case 429:
		return "", fmt.Errorf("rate limit exceeded")
	default:
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
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

	// Fetch secrets from vault if configured
	vaultServer := env["VAULT_SERVER"]
	vaultCert := env["VAULT_CLIENT_CERT_PATH"]
	vaultKey := env["VAULT_CLIENT_KEY_PATH"]
	vaultSecret := env["VAULT_SECRET_KEY"]
	vaultFile := env["VAULT_SECRETS_FILENAME"]
	if vaultServer != "" && vaultCert != "" && vaultKey != "" &&
		vaultSecret != "" &&
		vaultFile != "" {
		vaultClient := NewVaultClient(
			vaultServer,
			vaultCert,
			vaultKey,
			vaultSecret,
		)
		_, err := vaultClient.Connect()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to vault: %w", err)
		}
		secretsJSON, err := vaultClient.GetConfig(vaultFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets from vault: %w", err)
		}
		var secrets map[string]string
		if err := json.Unmarshal([]byte(secretsJSON), &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secrets JSON: %w", err)
		}
		// Override env with secrets
		if dbURL, ok := secrets["DATABASE_URL"]; ok {
			env["DATABASE_URL"] = dbURL
		}
		if jwtSecret, ok := secrets["JWT_SECRET"]; ok {
			env["JWT_SECRET"] = jwtSecret
		}
		if totpKey, ok := secrets["TOTP_ENCRYPTION_KEY"]; ok {
			env["TOTP_ENCRYPTION_KEY"] = totpKey
		}
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
	config.KVBackend = env["KV_BACKEND"]
	if config.KVBackend == "" {
		config.KVBackend = "memory" // default: in-process KV (no external shrmpl-kv needed)
	}
	config.ServerName = env["SERVER_NAME"]
	config.AppName = env["APP_NAME"]

	if serverPortStr := env["SERVER_PORT"]; serverPortStr != "" {
		if val, err := strconv.Atoi(serverPortStr); err == nil {
			config.ServerPort = val
		} else {
			return nil, fmt.Errorf("SERVER_PORT must be a valid integer")
		}
	} else {
		return nil, fmt.Errorf("SERVER_PORT is required")
	}

	if enableTLSStr := env["ENABLE_TLS"]; enableTLSStr != "" {
		if val, err := strconv.ParseBool(enableTLSStr); err == nil {
			config.EnableTLS = val
		} else {
			return nil, fmt.Errorf("ENABLE_TLS must be a valid boolean (true/false)")
		}
	} else {
		return nil, fmt.Errorf("ENABLE_TLS is required")
	}

	config.TLSCertPath = env["TLS_CERT_PATH"]
	config.TLSKeyPath = env["TLS_KEY_PATH"]
	config.TOTPEncryptionKey = env["TOTP_ENCRYPTION_KEY"]

	if thresholdStr := env["FAILED_LOGIN_ATTEMPT_THRESHOLD"]; thresholdStr != "" {
		if val, err := strconv.Atoi(thresholdStr); err == nil {
			config.FailedLoginAttemptThreshold = val
		} else {
			return nil, fmt.Errorf("FAILED_LOGIN_ATTEMPT_THRESHOLD must be a valid integer")
		}
	} else {
		return nil, fmt.Errorf("FAILED_LOGIN_ATTEMPT_THRESHOLD is required")
	}

	config.FailedLoginAttemptWindowDuration = env["FAILED_LOGIN_ATTEMPT_WINDOW_DURATION"]
	if config.FailedLoginAttemptWindowDuration == "" {
		return nil, fmt.Errorf(
			"FAILED_LOGIN_ATTEMPT_WINDOW_DURATION is required",
		)
	}

	config.FailedLoginLockoutDuration = env["FAILED_LOGIN_LOCKOUT_DURATION"]
	if config.FailedLoginLockoutDuration == "" {
		return nil, fmt.Errorf("FAILED_LOGIN_LOCKOUT_DURATION is required")
	}

	if corsOriginsStr := env["CORS_ALLOWED_ORIGINS"]; corsOriginsStr != "" {
		origins := strings.Split(corsOriginsStr, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		config.CorsAllowedOrigins = origins
	} else {
		config.CorsAllowedOrigins = []string{} // default to no CORS
	}

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
	switch config.KVBackend {
	case "memory":
		// in-process KV; KV_HOST_PORT not needed
	case "shrmpl":
		if config.KVHostPort == "" {
			return nil, fmt.Errorf("KV_HOST_PORT is required when KV_BACKEND=shrmpl")
		}
	default:
		return nil, fmt.Errorf("KV_BACKEND must be \"memory\" or \"shrmpl\", got %q", config.KVBackend)
	}
	if config.ServerName == "" {
		return nil, fmt.Errorf("SERVER_NAME is required")
	}
	if config.AppName == "" {
		return nil, fmt.Errorf("APP_NAME is required")
	}
	if config.EnableTLS {
		if config.TLSCertPath == "" {
			return nil, fmt.Errorf(
				"TLS_CERT_PATH is required when ENABLE_TLS is true",
			)
		}
		if config.TLSKeyPath == "" {
			return nil, fmt.Errorf(
				"TLS_KEY_PATH is required when ENABLE_TLS is true",
			)
		}
	}
	if config.TOTPEncryptionKey == "" {
		return nil, fmt.Errorf("TOTP_ENCRYPTION_KEY is required")
	}

	if config.FailedLoginAttemptThreshold <= 0 {
		return nil, fmt.Errorf(
			"FAILED_LOGIN_ATTEMPT_THRESHOLD must be greater than 0",
		)
	}

	if config.ServerPort <= 0 {
		return nil, fmt.Errorf("SERVER_PORT must be greater than 0")
	}

	if config.FailedLoginAttemptWindowDuration == "" {
		return nil, fmt.Errorf(
			"FAILED_LOGIN_ATTEMPT_WINDOW_DURATION is required",
		)
	}

	if config.FailedLoginLockoutDuration == "" {
		return nil, fmt.Errorf("FAILED_LOGIN_LOCKOUT_DURATION is required")
	}

	return &config, nil
}
