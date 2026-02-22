package cmd

import (
	"net/url"
	"testing"

	"github.com/gberlati/nube-cli/internal/secrets"
)

func TestPaginationFlags_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		flags   PaginationFlags
		wantP   string
		wantPP  string
		allPage bool
	}{
		{"defaults no page", PaginationFlags{Page: 0, PerPage: 30}, "", "30", true},
		{"specific page", PaginationFlags{Page: 2, PerPage: 10}, "2", "10", false},
		{"page only", PaginationFlags{Page: 3, PerPage: 0}, "3", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := url.Values{}
			tt.flags.Apply(q)

			if got := q.Get("page"); got != tt.wantP {
				t.Errorf("page = %q, want %q", got, tt.wantP)
			}

			if got := q.Get("per_page"); got != tt.wantPP {
				t.Errorf("per_page = %q, want %q", got, tt.wantPP)
			}

			if got := tt.flags.WantsAllPages(); got != tt.allPage {
				t.Errorf("WantsAllPages() = %v, want %v", got, tt.allPage)
			}
		})
	}
}

func TestAddQueryParam(t *testing.T) {
	t.Parallel()

	q := url.Values{}
	addQueryParam(q, "status", "open")
	addQueryParam(q, "empty", "")
	addQueryParam(q, "filter", "active")

	if got := q.Get("status"); got != "open" {
		t.Errorf("status = %q", got)
	}

	if q.Has("empty") {
		t.Error("empty param should not be set")
	}

	if got := q.Get("filter"); got != "active" {
		t.Errorf("filter = %q", got)
	}
}

func TestDefaultNewAPIClient_EnvVarBypassesKeyring(t *testing.T) {
	setupConfigDir(t)

	t.Setenv("NUBE_ACCESS_TOKEN", "env-token-abc")
	t.Setenv("NUBE_USER_ID", "42")

	// Call defaultNewAPIClient directly — it should return a client
	// without opening the keyring.
	client, err := defaultNewAPIClient(&RootFlags{})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestDefaultNewAPIClient_StandardPathUsesKeyring(t *testing.T) {
	setupConfigDir(t)

	// Ensure env vars are clear.
	t.Setenv("NUBE_ACCESS_TOKEN", "")
	t.Setenv("NUBE_USER_ID", "")

	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "u@test.com",
		UserID:      "123",
		AccessToken: "stored-token",
	})

	// Call defaultNewAPIClient directly — standard path should resolve
	// the account and token from the mock keyring.
	client, err := defaultNewAPIClient(&RootFlags{})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestExtractI18n(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		obj  map[string]any
		key  string
		want string
	}{
		{"plain string", map[string]any{"name": "Test"}, "name", "Test"},
		{"missing key", map[string]any{}, "name", ""},
		{"es preferred", map[string]any{"name": map[string]any{"es": "Hola", "pt": "Olá", "en": "Hello"}}, "name", "Hola"},
		{"fallback to pt", map[string]any{"name": map[string]any{"pt": "Olá", "en": "Hello"}}, "name", "Olá"},
		{"fallback to en", map[string]any{"name": map[string]any{"en": "Hello"}}, "name", "Hello"},
		{"fallback to first", map[string]any{"name": map[string]any{"fr": "Bonjour"}}, "name", "Bonjour"},
		{"empty i18n map", map[string]any{"name": map[string]any{}}, "name", ""},
		{"non-map non-string", map[string]any{"name": 42}, "name", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractI18n(tt.obj, tt.key); got != tt.want {
				t.Errorf("extractI18n() = %q, want %q", got, tt.want)
			}
		})
	}
}
