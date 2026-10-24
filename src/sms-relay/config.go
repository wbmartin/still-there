package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Config is the relay's one env file: flat KEY=VALUE, blank lines and #
// comments ignored, every key required, no defaults. TelnyxPublicKey is
// public; the consumer tokens are held only as SHA-256 hashes, so the file
// carries no secret but the database password.
type Config struct {
	DatabaseURL       string
	ServerBindAddress string
	ServerPort        int
	TelnyxPublicKey   string
	ConsumerTokenList []ConsumerToken
	RetainDays        int
	LogLevel          slog.Level
}

// ConsumerToken lets the holder of the token whose SHA-256 is Hash read
// Queue, and no other queue.
type ConsumerToken struct {
	Queue string
	Hash  [32]byte
}

// queueNamePattern is a queue's name in a URL path and in
// CONSUMER_TOKEN_LIST.
var queueNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// readEnvFile parses a KEY=VALUE file. A value in matching double or
// single quotes loses them; anything else is taken verbatim after trimming.
func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()
	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') &&
			value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		env[strings.TrimSpace(key)] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}
	return env, nil
}

// loadDatabaseURL reads only DATABASE_URL, for the migrate subcommand,
// whose file holds the DDL role's credentials.
func loadDatabaseURL(path string) (string, error) {
	env, err := readEnvFile(path)
	if err != nil {
		return "", err
	}
	if env["DATABASE_URL"] == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}
	return env["DATABASE_URL"], nil
}

func loadConfig(path string) (*Config, error) {
	env, err := readEnvFile(path)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{
		"DATABASE_URL", "SERVER_BIND_ADDRESS", "SERVER_PORT",
		"TELNYX_PUBLIC_KEY", "CONSUMER_TOKEN_LIST", "RETAIN_DAYS", "LOG_LEVEL",
	} {
		if env[key] == "" {
			return nil, fmt.Errorf("%s is required", key)
		}
	}
	config := &Config{
		DatabaseURL:       env["DATABASE_URL"],
		ServerBindAddress: env["SERVER_BIND_ADDRESS"],
		TelnyxPublicKey:   env["TELNYX_PUBLIC_KEY"],
	}
	if config.ServerPort, err = positiveInt(env, "SERVER_PORT"); err != nil {
		return nil, err
	}
	if config.RetainDays, err = positiveInt(env, "RETAIN_DAYS"); err != nil {
		return nil, err
	}
	// Checked here, so a bad key fails start-up rather than every webhook.
	if _, err := parsePublicKey(config.TelnyxPublicKey); err != nil {
		return nil, err
	}
	if config.ConsumerTokenList, err = parseConsumerTokenList(
		env["CONSUMER_TOKEN_LIST"],
	); err != nil {
		return nil, err
	}
	if err := config.LogLevel.UnmarshalText(
		[]byte(env["LOG_LEVEL"]),
	); err != nil {
		return nil, fmt.Errorf(
			"LOG_LEVEL must be debug, info, warn or error, got %q",
			env["LOG_LEVEL"],
		)
	}
	return config, nil
}

func positiveInt(env map[string]string, key string) (int, error) {
	value, err := strconv.Atoi(env[key])
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be a positive integer, got %q", key, env[key],
		)
	}
	return value, nil
}

// parseConsumerTokenList reads "<queue>:<sha256 hex>,..."; a queue may
// appear more than once (two tokens while one is rotated out).
func parseConsumerTokenList(raw string) ([]ConsumerToken, error) {
	var tokenList []ConsumerToken
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		queue, hashHex, ok := strings.Cut(entry, ":")
		if !ok || !queueNamePattern.MatchString(queue) {
			return nil, fmt.Errorf(
				"CONSUMER_TOKEN_LIST: %q is not <queue>:<sha256 hex>", entry,
			)
		}
		hash, err := hex.DecodeString(hashHex)
		if err != nil || len(hash) != 32 {
			return nil, fmt.Errorf(
				"CONSUMER_TOKEN_LIST: the hash for queue %q is not 64 hex "+
					"characters (sha256sum of the token)", queue,
			)
		}
		token := ConsumerToken{Queue: queue}
		copy(token.Hash[:], hash)
		tokenList = append(tokenList, token)
	}
	return tokenList, nil
}
