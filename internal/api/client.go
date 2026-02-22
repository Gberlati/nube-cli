package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// DefaultBaseURL is the Tienda Nube API base URL.
	DefaultBaseURL = "https://api.tiendanube.com/v1"
	// DefaultUserAgent is required by the Tienda Nube API (returns 400 if missing).
	DefaultUserAgent = "nube-cli (https://github.com/gberlati/nube-cli)"
)

// Client is the main HTTP client for the Tienda Nube API.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	storeID     string
	accessToken string
	userAgent   string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithHTTPClient overrides the underlying http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new API client for the given store.
// The storeID is the Tienda Nube user_id (store ID).
func New(storeID, accessToken string, opts ...Option) *Client {
	c := &Client{
		baseURL:     DefaultBaseURL,
		storeID:     storeID,
		accessToken: accessToken,
		userAgent:   DefaultUserAgent,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: NewRetryTransport(http.DefaultTransport),
		}
	}

	return c
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.baseURL, "/"), c.storeID, strings.TrimLeft(path, "/"))
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Tienda Nube uses "Authentication" header (not "Authorization").
	req.Header.Set("Authentication", "bearer "+c.accessToken)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req) //nolint:gosec // URL is constructed from configured base URL
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	return nil, parseErrorResponse(resp)
}

// Get performs a GET request to the given path.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}

	return c.do(req)
}

// Post performs a POST request with JSON body.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

// Put performs a PUT request with JSON body.
func (c *Client) Put(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

// DecodeResponse reads and decodes a JSON response body into the given type.
func DecodeResponse[T any](resp *http.Response) (T, error) {
	var result T

	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

func parseErrorResponse(resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}

	// Try to parse structured error from response.
	var parsed struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		Description string `json:"description"`
	}

	if json.Unmarshal(body, &parsed) == nil {
		apiErr.Code = parsed.Code

		if parsed.Message != "" {
			apiErr.Message = parsed.Message
		} else if parsed.Description != "" {
			apiErr.Message = parsed.Description
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &AuthError{Message: apiErr.Message}
	}

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Resource: "resource"}
	}

	return apiErr
}
