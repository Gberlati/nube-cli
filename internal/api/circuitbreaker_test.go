package api

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StartsClosedState(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()
	if cb.IsOpen() {
		t.Fatal("expected circuit to start closed")
	}
}

func TestCircuitBreaker_OpensAtThreshold(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	for i := range CircuitBreakerThreshold - 1 {
		opened := cb.RecordFailure()
		if opened {
			t.Fatalf("circuit opened early at failure %d", i+1)
		}
	}

	opened := cb.RecordFailure()
	if !opened {
		t.Fatal("expected circuit to open at threshold")
	}

	if !cb.IsOpen() {
		t.Fatal("expected circuit to be open")
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	// Record some failures but not enough to open.
	for range CircuitBreakerThreshold - 1 {
		cb.RecordFailure()
	}

	cb.RecordSuccess()

	// Now recording threshold-1 more failures should not open the circuit.
	for range CircuitBreakerThreshold - 1 {
		cb.RecordFailure()
	}

	if cb.IsOpen() {
		t.Fatal("circuit should still be closed after success reset")
	}
}

func TestCircuitBreaker_SuccessClosesOpenCircuit(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	for range CircuitBreakerThreshold {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Fatal("expected circuit to be open")
	}

	cb.RecordSuccess()

	if cb.IsOpen() {
		t.Fatal("expected circuit to be closed after success")
	}
}

func TestCircuitBreaker_TimeoutReset(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	for range CircuitBreakerThreshold {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Fatal("expected circuit to be open")
	}

	// Simulate timeout by backdating lastFailure.
	cb.mu.Lock()
	cb.lastFailure = time.Now().Add(-(CircuitBreakerResetTime + time.Second))
	cb.mu.Unlock()

	if cb.IsOpen() {
		t.Fatal("expected circuit to be closed after timeout")
	}
}
