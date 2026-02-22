package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gberlati/nube-cli/internal/api"
)

func TestExitErr_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *ExitErr
		want string
	}{
		{"with wrapped error", &ExitErr{Code: 1, Err: errors.New("fail")}, "fail"},
		{"nil Err field", &ExitErr{Code: 0, Err: nil}, ""},
		{"nil receiver", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitErr_Unwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner")
	ee := &ExitErr{Code: 2, Err: inner}

	if !errors.Is(ee, inner) {
		t.Error("errors.Is should find inner error")
	}

	wrapped := fmt.Errorf("wrap: %w", ee)
	var target *ExitErr
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should find ExitErr through wrapping")
	}

	if target.Code != 2 {
		t.Errorf("Code = %d, want 2", target.Code)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"exit error code 2", &ExitErr{Code: 2, Err: errors.New("e")}, 2},
		{"exit error code 0", &ExitErr{Code: 0, Err: errors.New("e")}, 0},
		{"wrapped exit error", fmt.Errorf("wrap: %w", &ExitErr{Code: 3, Err: errors.New("e")}), 3},
		{"bare error", errors.New("fail"), 1},
		{"negative code", &ExitErr{Code: -1, Err: errors.New("e")}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStableExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"auth error", &api.AuthError{Message: "bad token"}, ExitAuthRequired},
		{"not found", &api.NotFoundError{Resource: "product"}, ExitNotFound},
		{"permission denied", &api.PermissionDeniedError{}, ExitPermissionDenied},
		{"rate limited", &api.RateLimitError{Retries: 3}, ExitRateLimited},
		{"circuit breaker", &api.CircuitBreakerError{Failures: 5}, ExitRetryable},
		{"payment required", &api.PaymentRequiredError{}, ExitPaymentRequired},
		{"validation", &api.ValidationError{StatusCode: 422, Fields: map[string][]string{"name": {"required"}}}, ExitValidation},
		{"api 5xx", &api.APIError{StatusCode: 500, Message: "server"}, ExitRetryable},
		{"api other", &api.APIError{StatusCode: 400, Message: "bad"}, ExitError},
		{"generic", errors.New("boom"), ExitError},
		{"wrapped auth", fmt.Errorf("wrap: %w", &api.AuthError{}), ExitAuthRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stableExitCode(tt.err); got != tt.want {
				t.Errorf("stableExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
