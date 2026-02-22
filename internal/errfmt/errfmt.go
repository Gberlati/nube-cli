package errfmt

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/credstore"
)

func Format(err error) string {
	if err == nil {
		return ""
	}

	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return formatParseError(parseErr)
	}

	var credErr *credstore.OAuthClientMissingError
	if errors.As(err, &credErr) {
		return "OAuth client credentials missing.\nCreate an app at https://partners.tiendanube.com and save credentials.\nThen run: nube auth credentials <credentials.json>"
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("API error (HTTP %d): %s", apiErr.StatusCode, apiErr.Message)
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return "Authentication failed. Check your access token or run: nube login"
	}

	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return fmt.Sprintf("Rate limit exceeded after %d retries. Try again in a few seconds.", rateLimitErr.Retries)
	}

	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return notFoundErr.Error()
	}

	var validationErr *api.ValidationError
	if errors.As(err, &validationErr) {
		return formatValidationError(validationErr)
	}

	var paymentErr *api.PaymentRequiredError
	if errors.As(err, &paymentErr) {
		return "Store access suspended (payment required). Check your Tienda Nube subscription."
	}

	var permDeniedErr *api.PermissionDeniedError
	if errors.As(err, &permDeniedErr) {
		if permDeniedErr.Message != "" {
			return fmt.Sprintf("Permission denied: %s", permDeniedErr.Message)
		}

		return "Permission denied"
	}

	var cbErr *api.CircuitBreakerError
	if errors.As(err, &cbErr) {
		return "API temporarily unavailable (circuit breaker open). Try again shortly."
	}

	if errors.Is(err, os.ErrNotExist) {
		return err.Error()
	}

	var userErr *UserFacingError
	if errors.As(err, &userErr) {
		return userErr.Message
	}

	return err.Error()
}

// UserFacingError forces a specific message, while preserving the underlying cause.
type UserFacingError struct {
	Message string
	Cause   error
}

func (e *UserFacingError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e *UserFacingError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func NewUserFacingError(message string, cause error) error {
	return &UserFacingError{Message: message, Cause: cause}
}

func formatValidationError(err *api.ValidationError) string {
	// Sort field names for deterministic output.
	fields := make([]string, 0, len(err.Fields))
	for f := range err.Fields {
		fields = append(fields, f)
	}

	sort.Strings(fields)

	parts := make([]string, 0, len(fields))

	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s: %s", f, strings.Join(err.Fields[f], ", ")))
	}

	return fmt.Sprintf("Validation error: %s", strings.Join(parts, "; "))
}

func formatParseError(err *kong.ParseError) string {
	msg := err.Error()

	if strings.Contains(msg, "did you mean") {
		return msg
	}

	if strings.HasPrefix(msg, "unknown flag") {
		return msg + "\nRun with --help to see available flags"
	}

	if strings.Contains(msg, "missing") || strings.Contains(msg, "required") {
		return msg + "\nRun with --help to see usage"
	}

	return msg
}
