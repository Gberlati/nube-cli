package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/secrets"
)

func setupMockAPIClient(t *testing.T, handler http.Handler) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := newAPIClient
	newAPIClient = func(_ *RootFlags) (*api.Client, error) {
		return api.New("123", "test-token", api.WithBaseURL(srv.URL+"/v1"), api.WithHTTPClient(srv.Client())), nil
	}
	t.Cleanup(func() { newAPIClient = orig })
}

func TestStoreGet_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              123,
			"name":            map[string]any{"es": "Mi Tienda"},
			"email":           "store@example.com",
			"original_domain": "mitienda.mitiendanube.com",
			"plan_name":       "Enterprise",
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"store", "get", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["email"] != "store@example.com" {
		t.Errorf("email = %v", got["email"])
	}
}

func TestStoreGet_Human(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              123,
			"name":            map[string]any{"es": "Mi Tienda"},
			"email":           "store@example.com",
			"original_domain": "mitienda.example.com",
			"plan_name":       "Pro",
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"store", "get"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Mi Tienda") {
		t.Errorf("output = %q, want containing 'Mi Tienda'", output)
	}
}
