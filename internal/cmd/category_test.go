package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/credstore"
)

func TestCategoryList_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":            10,
				"name":          map[string]any{"es": "Ropa"},
				"handle":        map[string]any{"es": "ropa"},
				"parent":        nil,
				"subcategories": []any{map[string]any{"id": 11}},
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"category", "list", "--json", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if len(got) != 1 {
		t.Errorf("got %d categories, want 1", len(got))
	}
}

func TestCategoryGet_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "categories/10") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            10,
			"name":          map[string]any{"es": "Ropa"},
			"handle":        map[string]any{"es": "ropa"},
			"parent":        nil,
			"subcategories": []any{},
			"created_at":    "2025-01-01T00:00:00Z",
			"updated_at":    "2025-01-02T00:00:00Z",
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"category", "get", "10", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jsonStr(got, "id") != "10" {
		t.Errorf("id = %v", got["id"])
	}
}

func TestCategoryList_Table(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":            10,
				"name":          map[string]any{"es": "Ropa"},
				"handle":        map[string]any{"es": "ropa"},
				"parent":        nil,
				"subcategories": []any{},
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"category", "list", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Ropa") {
		t.Errorf("output = %q, want containing 'Ropa'", output)
	}
}
