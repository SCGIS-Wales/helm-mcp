package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StartsInClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("expected Allow() = true in closed state")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 3, RecoveryTimeout: 30 * time.Second}
	cb := NewCircuitBreaker(cfg)

	// Record failures up to threshold.
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("expected circuit to remain closed before threshold")
	}

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected StateOpen after %d failures, got %s", cfg.FailureThreshold, cb.State())
	}
	if cb.Allow() {
		t.Error("expected Allow() = false in open state")
	}
}

func TestCircuitBreaker_SuccessResets(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 3, RecoveryTimeout: 30 * time.Second}
	cb := NewCircuitBreaker(cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	if cb.Failures() != 0 {
		t.Errorf("expected failures reset to 0, got %d", cb.Failures())
	}
	if cb.State() != StateClosed {
		t.Error("expected StateClosed after success")
	}
}

func TestCircuitBreaker_HalfOpenAfterRecovery(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, RecoveryTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	// Open the circuit.
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected open state")
	}

	// Advance time past recovery timeout.
	now = now.Add(200 * time.Millisecond)

	if !cb.Allow() {
		t.Error("expected Allow() = true after recovery timeout (half-open)")
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_SuccessCloses(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, RecoveryTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.RecordFailure()
	cb.RecordFailure()
	now = now.Add(200 * time.Millisecond)
	cb.Allow() // transitions to half-open

	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed after half-open success, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_FailureReopens(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, RecoveryTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.RecordFailure()
	cb.RecordFailure()
	now = now.Add(200 * time.Millisecond)
	cb.Allow() // transitions to half-open

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected StateOpen after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	if cfg.FailureThreshold != 5 {
		t.Errorf("expected default threshold 5, got %d", cfg.FailureThreshold)
	}
	if cfg.RecoveryTimeout != 30*time.Second {
		t.Errorf("expected 30s recovery timeout, got %v", cfg.RecoveryTimeout)
	}
}

func TestCircuitBreaker_InvalidConfig(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: -1, RecoveryTimeout: -1})
	if cb.failureThreshold != 5 {
		t.Error("expected invalid threshold to be corrected to default")
	}
	if cb.recoveryTimeout != 30*time.Second {
		t.Error("expected invalid recovery timeout to be corrected to default")
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestErrCircuitOpen(t *testing.T) {
	if ErrCircuitOpen == nil {
		t.Fatal("ErrCircuitOpen should not be nil")
	}
	if ErrCircuitOpen.Error() == "" {
		t.Error("ErrCircuitOpen should have a message")
	}
}
