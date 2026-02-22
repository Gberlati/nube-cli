package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/secrets"
)

func TestCustomerList_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         200,
				"name":       "Juan Perez",
				"email":      "juan@example.com",
				"phone":      "+5491155551234",
				"created_at": "2025-03-01T00:00:00Z",
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"customer", "list", "--json", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if len(got) != 1 {
		t.Errorf("got %d customers, want 1", len(got))
	}
}

func TestCustomerGet_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "customers/200") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         200,
			"name":       "Juan Perez",
			"email":      "juan@example.com",
			"phone":      "+5491155551234",
			"created_at": "2025-03-01T00:00:00Z",
			"updated_at": "2025-03-02T00:00:00Z",
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"customer", "get", "200", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jsonStr(got, "id") != "200" {
		t.Errorf("id = %v", got["id"])
	}
}

func TestCustomerList_Table(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         200,
				"name":       "Juan Perez",
				"email":      "juan@example.com",
				"phone":      "+5491155551234",
				"created_at": "2025-03-01T00:00:00Z",
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"customer", "list", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Juan Perez") {
		t.Errorf("output = %q, want containing 'Juan Perez'", output)
	}
}
