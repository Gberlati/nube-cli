package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/secrets"
)

func TestOrderList_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":              101,
				"number":          "1001",
				"status":          "open",
				"payment_status":  "paid",
				"shipping_status": "shipped",
				"total":           "150.00",
				"created_at":      "2025-01-01T00:00:00Z",
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"order", "list", "--json", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if len(got) != 1 {
		t.Errorf("got %d orders, want 1", len(got))
	}
}

func TestOrderGet_JSON(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "orders/101") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              101,
			"number":          "1001",
			"status":          "open",
			"payment_status":  "paid",
			"shipping_status": "shipped",
			"total":           "150.00",
			"currency":        "ARS",
			"created_at":      "2025-01-01T00:00:00Z",
			"updated_at":      "2025-01-02T00:00:00Z",
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"order", "get", "101", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jsonStr(got, "id") != "101" {
		t.Errorf("id = %v", got["id"])
	}
}

func TestOrderList_Table(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{Email: "u@test.com", AccessToken: "tok"})

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":              101,
				"number":          "1001",
				"status":          "open",
				"payment_status":  "paid",
				"shipping_status": "shipped",
				"total":           "150.00",
				"created_at":      "2025-01-01T00:00:00Z",
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"order", "list", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1001") {
		t.Errorf("output = %q, want containing '1001'", output)
	}
}
