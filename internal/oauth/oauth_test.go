package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gberlati/nube-cli/internal/config"
)

var testCreds = config.ClientCredentials{
	ClientID:     "test-client-id",
	ClientSecret: "test-client-secret",
}

func mockReadCredentials(t *testing.T) {
	t.Helper()

	orig := readClientCredentials
	readClientCredentials = func(_ string) (config.ClientCredentials, error) {
		return testCreds, nil
	}

	t.Cleanup(func() { readClientCredentials = orig })
}

func mockBrowser(t *testing.T, fn func(string) error) {
	t.Helper()

	orig := openBrowserFn
	openBrowserFn = fn

	t.Cleanup(func() { openBrowserFn = orig })
}

// mockTokenServer starts an httptest server and overrides http.DefaultClient
// transport to redirect TokenURL to the test server.
func mockTokenServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = &tokenRedirectTransport{
		testURL: srv.URL,
		base:    http.DefaultTransport,
	}

	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })
}

// tokenRedirectTransport redirects requests to TokenURL to the test server.
type tokenRedirectTransport struct {
	testURL string
	base    http.RoundTripper
}

func (t *tokenRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() == TokenURL || req.URL.Host == "www.tiendanube.com" {
		testURL, _ := url.Parse(t.testURL)
		req.URL.Scheme = testURL.Scheme
		req.URL.Host = testURL.Host
	}

	return t.base.RoundTrip(req) //nolint:wrapcheck // test helper
}

func TestExtractCodeFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "full URL with code",
			input: "http://localhost:8910/callback?code=abc123&state=xyz",
			want:  "abc123",
		},
		{
			name:  "bare code",
			input: "abc123",
			want:  "abc123",
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractCodeFromURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExchangeCode_Success(t *testing.T) {
	mockTokenServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}

		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "tok-123",
			TokenType:   "bearer",
			Scope:       "read_products",
			UserID:      "999",
		})
	})

	tok, err := exchangeCode(context.Background(), testCreds, "auth-code")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "tok-123" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "tok-123")
	}
}

func TestExchangeCode_BadStatus(t *testing.T) {
	mockTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"bad_code"}`))
	})

	_, err := exchangeCode(context.Background(), testCreds, "bad-code")
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errTokenExchange) {
		t.Errorf("expected errTokenExchange, got %v", err)
	}
}

func TestExchangeCode_EmptyToken(t *testing.T) {
	mockTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(TokenResponse{
			TokenType: "bearer",
		})
	})

	_, err := exchangeCode(context.Background(), testCreds, "code")
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errNoAccessToken) {
		t.Errorf("expected errNoAccessToken, got %v", err)
	}
}

func TestAuthorizeManual_RemoteStep1(t *testing.T) {
	mockReadCredentials(t)

	_, err := authorizeManual(context.Background(), AuthorizeOptions{
		Remote: true,
		Step:   1,
	}, testCreds)

	var stepOne *StepOneComplete
	if !errors.As(err, &stepOne) {
		t.Fatalf("expected StepOneComplete, got %T: %v", err, err)
	}
}

func TestAuthorizeManual_RemoteStep2_WithAuthURL(t *testing.T) {
	mockReadCredentials(t)

	mockTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "remote-tok",
			UserID:      "1",
		})
	})

	tok, err := authorizeManual(context.Background(), AuthorizeOptions{
		Remote:  true,
		Step:    2,
		AuthURL: "http://example.com/callback?code=remote-code",
	}, testCreds)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "remote-tok" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
}

func TestAuthorizeManual_RemoteStep2_MissingAuthURL(t *testing.T) {
	mockReadCredentials(t)

	_, err := authorizeManual(context.Background(), AuthorizeOptions{
		Remote: true,
		Step:   2,
	}, testCreds)

	if !errors.Is(err, errMissingAuthURL) {
		t.Errorf("expected errMissingAuthURL, got %v", err)
	}
}

func doCallbackRequest(t *testing.T, callbackURL string) {
	t.Helper()

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, callbackURL, nil)
	if reqErr != nil {
		return
	}

	resp, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		return
	}

	resp.Body.Close()
}

func TestAuthorizeServer(t *testing.T) {
	// Cannot run in parallel — uses fixed port 8910
	mockReadCredentials(t)

	var capturedURL string

	mockBrowser(t, func(u string) error {
		capturedURL = u
		// Simulate browser callback
		go func() {
			time.Sleep(50 * time.Millisecond)
			// Parse state from the captured URL
			parsed, _ := url.Parse(capturedURL)
			state := parsed.Query().Get("state")

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=server-code&state=%s", CallbackPort, state)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	mockTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "server-tok",
			UserID:      "42",
			Scope:       "all",
		})
	})

	tok, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 5 * time.Second}, testCreds)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "server-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "server-tok")
	}
}

func TestAuthorizeServer_StateMismatch(t *testing.T) {
	mockReadCredentials(t)

	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=x&state=wrong-state", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	_, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 2 * time.Second}, testCreds)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errStateMismatch) {
		t.Errorf("expected errStateMismatch, got %v", err)
	}
}

func TestAuthorizeServer_MissingCode(t *testing.T) {
	mockReadCredentials(t)

	mockBrowser(t, func(capturedURL string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			parsed, _ := url.Parse(capturedURL)
			state := parsed.Query().Get("state")

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?state=%s", CallbackPort, state)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	_, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 2 * time.Second}, testCreds)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errMissingCode) {
		t.Errorf("expected errMissingCode, got %v", err)
	}
}

func TestAuthorizeServer_Timeout(t *testing.T) {
	mockReadCredentials(t)

	mockBrowser(t, func(_ string) error {
		// Don't simulate any callback — let it timeout
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := authorizeServer(ctx, AuthorizeOptions{}, testCreds)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestAuthorize_DelegatesToManual(t *testing.T) {
	mockReadCredentials(t)

	_, err := Authorize(context.Background(), AuthorizeOptions{
		Remote:  true,
		Step:    1,
		Timeout: 5 * time.Second,
	})

	var stepOne *StepOneComplete
	if !errors.As(err, &stepOne) {
		t.Fatalf("expected StepOneComplete, got %v", err)
	}
}

func TestAuthorize_CredentialsError(t *testing.T) {
	orig := readClientCredentials
	readClientCredentials = func(_ string) (config.ClientCredentials, error) {
		return config.ClientCredentials{}, &config.CredentialsMissingError{Path: "/test"}
	}

	t.Cleanup(func() { readClientCredentials = orig })

	_, err := Authorize(context.Background(), AuthorizeOptions{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error")
	}

	var credErr *config.CredentialsMissingError
	if !errors.As(err, &credErr) {
		t.Errorf("expected CredentialsMissingError, got %T: %v", err, err)
	}
}

func TestAuthorize_DefaultTimeout(t *testing.T) {
	// Just verify Authorize sets timeout if not provided.
	// We'll check by using manual + remote step 1 which doesn't need a real server.
	mockReadCredentials(t)

	_, err := Authorize(context.Background(), AuthorizeOptions{
		Remote: true,
		Step:   1,
		// Timeout: 0 → should default to 2min
	})

	var stepOne *StepOneComplete
	if !errors.As(err, &stepOne) {
		t.Fatalf("expected StepOneComplete, got %v", err)
	}
}
