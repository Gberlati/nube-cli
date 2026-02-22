package api_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gberlati/nube-cli/internal/api"
)

var errSentinel = errors.New("other")

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.APIError
		want string
	}{
		{
			name: "with message",
			err:  &api.APIError{StatusCode: 400, Message: "bad request"},
			want: "API error 400: bad request",
		},
		{
			name: "with code only",
			err:  &api.APIError{StatusCode: 422, Code: "unprocessable"},
			want: "API error 422: unprocessable",
		},
		{
			name: "message takes precedence over code",
			err:  &api.APIError{StatusCode: 400, Code: "bad_request", Message: "invalid input"},
			want: "API error 400: invalid input",
		},
		{
			name: "status code only",
			err:  &api.APIError{StatusCode: 500},
			want: "API error 500",
		},
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

func TestRateLimitError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.RateLimitError
		want string
	}{
		{
			name: "with reset duration",
			err:  &api.RateLimitError{Reset: 5 * time.Second, Retries: 3},
			want: "rate limit exceeded, retry after 5s (attempted 3 retries)",
		},
		{
			name: "without reset duration",
			err:  &api.RateLimitError{Retries: 5},
			want: "rate limit exceeded after 5 retries",
		},
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

func TestNotFoundError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.NotFoundError
		want string
	}{
		{
			name: "with ID",
			err:  &api.NotFoundError{Resource: "product", ID: "123"},
			want: "product not found: 123",
		},
		{
			name: "without ID",
			err:  &api.NotFoundError{Resource: "product"},
			want: "product not found",
		},
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

func TestAuthError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.AuthError
		want string
	}{
		{
			name: "with message",
			err:  &api.AuthError{Message: "token expired"},
			want: "authentication failed: token expired",
		},
		{
			name: "without message",
			err:  &api.AuthError{},
			want: "authentication failed",
		},
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

func TestIsAPIError(t *testing.T) {
	t.Parallel()

	apiErr := &api.APIError{StatusCode: 400}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", apiErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", apiErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsAPIError(tt.err); got != tt.want {
				t.Errorf("IsAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	t.Parallel()

	rlErr := &api.RateLimitError{Retries: 3}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", rlErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", rlErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsRateLimitError(tt.err); got != tt.want {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	nfErr := &api.NotFoundError{Resource: "product"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", nfErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", nfErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	t.Parallel()

	authErr := &api.AuthError{Message: "expired"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", authErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", authErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsAuthError(tt.err); got != tt.want {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.ValidationError
		want string
	}{
		{
			name: "single field",
			err:  &api.ValidationError{StatusCode: 422, Fields: map[string][]string{"name": {"is too long"}}},
			want: "validation error: name: is too long",
		},
		{
			name: "no fields",
			err:  &api.ValidationError{StatusCode: 422},
			want: "validation error 422",
		},
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

func TestIsValidationError(t *testing.T) {
	t.Parallel()

	vErr := &api.ValidationError{StatusCode: 422, Fields: map[string][]string{"name": {"required"}}}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", vErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", vErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsValidationError(tt.err); got != tt.want {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaymentRequiredError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.PaymentRequiredError
		want string
	}{
		{
			name: "with message",
			err:  &api.PaymentRequiredError{Message: "subscription expired"},
			want: "payment required: subscription expired",
		},
		{
			name: "without message",
			err:  &api.PaymentRequiredError{},
			want: "payment required",
		},
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

func TestIsPaymentRequiredError(t *testing.T) {
	t.Parallel()

	prErr := &api.PaymentRequiredError{Message: "suspended"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", prErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", prErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsPaymentRequiredError(tt.err); got != tt.want {
				t.Errorf("IsPaymentRequiredError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionDeniedError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.PermissionDeniedError
		want string
	}{
		{
			name: "with message",
			err:  &api.PermissionDeniedError{Message: "insufficient scope"},
			want: "permission denied: insufficient scope",
		},
		{
			name: "without message",
			err:  &api.PermissionDeniedError{},
			want: "permission denied",
		},
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

func TestIsPermissionDeniedError(t *testing.T) {
	t.Parallel()

	pdErr := &api.PermissionDeniedError{Message: "denied"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", pdErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", pdErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsPermissionDeniedError(tt.err); got != tt.want {
				t.Errorf("IsPermissionDeniedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCircuitBreakerError_Error(t *testing.T) {
	t.Parallel()

	err := &api.CircuitBreakerError{Failures: 5}
	want := "circuit breaker open after 5 consecutive failures"

	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestIsCircuitBreakerError(t *testing.T) {
	t.Parallel()

	cbErr := &api.CircuitBreakerError{Failures: 5}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct", cbErr, true},
		{"wrapped", fmt.Errorf("wrap: %w", cbErr), true},
		{"negative", errSentinel, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := api.IsCircuitBreakerError(tt.err); got != tt.want {
				t.Errorf("IsCircuitBreakerError() = %v, want %v", got, tt.want)
			}
		})
	}
}
