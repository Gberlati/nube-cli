package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/oauth"
	"github.com/gberlati/nube-cli/internal/secrets"
)

func mockAuthorizeOAuth(t *testing.T, tok oauth.TokenResponse, err error) {
	t.Helper()
	orig := authorizeOAuth
	authorizeOAuth = func(_ context.Context, _ oauth.AuthorizeOptions) (oauth.TokenResponse, error) {
		return tok, err
	}
	t.Cleanup(func() { authorizeOAuth = orig })
}

func TestAuthAdd_Success(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{
		AccessToken: "tok-123",
		UserID:      "999",
		Scope:       "read_products write_products",
	}, nil)

	buf := captureStdout(t)
	err := Execute([]string{"auth", "add", "user@example.com"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "user@example.com") {
		t.Errorf("output = %q, want containing email", output)
	}
}

func TestAuthAdd_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{
		AccessToken: "tok",
		UserID:      "1",
		Scope:       "all",
	}, nil)

	buf := captureStdout(t)
	err := Execute([]string{"auth", "add", "user@example.com", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["stored"] != true {
		t.Errorf("stored = %v", got["stored"])
	}
}

func TestAuthAdd_EmptyEmail(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"auth", "add", ""})
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestAuthAdd_OAuthError(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{}, errors.New("oauth failed"))

	_ = captureStdout(t)
	err := Execute([]string{"auth", "add", "user@example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthAdd_BrokerURL(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)

	var capturedOpts oauth.AuthorizeOptions

	orig := authorizeOAuth
	authorizeOAuth = func(_ context.Context, opts oauth.AuthorizeOptions) (oauth.TokenResponse, error) {
		capturedOpts = opts
		return oauth.TokenResponse{
			AccessToken: "tok",
			UserID:      "1",
		}, nil
	}
	t.Cleanup(func() { authorizeOAuth = orig })

	_ = captureStdout(t)
	err := Execute([]string{"auth", "add", "user@example.com", "--broker-url", "http://broker.test"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if capturedOpts.BrokerURL != "http://broker.test" {
		t.Errorf("BrokerURL = %q, want %q", capturedOpts.BrokerURL, "http://broker.test")
	}
}

func TestAuthList(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		AccessToken: "tok",
	})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "list"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(buf.String(), "user@example.com") {
		t.Errorf("output = %q, want containing email", buf.String())
	}
}

func TestAuthRemove(t *testing.T) {
	setupConfigDir(t)
	store := setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		AccessToken: "tok",
	})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "remove", "user@example.com", "--force"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	_ = buf.String()

	// Verify token was deleted
	_, getErr := store.GetToken("default", "user@example.com")
	if getErr == nil {
		t.Error("expected token to be deleted")
	}
}

func TestAuthTokensList(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		AccessToken: "tok",
	})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "tokens", "list"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(buf.String(), "user@example.com") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestAuthStatus(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)

	buf := captureStdout(t)
	err := Execute([]string{"auth", "status"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(buf.String(), "config") {
		t.Errorf("output = %q, want containing 'config'", buf.String())
	}
}

func TestAuthToken_Plain(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		UserID:      "999",
		AccessToken: "secret-tok",
	})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "token", "user@example.com"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "secret-tok" {
		t.Errorf("output = %q, want %q", got, "secret-tok")
	}
}

func TestAuthToken_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		UserID:      "999",
		AccessToken: "secret-tok",
	})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "token", "user@example.com", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["access_token"] != "secret-tok" {
		t.Errorf("access_token = %v", got["access_token"])
	}

	if got["user_id"] != "999" {
		t.Errorf("user_id = %v", got["user_id"])
	}

	if got["email"] != "user@example.com" {
		t.Errorf("email = %v", got["email"])
	}
}

func TestAuthToken_DefaultAccount(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "only@example.com",
		UserID:      "111",
		AccessToken: "only-tok",
	})

	buf := captureStdout(t)
	// No email arg — should auto-resolve to the single stored account.
	err := Execute([]string{"auth", "token"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "only-tok" {
		t.Errorf("output = %q, want %q", got, "only-tok")
	}
}

func TestAuthToken_Export(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		UserID:      "999",
		AccessToken: "secret-tok",
	})

	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	_ = captureStdout(t)
	err := Execute([]string{"auth", "token", "user@example.com", "--export", envFile})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	content, readErr := os.ReadFile(envFile)
	if readErr != nil {
		t.Fatalf("read env file: %v", readErr)
	}

	got := string(content)
	if !strings.Contains(got, "NUBE_ACCESS_TOKEN=secret-tok") {
		t.Errorf("env file missing NUBE_ACCESS_TOKEN, got: %q", got)
	}

	if !strings.Contains(got, "NUBE_USER_ID=999") {
		t.Errorf("env file missing NUBE_USER_ID, got: %q", got)
	}

	// Check file permissions.
	info, statErr := os.Stat(envFile)
	if statErr != nil {
		t.Fatalf("stat: %v", statErr)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestAuthToken_ExportJSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "user@example.com",
		UserID:      "999",
		AccessToken: "secret-tok",
	})

	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	buf := captureStdout(t)
	err := Execute([]string{"auth", "token", "user@example.com", "--export", envFile, "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["exported"] != true {
		t.Errorf("exported = %v", got["exported"])
	}

	if got["path"] != envFile {
		t.Errorf("path = %v", got["path"])
	}
}
