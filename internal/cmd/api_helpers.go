package cmd

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/credstore"
)

// newAPIClient composes store resolution + api.New.
// It is a package-level var so tests can swap it.
var newAPIClient = defaultNewAPIClient

func defaultNewAPIClient(flags *RootFlags) (*api.Client, error) {
	// Fast path: env-var token bypasses credential file entirely.
	if tok := os.Getenv("NUBE_ACCESS_TOKEN"); tok != "" {
		userID := os.Getenv("NUBE_USER_ID")
		if userID == "" {
			slog.Warn("NUBE_USER_ID not set; API calls that require a store ID will fail")
		}

		return api.New(userID, tok), nil
	}

	// Standard path: resolve store profile.
	_, profile, err := credstore.ResolveStore(flags.Store)
	if err != nil {
		return nil, &ExitErr{Code: ExitConfig, Err: err}
	}

	return api.New(profile.StoreID, profile.AccessToken), nil
}

// PaginationFlags embeds --page, --per-page for paginated list commands.
type PaginationFlags struct {
	Page    int `help:"Page number (omit to fetch all pages)" default:"0"`
	PerPage int `help:"Results per page" default:"30" aliases:"max"`
}

// Apply sets pagination query params. If Page is 0, the caller should use CollectAllPages.
func (p PaginationFlags) Apply(q url.Values) {
	if p.Page > 0 {
		q.Set("page", url.QueryEscape(itoa(p.Page)))
	}

	if p.PerPage > 0 {
		q.Set("per_page", url.QueryEscape(itoa(p.PerPage)))
	}
}

// WantsAllPages returns true when no explicit --page was given.
func (p PaginationFlags) WantsAllPages() bool {
	return p.Page <= 0
}

// addQueryParam sets key=value in q if value is non-empty.
func addQueryParam(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

// extractI18n returns the best available translation from an i18n map.
// Tienda Nube returns multilingual fields as {"es":"...","pt":"...","en":"..."}.
func extractI18n(obj map[string]any, key string) string {
	raw, ok := obj[key]
	if !ok {
		return ""
	}

	// If it's a plain string, return directly.
	if s, isStr := raw.(string); isStr {
		return s
	}

	m, isMap := raw.(map[string]any)
	if !isMap {
		return ""
	}

	// Prefer: es > pt > en > first available.
	for _, lang := range []string{"es", "pt", "en"} {
		if v, ok := m[lang]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}

	for _, v := range m {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}

	return ""
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
