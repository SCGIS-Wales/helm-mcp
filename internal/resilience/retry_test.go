package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	cfg := DefaultRetryConfig()
	calls := 0

	result, err := Do(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDo_SuccessOnRetry(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}
	calls := 0

	result, err := Do(context.Background(), cfg, func(_ context.Context) (int, error) {
		calls++
		if calls < 3 {
			return 0, errors.New("transient error")
		}
		return 42, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_AllAttemptsFail(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}
	calls := 0
	expectedErr := errors.New("persistent error")

	_, err := Do(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "", expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected persistent error, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := Do(ctx, cfg, func(_ context.Context) (string, error) {
		calls++
		return "", errors.New("error")
	})

	if err == nil {
		t.Error("expected an error")
	}
	if calls >= 5 {
		t.Errorf("expected fewer than 5 calls due to cancellation, got %d", calls)
	}
}

func TestDo_ShouldRetryFilter(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		ShouldRetry: func(err error) bool {
			return errors.Is(err, retryableErr)
		},
	}

	// Non-retryable error should not be retried.
	calls := 0
	_, err := Do(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "", nonRetryableErr
	})

	if !errors.Is(err, nonRetryableErr) {
		t.Errorf("expected non-retryable error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call for non-retryable error, got %d", calls)
	}
}

func TestDo_ZeroMaxAttempts(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 0}
	calls := 0

	_, err := Do(context.Background(), cfg, func(_ context.Context) (string, error) {
		calls++
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call with zero MaxAttempts (defaults to 1), got %d", calls)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 500*time.Millisecond {
		t.Errorf("expected BaseDelay=500ms, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("expected MaxDelay=10s, got %v", cfg.MaxDelay)
	}
}

func TestBackoffDelay(t *testing.T) {
	base := 100 * time.Millisecond
	max := 5 * time.Second

	// Attempt 0: should be around base + jitter.
	d0 := backoffDelay(0, base, max)
	if d0 < base || d0 > base*2 {
		t.Errorf("attempt 0 delay %v outside expected range [%v, %v]", d0, base, base*2)
	}

	// Attempt 3: should be capped at max.
	d3 := backoffDelay(10, base, max)
	if d3 > max {
		t.Errorf("attempt 10 delay %v exceeds max %v", d3, max)
	}
}
