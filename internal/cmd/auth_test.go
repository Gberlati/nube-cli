package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/credstore"
	"github.com/gberlati/nube-cli/internal/oauth"
)

func mockAuthorizeOAuth(t *testing.T, tok oauth.TokenResponse, err error) {
	t.Helper()
	orig := authorizeOAuth
	authorizeOAuth = func(_ context.Context, _ oauth.AuthorizeOptions) (oauth.TokenResponse, error) {
		return tok, err
	}
	t.Cleanup(func() { authorizeOAuth = orig })
}

func TestLogin_Success(t *testing.T) {
	setupConfigDir(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{
		AccessToken: "tok-123",
		UserID:      "999",
		Scope:       "read_products write_products",
	}, nil)

	buf := captureStdout(t)
	err := Execute([]string{"login", "my-shop"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-shop") {
		t.Errorf("output = %q, want containing profile name", output)
	}

	// Verify stored.
	p, getErr := credstore.GetStore("my-shop")
	if getErr != nil {
		t.Fatalf("GetStore: %v", getErr)
	}

	if p.AccessToken != "tok-123" {
		t.Errorf("AccessToken = %q", p.AccessToken)
	}
}

func TestLogin_JSON(t *testing.T) {
	setupConfigDir(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{
		AccessToken: "tok",
		UserID:      "1",
		Scope:       "all",
	}, nil)

	buf := captureStdout(t)
	err := Execute([]string{"login", "shop", "--json"})
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

func TestLogin_AutoName(t *testing.T) {
	setupConfigDir(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{
		AccessToken: "tok",
		UserID:      "42",
	}, nil)

	buf := captureStdout(t)
	err := Execute([]string{"login"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "store-42") {
		t.Errorf("output = %q, want containing auto-generated name", output)
	}
}

func TestLogin_OAuthError(t *testing.T) {
	setupConfigDir(t)
	mockAuthorizeOAuth(t, oauth.TokenResponse{}, errors.New("oauth failed"))

	_ = captureStdout(t)
	err := Execute([]string{"login", "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogin_BrokerURL(t *testing.T) {
	setupConfigDir(t)

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
	err := Execute([]string{"login", "test", "--broker-url", "http://broker.test"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if capturedOpts.BrokerURL != "http://broker.test" {
		t.Errorf("BrokerURL = %q, want %q", capturedOpts.BrokerURL, "http://broker.test")
	}
}

func TestAuthList(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"my-shop": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "my-shop")

	buf := captureStdout(t)
	err := Execute([]string{"auth", "list"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(buf.String(), "my-shop") {
		t.Errorf("output = %q, want containing store name", buf.String())
	}
}

func TestLogout(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"my-shop": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "my-shop")

	buf := captureStdout(t)
	err := Execute([]string{"logout", "my-shop", "--force"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	_ = buf.String()

	// Verify deleted.
	_, getErr := credstore.GetStore("my-shop")
	if getErr == nil {
		t.Error("expected store to be deleted")
	}
}

func TestAuthStatus(t *testing.T) {
	setupConfigDir(t)

	buf := captureStdout(t)
	err := Execute([]string{"auth", "status"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(buf.String(), "credentials") {
		t.Errorf("output = %q, want containing 'credentials'", buf.String())
	}
}

func TestAuthToken_Plain(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"my-shop": {StoreID: "999", AccessToken: "secret-tok"},
	}
	setupCredStore(t, stores, "my-shop")

	buf := captureStdout(t)
	err := Execute([]string{"auth", "token", "my-shop"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "secret-tok" {
		t.Errorf("output = %q, want %q", got, "secret-tok")
	}
}

func TestAuthToken_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"my-shop": {StoreID: "999", AccessToken: "secret-tok"},
	}
	setupCredStore(t, stores, "my-shop")

	buf := captureStdout(t)
	err := Execute([]string{"auth", "token", "my-shop", "--json"})
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

	if got["store_id"] != "999" {
		t.Errorf("store_id = %v", got["store_id"])
	}
}

func TestAuthToken_DefaultStore(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"only": {StoreID: "111", AccessToken: "only-tok"},
	}
	setupCredStore(t, stores, "only")

	buf := captureStdout(t)
	// No name arg — should auto-resolve to the single stored profile.
	err := Execute([]string{"auth", "token"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "only-tok" {
		t.Errorf("output = %q, want %q", got, "only-tok")
	}
}

func TestAuthDefault(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"a": {StoreID: "1", AccessToken: "ta"},
		"b": {StoreID: "2", AccessToken: "tb"},
	}
	setupCredStore(t, stores, "a")

	buf := captureStdout(t)
	err := Execute([]string{"auth", "default", "b"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	_ = buf.String()

	f, _ := credstore.Read()
	if f.DefaultStore != "b" {
		t.Errorf("DefaultStore = %q, want %q", f.DefaultStore, "b")
	}
}
