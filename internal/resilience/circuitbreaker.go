package resilience

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests through (normal operation).
	StateClosed CircuitState = iota
	// StateOpen blocks requests (failures exceeded threshold).
	StateOpen
	// StateHalfOpen allows a single probe request to test recovery.
	StateHalfOpen
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open: backend unavailable, please retry later")

// CircuitBreaker implements the three-state circuit breaker pattern.
// It tracks consecutive failures and opens the circuit when a threshold is exceeded,
// preventing cascading failures to an unhealthy backend.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failures         int
	failureThreshold int
	recoveryTimeout  time.Duration
	lastFailure      time.Time
	nowFunc          func() time.Time // for testing
}

// CircuitBreakerConfig holds configuration for a CircuitBreaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	FailureThreshold int
	// RecoveryTimeout is how long to wait before transitioning from Open to HalfOpen.
	RecoveryTimeout time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults for MCP tool calls.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
	}
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		recoveryTimeout:  cfg.RecoveryTimeout,
		nowFunc:          time.Now,
	}
}

// Allow checks whether a request should be allowed through.
// Returns true if the request is allowed, false if the circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if cb.nowFunc().Sub(cb.lastFailure) >= cb.recoveryTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		// Only one probe at a time in half-open state.
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful request and resets the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure records a failed request and potentially opens the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = cb.nowFunc()

	if cb.state == StateHalfOpen || cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Failures returns the current consecutive failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// String returns the string representation of a CircuitState.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
