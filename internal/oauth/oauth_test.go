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

	"github.com/gberlati/nube-cli/internal/credstore"
)

var testCreds = clientCredentials{
	clientID:     "test-client-id",
	clientSecret: "test-client-secret",
}

func mockReadOAuthClientOK(t *testing.T) {
	t.Helper()

	orig := readOAuthClient
	readOAuthClient = func(_ string) (clientCredentials, error) {
		return testCreds, nil
	}

	t.Cleanup(func() { readOAuthClient = orig })
}

func mockReadOAuthClientFail(t *testing.T) {
	t.Helper()

	orig := readOAuthClient
	readOAuthClient = func(_ string) (clientCredentials, error) {
		return clientCredentials{}, &credstore.OAuthClientMissingError{Name: "default"}
	}

	t.Cleanup(func() { readOAuthClient = orig })
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
	mockReadOAuthClientOK(t)

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

	tok, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 5 * time.Second}, testCreds, "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "server-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "server-tok")
	}
}

func TestAuthorizeServer_StateMismatch(t *testing.T) {
	mockReadOAuthClientOK(t)

	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=x&state=wrong-state", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	_, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 2 * time.Second}, testCreds, "")
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errStateMismatch) {
		t.Errorf("expected errStateMismatch, got %v", err)
	}
}

func TestAuthorizeServer_MissingCode(t *testing.T) {
	mockReadOAuthClientOK(t)

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

	_, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 2 * time.Second}, testCreds, "")
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, errMissingCode) {
		t.Errorf("expected errMissingCode, got %v", err)
	}
}

func TestAuthorizeServer_Timeout(t *testing.T) {
	mockReadOAuthClientOK(t)

	mockBrowser(t, func(_ string) error {
		// Don't simulate any callback — let it timeout
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := authorizeServer(ctx, AuthorizeOptions{}, testCreds, "")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestAuthorizeServer_BrokerFlow(t *testing.T) {
	// Cannot run in parallel — uses fixed port 8910
	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=broker-tok&user_id=77", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	tok, err := authorizeServer(context.Background(), AuthorizeOptions{Timeout: 5 * time.Second}, clientCredentials{}, "http://broker.example.com")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "broker-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "broker-tok")
	}

	if tok.UserID.String() != "77" {
		t.Errorf("UserID = %q, want %q", tok.UserID.String(), "77")
	}
}

func TestAuthorize_BrokerSkipsCredentials(t *testing.T) {
	// Make readOAuthClient fail — broker should not need them.
	mockReadOAuthClientFail(t)

	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=broker-tok&user_id=88", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	tok, err := Authorize(context.Background(), AuthorizeOptions{
		Timeout:   5 * time.Second,
		BrokerURL: "http://broker.example.com",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "broker-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "broker-tok")
	}
}

func TestAuthorize_FallsBackToNativeWithCredentials(t *testing.T) {
	mockReadOAuthClientOK(t)

	var capturedURL string

	mockBrowser(t, func(u string) error {
		capturedURL = u

		go func() {
			time.Sleep(50 * time.Millisecond)
			parsed, _ := url.Parse(capturedURL)
			state := parsed.Query().Get("state")

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=native-code&state=%s", CallbackPort, state)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	mockTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "native-tok",
			UserID:      "55",
		})
	})

	// No broker URL — should use native flow with credentials.
	tok, err := Authorize(context.Background(), AuthorizeOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "native-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "native-tok")
	}
}

func TestAuthorize_CredentialsError_FallsBackToBroker(t *testing.T) {
	// No broker URL + credentials fail → falls back to default broker.
	mockReadOAuthClientFail(t)

	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=fallback-tok&user_id=99", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	tok, err := Authorize(context.Background(), AuthorizeOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "fallback-tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "fallback-tok")
	}
}

func TestAuthorize_DefaultTimeout(t *testing.T) {
	// Verify Authorize sets timeout if not provided.
	// We use broker flow with a quick callback to test without a real server.
	mockBrowser(t, func(_ string) error {
		go func() {
			time.Sleep(50 * time.Millisecond)

			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=tok&user_id=1", CallbackPort)
			doCallbackRequest(t, callbackURL)
		}()

		return nil
	})

	tok, err := Authorize(context.Background(), AuthorizeOptions{
		BrokerURL: "http://broker.example.com",
		// Timeout: 0 → should default to 2min
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if tok.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "tok")
	}
}
