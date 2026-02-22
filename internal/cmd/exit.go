package cmd

import (
	"errors"

	"github.com/gberlati/nube-cli/internal/api"
)

// Stable exit codes — agents and scripts can rely on these values.
const (
	ExitOK               = 0
	ExitError            = 1
	ExitUsage            = 2
	ExitAuthRequired     = 3
	ExitNotFound         = 4
	ExitPermissionDenied = 5
	ExitRateLimited      = 6
	ExitRetryable        = 7
	ExitConfig           = 8
	ExitCancelled        = 9
	ExitPaymentRequired  = 10
	ExitValidation       = 11
)

// exitCodeMap documents the stable exit codes for agent tooling.
var exitCodeMap = []struct {
	Code int    `json:"code"`
	Name string `json:"name"`
	Desc string `json:"description"`
}{
	{ExitOK, "ok", "Success"},
	{ExitError, "error", "Generic error"},
	{ExitUsage, "usage", "Invalid usage / bad arguments"},
	{ExitAuthRequired, "auth_required", "Authentication required (HTTP 401)"},
	{ExitNotFound, "not_found", "Resource not found (HTTP 404)"},
	{ExitPermissionDenied, "permission_denied", "Permission denied (HTTP 403)"},
	{ExitRateLimited, "rate_limited", "Rate limited (HTTP 429)"},
	{ExitRetryable, "retryable", "Retryable server error (HTTP 5xx)"},
	{ExitConfig, "config", "Missing config or credentials"},
	{ExitCancelled, "cancelled", "User cancelled"},
	{ExitPaymentRequired, "payment_required", "Payment required (HTTP 402)"},
	{ExitValidation, "validation", "Validation error (HTTP 422)"},
}

type ExitErr struct {
	Code int
	Err  error
}

func (e *ExitErr) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

func (e *ExitErr) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var ee *ExitErr
	if errors.As(err, &ee) && ee != nil {
		if ee.Code < 0 {
			return 1
		}

		return ee.Code
	}

	return 1
}

// stableExitCode maps API errors to stable exit codes.
func stableExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	var ee *ExitErr
	if errors.As(err, &ee) {
		return ee.Code
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return ExitAuthRequired
	}

	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return ExitNotFound
	}

	var permErr *api.PermissionDeniedError
	if errors.As(err, &permErr) {
		return ExitPermissionDenied
	}

	var rlErr *api.RateLimitError
	if errors.As(err, &rlErr) {
		return ExitRateLimited
	}

	var cbErr *api.CircuitBreakerError
	if errors.As(err, &cbErr) {
		return ExitRetryable
	}

	var payErr *api.PaymentRequiredError
	if errors.As(err, &payErr) {
		return ExitPaymentRequired
	}

	var valErr *api.ValidationError
	if errors.As(err, &valErr) {
		return ExitValidation
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode >= 500 {
			return ExitRetryable
		}

		return ExitError
	}

	return ExitError
}
