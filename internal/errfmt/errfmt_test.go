package errfmt_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/config"
	"github.com/gberlati/nube-cli/internal/errfmt"
)

var (
	errTestCause   = errors.New("root cause")
	errTestGeneric = errors.New("something went wrong")
	errTestContext = errors.New("context")
)

func TestFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "nil",
			err:      nil,
			contains: "",
		},
		{
			name:     "api error",
			err:      &api.APIError{StatusCode: 500, Message: "internal"},
			contains: "API error (HTTP 500): internal",
		},
		{
			name:     "auth error",
			err:      &api.AuthError{Message: "expired"},
			contains: "Authentication failed",
		},
		{
			name:     "rate limit error",
			err:      &api.RateLimitError{Retries: 3},
			contains: "Rate limit exceeded after 3 retries",
		},
		{
			name:     "not found error",
			err:      &api.NotFoundError{Resource: "product", ID: "42"},
			contains: "product not found: 42",
		},
		{
			name:     "credentials missing",
			err:      &config.CredentialsMissingError{Path: "/home/test/.config/nube-cli/credentials.json"},
			contains: "OAuth client credentials missing",
		},
		{
			name:     "credentials missing contains path",
			err:      &config.CredentialsMissingError{Path: "/expected/path"},
			contains: "/expected/path",
		},
		{
			name:     "validation error",
			err:      &api.ValidationError{StatusCode: 422, Fields: map[string][]string{"name": {"is too long"}}},
			contains: "Validation error: name: is too long",
		},
		{
			name:     "payment required error",
			err:      &api.PaymentRequiredError{Message: "suspended"},
			contains: "Store access suspended (payment required)",
		},
		{
			name:     "permission denied with message",
			err:      &api.PermissionDeniedError{Message: "insufficient scope"},
			contains: "Permission denied: insufficient scope",
		},
		{
			name:     "permission denied without message",
			err:      &api.PermissionDeniedError{},
			contains: "Permission denied",
		},
		{
			name:     "circuit breaker error",
			err:      &api.CircuitBreakerError{Failures: 5},
			contains: "API temporarily unavailable (circuit breaker open)",
		},
		{
			name:     "user facing error",
			err:      errfmt.NewUserFacingError("custom message", errTestCause),
			contains: "custom message",
		},
		{
			name:     "wrapped api error",
			err:      errors.Join(errTestContext, &api.APIError{StatusCode: 403, Message: "forbidden"}),
			contains: "API error (HTTP 403): forbidden",
		},
		{
			name:     "generic error",
			err:      errTestGeneric,
			contains: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errfmt.Format(tt.err)
			if tt.err == nil {
				if got != "" {
					t.Errorf("Format(nil) = %q, want empty", got)
				}

				return
			}

			if !strings.Contains(got, tt.contains) {
				t.Errorf("Format() = %q, want containing %q", got, tt.contains)
			}
		})
	}
}

func TestUserFacingError(t *testing.T) {
	t.Parallel()

	err := errfmt.NewUserFacingError("friendly message", errTestCause)

	if err.Error() != "friendly message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "friendly message")
	}

	if !errors.Is(err, errTestCause) {
		t.Error("Unwrap should expose the cause")
	}
}
