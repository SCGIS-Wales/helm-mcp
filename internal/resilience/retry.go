package resilience

import (
	"context"
	"math/rand/v2"
	"time"
)

// RetryConfig holds configuration for the retry mechanism.
type RetryConfig struct {
	// MaxAttempts is the maximum number of total attempts (including the first).
	MaxAttempts int
	// BaseDelay is the initial delay before the first retry.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff delay.
	MaxDelay time.Duration
	// ShouldRetry determines if an error is retryable. If nil, all errors are retried.
	ShouldRetry func(error) bool
}

// DefaultRetryConfig returns sensible defaults for MCP tool calls.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}
}

// Do executes fn with retry logic according to the configuration.
// It respects context cancellation between retries.
func Do[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	var lastErr error
	var zero T

	for attempt := range cfg.MaxAttempts {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry if context is done.
		if ctx.Err() != nil {
			return zero, lastErr
		}

		// Check if the error is retryable.
		if cfg.ShouldRetry != nil && !cfg.ShouldRetry(err) {
			return zero, lastErr
		}

		// Don't sleep after the last attempt.
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)
		select {
		case <-ctx.Done():
			return zero, lastErr
		case <-time.After(delay):
		}
	}

	return zero, lastErr
}

// backoffDelay calculates exponential backoff with jitter.
// delay = min(maxDelay, baseDelay * 2^attempt + random_jitter)
func backoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	delay := baseDelay
	for range attempt {
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
			break
		}
	}

	// Add jitter: random value between 0 and delay/2.
	// Cryptographic randomness is not needed for retry jitter.
	jitter := time.Duration(rand.Int64N(int64(delay/2) + 1)) //nolint:gosec // jitter does not require crypto/rand
	delay += jitter

	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}
