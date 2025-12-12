package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryConfig holds configuration for database operation retries
type RetryConfig struct {
	MaxRetries      uint64
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: time.Second,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
	}
}

// isRetryableError determines if a database error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Retry on connection-related errors
	if err == sql.ErrConnDone || err == sql.ErrTxDone {
		return true
	}

	// Check for common PostgreSQL connection errors
	errStr := err.Error()
	return errStr == "connection refused" ||
		errStr == "connection reset by peer" ||
		errStr == "broken pipe" ||
		errStr == "i/o timeout"
}

// ExecuteWithRetry executes a database operation with exponential backoff retry
func ExecuteWithRetry(
	ctx context.Context, config RetryConfig, operation func() error,
) error {
	backoffStrategy := backoff.WithMaxRetries(
		backoff.NewExponentialBackOff(
			backoff.WithInitialInterval(config.InitialInterval),
			backoff.WithMaxInterval(config.MaxInterval),
			backoff.WithMultiplier(config.Multiplier),
		),
		config.MaxRetries,
	)

	return backoff.Retry(func() error {
		err := operation()
		if !isRetryableError(err) {
			return backoff.Permanent(err) // Don't retry non-retryable errors
		}
		return err
	}, backoff.WithContext(backoffStrategy, ctx))
}
