package cmd

import (
	"os"
	"strings"

	"github.com/gberlati/nube-cli/internal/config"
	"github.com/gberlati/nube-cli/internal/secrets"
)

var openSecretsStore = secrets.OpenDefault

func requireAccount(flags *RootFlags) (string, error) {
	store, _ := openSecretsStore()
	return requireAccountWithStore(flags, store)
}

// requireAccountWithStore resolves the active account email.
// If store is nil, keyring-based fallback (default account, single-token) is skipped.
func requireAccountWithStore(flags *RootFlags, store secrets.Store) (string, error) {
	client := config.DefaultClientName

	var err error

	if flags != nil {
		client, err = config.NormalizeClientNameOrDefault(flags.Client)
	}

	if err != nil {
		return "", err
	}

	if v := strings.TrimSpace(flags.Account); v != "" {
		if resolved, ok, resolveErr := resolveAccountAlias(v); resolveErr != nil {
			return "", resolveErr
		} else if ok {
			return resolved, nil
		}

		if shouldAutoSelectAccount(v) {
			v = ""
		}

		if v != "" {
			return v, nil
		}
	}

	if v := strings.TrimSpace(os.Getenv("NUBE_ACCOUNT")); v != "" {
		if resolved, ok, resolveErr := resolveAccountAlias(v); resolveErr != nil {
			return "", resolveErr
		} else if ok {
			return resolved, nil
		}

		if shouldAutoSelectAccount(v) {
			v = ""
		}

		if v != "" {
			return v, nil
		}
	}

	if store != nil {
		if defaultEmail, defErr := store.GetDefaultAccount(client); defErr == nil {
			defaultEmail = strings.TrimSpace(defaultEmail)
			if defaultEmail != "" {
				return defaultEmail, nil
			}
		}

		if toks, listErr := store.ListTokens(); listErr == nil {
			filtered := make([]secrets.Token, 0, len(toks))
			for _, tok := range toks {
				if strings.TrimSpace(tok.Email) == "" {
					continue
				}

				if tok.Client == client {
					filtered = append(filtered, tok)
				}
			}

			if len(filtered) == 1 {
				if v := strings.TrimSpace(filtered[0].Email); v != "" {
					return v, nil
				}
			}

			if len(filtered) == 0 && len(toks) == 1 {
				if v := strings.TrimSpace(toks[0].Email); v != "" {
					return v, nil
				}
			}
		}
	}

	return "", usagef("missing --account (or set NUBE_ACCOUNT, or store exactly one token via `nube auth add`)")
}

func resolveAccountAlias(value string) (string, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "@") || shouldAutoSelectAccount(value) {
		return "", false, nil
	}

	return config.ResolveAccountAlias(value)
}

func shouldAutoSelectAccount(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto", "default":
		return true
	default:
		return false
	}
}
