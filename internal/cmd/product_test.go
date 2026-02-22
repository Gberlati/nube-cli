package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/credstore"
)

func TestProductList_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":        1,
				"name":      map[string]any{"es": "Product 1"},
				"handle":    map[string]any{"es": "product-1"},
				"published": true,
				"variants":  []any{map[string]any{"price": "10.00"}},
			},
			{
				"id":        2,
				"name":      map[string]any{"es": "Product 2"},
				"handle":    map[string]any{"es": "product-2"},
				"published": false,
				"variants":  []any{},
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"product", "list", "--json", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if len(got) != 2 {
		t.Errorf("got %d products, want 2", len(got))
	}
}

func TestProductList_Table(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":        1,
				"name":      map[string]any{"es": "Product A"},
				"handle":    map[string]any{"es": "product-a"},
				"published": true,
				"variants":  []any{map[string]any{"price": "25.00"}},
			},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"product", "list", "--page", "1"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Product A") {
		t.Errorf("output = %q, want containing 'Product A'", output)
	}
}

func TestProductGet_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "products/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        42,
			"name":      map[string]any{"es": "Zapato"},
			"handle":    map[string]any{"es": "zapato"},
			"published": true,
			"variants":  []any{map[string]any{"sku": "ZP-001", "price": "50.00"}},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"product", "get", "42", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jsonStr(got, "id") != "42" {
		t.Errorf("id = %v", got["id"])
	}
}

func TestProductGetBySku_JSON(t *testing.T) {
	stores := map[string]credstore.StoreProfile{
		"test": {StoreID: "123", AccessToken: "tok"},
	}
	setupCredStore(t, stores, "test")

	setupMockAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "products/sku/ABC-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     99,
			"name":   map[string]any{"es": "Remera"},
			"handle": map[string]any{"es": "remera"},
		})
	}))

	buf := captureStdout(t)
	err := Execute([]string{"product", "get-by-sku", "ABC-123", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jsonStr(got, "id") != "99" {
		t.Errorf("id = %v", got["id"])
	}
}
