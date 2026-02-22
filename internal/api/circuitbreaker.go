package api

import (
	"log/slog"
	"sync"
	"time"
)

const (
	// CircuitBreakerThreshold is the number of consecutive failures to open the circuit.
	CircuitBreakerThreshold = 5
	// CircuitBreakerResetTime is how long to wait before attempting to close the circuit.
	CircuitBreakerResetTime = 30 * time.Second
)

// CircuitBreaker prevents cascading failures by tracking consecutive errors
// and short-circuiting requests when the failure threshold is exceeded.
type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	open        bool
}

// NewCircuitBreaker creates a new CircuitBreaker in the closed state.
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{}
}

// RecordSuccess resets the failure count and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	wasOpen := cb.open
	cb.failures = 0
	cb.open = false

	if wasOpen {
		slog.Info("circuit breaker reset")
	}
}

// RecordFailure increments the failure count and opens the circuit if the threshold is reached.
// Returns true if the circuit just opened.
func (cb *CircuitBreaker) RecordFailure() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= CircuitBreakerThreshold {
		cb.open = true
		slog.Warn("circuit breaker opened", "failures", cb.failures)

		return true
	}

	return false
}

// IsOpen returns true if the circuit is open (requests should be rejected).
// Automatically resets after CircuitBreakerResetTime has elapsed.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}

	if time.Since(cb.lastFailure) > CircuitBreakerResetTime {
		cb.open = false
		cb.failures = 0

		slog.Info("circuit breaker attempting reset after timeout")

		return false
	}

	return true
}
