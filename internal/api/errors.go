package api

import (
	"errors"
	"fmt"
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
