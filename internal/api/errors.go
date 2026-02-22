package api

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// APIError wraps a non-2xx HTTP response from the Tienda Nube API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
	}

	if e.Code != "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Code)
	}

	return fmt.Sprintf("API error %d", e.StatusCode)
}

// RateLimitError indicates a 429 response after exhausting retries.
type RateLimitError struct {
	Limit     int
	Remaining int
	Reset     time.Duration
	Retries   int
}

func (e *RateLimitError) Error() string {
	if e.Reset > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %s (attempted %d retries)", e.Reset, e.Retries)
	}

	return fmt.Sprintf("rate limit exceeded after %d retries", e.Retries)
}

// NotFoundError indicates a 404 response for a specific resource.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}

	return fmt.Sprintf("%s not found", e.Resource)
}

// AuthError indicates an authentication failure (401).
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("authentication failed: %s", e.Message)
	}

	return "authentication failed"
}

// IsAPIError checks if the error is an API error.
func IsAPIError(err error) bool {
	var e *APIError
	return errors.As(err, &e)
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	var e *AuthError
	return errors.As(err, &e)
}

// ValidationError indicates a 422 field validation error.
type ValidationError struct {
	StatusCode int
	Fields     map[string][]string // field -> error messages
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))

	for field, msgs := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(msgs, ", ")))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("validation error %d", e.StatusCode)
	}

	return fmt.Sprintf("validation error: %s", strings.Join(parts, "; "))
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	var e *ValidationError
	return errors.As(err, &e)
}

// PaymentRequiredError indicates a 402 response (store subscription suspended).
type PaymentRequiredError struct {
	Message string
}

func (e *PaymentRequiredError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("payment required: %s", e.Message)
	}

	return "payment required"
}

// IsPaymentRequiredError checks if the error is a payment required error.
func IsPaymentRequiredError(err error) bool {
	var e *PaymentRequiredError
	return errors.As(err, &e)
}

// PermissionDeniedError indicates a 403 response.
type PermissionDeniedError struct {
	Message string
}

func (e *PermissionDeniedError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("permission denied: %s", e.Message)
	}

	return "permission denied"
}

// IsPermissionDeniedError checks if the error is a permission denied error.
func IsPermissionDeniedError(err error) bool {
	var e *PermissionDeniedError
	return errors.As(err, &e)
}

// CircuitBreakerError indicates the circuit breaker is open.
type CircuitBreakerError struct {
	Failures int
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker open after %d consecutive failures", e.Failures)
}

// IsCircuitBreakerError checks if the error is a circuit breaker error.
func IsCircuitBreakerError(err error) bool {
	var e *CircuitBreakerError
	return errors.As(err, &e)
}
