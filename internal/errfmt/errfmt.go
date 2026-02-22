package errfmt

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/99designs/keyring"
	"github.com/alecthomas/kong"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/config"
)

func Format(err error) string {
	if err == nil {
		return ""
	}

	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return formatParseError(parseErr)
	}

	var credErr *config.CredentialsMissingError
	if errors.As(err, &credErr) {
		return fmt.Sprintf(
			"OAuth client credentials missing.\nCreate an app at https://partners.tiendanube.com and save credentials.\nThen run: nube auth credentials <credentials.json> (expected at %s)",
			credErr.Path,
		)
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("API error (HTTP %d): %s", apiErr.StatusCode, apiErr.Message)
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return "Authentication failed. Check your access token or run: nube auth add <email>"
	}

	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return fmt.Sprintf("Rate limit exceeded after %d retries. Try again in a few seconds.", rateLimitErr.Retries)
	}

	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return notFoundErr.Error()
	}

	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "Token not found in keyring. Run: nube auth add <email>"
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
