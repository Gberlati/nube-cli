package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gberlati/nube-cli/internal/config"
	"github.com/gberlati/nube-cli/internal/oauth"
	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/secrets"
	"github.com/gberlati/nube-cli/internal/ui"
)

var authorizeOAuth = oauth.Authorize

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// AuthCmd is the top-level auth command.
type AuthCmd struct {
	Credentials AuthCredentialsCmd `cmd:"" name:"credentials" help:"Manage OAuth client credentials"`
	Add         AuthAddCmd         `cmd:"" name:"add" help:"Authorize and store an access token"`
	List        AuthListCmd        `cmd:"" name:"list" help:"List stored accounts"`
	Aliases     AuthAliasCmd       `cmd:"" name:"alias" help:"Manage account aliases"`
	Status      AuthStatusCmd      `cmd:"" name:"status" help:"Show auth configuration and keyring backend"`
	Remove      AuthRemoveCmd      `cmd:"" name:"remove" help:"Remove a stored access token"`
	Tokens      AuthTokensCmd      `cmd:"" name:"tokens" help:"Manage stored access tokens"`
}

// --- Credentials ---

type AuthCredentialsCmd struct {
	Set  AuthCredentialsSetCmd  `cmd:"" default:"withargs" help:"Store OAuth client credentials"`
	List AuthCredentialsListCmd `cmd:"" name:"list" help:"List stored OAuth client credentials"`
}

type AuthCredentialsSetCmd struct {
	Path string `arg:"" name:"credentials" help:"Path to credentials.json or '-' for stdin"`
}

func (c *AuthCredentialsSetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := config.NormalizeClientNameOrDefault(flags.Client)
	if err != nil {
		return err
	}

	inPath := c.Path

	var b []byte
	if inPath == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		inPath, err = config.ExpandPath(inPath)
		if err != nil {
			return err
		}

		b, err = os.ReadFile(inPath) //nolint:gosec // user-provided path
	}

	if err != nil {
		return err
	}

	var creds config.ClientCredentials
	if parseErr := parseClientCredentials(b, &creds); parseErr != nil {
		return parseErr
	}

	if err := config.WriteClientCredentialsFor(client, creds); err != nil {
		return err
	}

	outPath, _ := config.ClientCredentialsPathFor(client)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"saved":  true,
			"path":   outPath,
			"client": client,
		})
	}

	u.Out().Printf("path\t%s", outPath)
	u.Out().Printf("client\t%s", client)

	return nil
}

func parseClientCredentials(b []byte, creds *config.ClientCredentials) error {
	// Tienda Nube uses flat JSON: {"client_id": "...", "client_secret": "..."}
	if err := json.Unmarshal(b, creds); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	if creds.ClientID == "" || creds.ClientSecret == "" {
		return fmt.Errorf("credentials.json must contain client_id and client_secret")
	}

	return nil
}

type AuthCredentialsListCmd struct{}

func (c *AuthCredentialsListCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	creds, err := config.ListClientCredentials()
	if err != nil {
		return err
	}

	type entry struct {
		Client  string `json:"client"`
		Path    string `json:"path,omitempty"`
		Default bool   `json:"default"`
	}

	entries := make([]entry, 0, len(creds))
	for _, info := range creds {
		entries = append(entries, entry{
			Client:  info.Client,
			Path:    info.Path,
			Default: info.Default,
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Client < entries[j].Client })

	if len(entries) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"clients": []entry{}})
		}

		u.Err().Println("No OAuth client credentials stored")

		return nil
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"clients": entries})
	}

	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "CLIENT\tPATH")

	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", e.Client, e.Path)
	}

	return nil
}

// --- Auth Add ---

type AuthAddCmd struct {
	Email     string        `arg:"" name:"email" help:"Account email"`
	Timeout   time.Duration `name:"timeout" help:"Authorization timeout" default:"5m"`
	BrokerURL string        `name:"broker-url" help:"OAuth broker URL (overrides default)" env:"NUBE_AUTH_BROKER"`
}

func (c *AuthAddCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	email := normalizeEmail(c.Email)
	if email == "" {
		return usagef("empty email")
	}

	client, err := config.NormalizeClientNameOrDefault(flags.Client)
	if err != nil {
		return err
	}

	tok, err := authorizeOAuth(ctx, oauth.AuthorizeOptions{
		Timeout:   c.Timeout,
		Client:    client,
		BrokerURL: c.BrokerURL,
	})
	if err != nil {
		return err
	}

	store, storeErr := openSecretsStore()
	if storeErr != nil {
		return storeErr
	}

	userID := tok.UserID.String()

	var scopes []string
	if tok.Scope != "" {
		scopes = strings.Split(tok.Scope, " ")
	}

	if err := store.SetToken(client, email, secrets.Token{
		Client:      client,
		Email:       email,
		UserID:      userID,
		Scopes:      scopes,
		AccessToken: tok.AccessToken,
	}); err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"stored":  true,
			"email":   email,
			"user_id": userID,
			"client":  client,
			"scopes":  scopes,
		})
	}

	u.Out().Printf("email\t%s", email)
	u.Out().Printf("user_id\t%s", userID)
	u.Out().Printf("client\t%s", client)

	return nil
}

// --- Auth List ---

type AuthListCmd struct{}

func (c *AuthListCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	store, err := openSecretsStore()
	if err != nil {
		return err
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return err
	}

	sort.Slice(tokens, func(i, j int) bool { return tokens[i].Email < tokens[j].Email })

	if outfmt.IsJSON(ctx) {
		type item struct {
			Email     string   `json:"email"`
			Client    string   `json:"client,omitempty"`
			UserID    string   `json:"user_id,omitempty"`
			Scopes    []string `json:"scopes,omitempty"`
			CreatedAt string   `json:"created_at,omitempty"`
		}

		out := make([]item, 0, len(tokens))
		for _, t := range tokens {
			created := ""
			if !t.CreatedAt.IsZero() {
				created = t.CreatedAt.UTC().Format(time.RFC3339)
			}

			out = append(out, item{
				Email:     t.Email,
				Client:    t.Client,
				UserID:    t.UserID,
				Scopes:    t.Scopes,
				CreatedAt: created,
			})
		}

		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"accounts": out})
	}

	if len(tokens) == 0 {
		u.Err().Println("No tokens stored")
		return nil
	}

	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "EMAIL\tCLIENT\tSTORE ID\tCREATED")

	for _, t := range tokens {
		created := ""
		if !t.CreatedAt.IsZero() {
			created = t.CreatedAt.UTC().Format(time.RFC3339)
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Email, t.Client, t.UserID, created)
	}

	return nil
}

// --- Auth Status ---

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	configPath, err := config.ConfigPath()
	if err != nil {
		return err
	}

	configExists, err := config.ConfigExists()
	if err != nil {
		return err
	}

	backendInfo, err := secrets.ResolveKeyringBackendInfo()
	if err != nil {
		return err
	}

	account := ""
	client := ""
	credentialsPath := ""
	credentialsExists := false

	if flags != nil {
		if a, acctErr := requireAccount(flags); acctErr == nil {
			account = a
			resolvedClient, resolveErr := config.NormalizeClientNameOrDefault(flags.Client)

			if resolveErr == nil {
				client = resolvedClient

				path, pathErr := config.ClientCredentialsPathFor(client)
				if pathErr == nil {
					credentialsPath = path
					if st, statErr := os.Stat(path); statErr == nil && !st.IsDir() {
						credentialsExists = true
					}
				}
			}
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"config": map[string]any{
				"path":   configPath,
				"exists": configExists,
			},
			"keyring": map[string]any{
				"backend": backendInfo.Value,
				"source":  backendInfo.Source,
			},
			"account": map[string]any{
				"email":              account,
				"client":             client,
				"credentials_path":   credentialsPath,
				"credentials_exists": credentialsExists,
			},
		})
	}

	u.Out().Printf("config_path\t%s", configPath)
	u.Out().Printf("config_exists\t%t", configExists)
	u.Out().Printf("keyring_backend\t%s", backendInfo.Value)
	u.Out().Printf("keyring_backend_source\t%s", backendInfo.Source)

	if account != "" {
		u.Out().Printf("account\t%s", account)
		u.Out().Printf("client\t%s", client)

		if credentialsPath != "" {
			u.Out().Printf("credentials_path\t%s", credentialsPath)
		}

		u.Out().Printf("credentials_exists\t%t", credentialsExists)
	}

	return nil
}

// --- Auth Remove ---

type AuthRemoveCmd struct {
	Email string `arg:"" name:"email" help:"Account email"`
}

func (c *AuthRemoveCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	email := normalizeEmail(c.Email)
	if email == "" {
		return usagef("empty email")
	}

	if err := confirmDestructive(flags, fmt.Sprintf("remove stored token for %s", email)); err != nil {
		return err
	}

	store, err := openSecretsStore()
	if err != nil {
		return err
	}

	client, clientErr := config.NormalizeClientNameOrDefault(flags.Client)
	if clientErr != nil {
		return clientErr
	}

	if err := store.DeleteToken(client, email); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("deleted", true),
		kv("email", email),
		kv("client", client),
	)
}

// --- Auth Tokens ---

type AuthTokensCmd struct {
	List   AuthTokensListCmd   `cmd:"" name:"list" help:"List stored tokens (by key only)"`
	Delete AuthTokensDeleteCmd `cmd:"" name:"delete" help:"Delete a stored access token"`
}

type AuthTokensListCmd struct{}

func (c *AuthTokensListCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	store, err := openSecretsStore()
	if err != nil {
		return err
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if strings.TrimSpace(tok.Email) == "" {
			continue
		}

		filtered = append(filtered, secrets.TokenKey(tok.Client, tok.Email))
	}

	sort.Strings(filtered)

	if len(filtered) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"keys": []string{}})
		}

		u.Err().Println("No tokens stored")

		return nil
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"keys": filtered})
	}

	for _, k := range filtered {
		u.Out().Println(k)
	}

	return nil
}

type AuthTokensDeleteCmd struct {
	Email string `arg:"" name:"email" help:"Account email"`
}

func (c *AuthTokensDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	email := strings.TrimSpace(c.Email)
	if email == "" {
		return usagef("empty email")
	}

	if err := confirmDestructive(flags, fmt.Sprintf("delete stored token for %s", email)); err != nil {
		return err
	}

	store, err := openSecretsStore()
	if err != nil {
		return err
	}

	client, clientErr := config.NormalizeClientNameOrDefault(flags.Client)
	if clientErr != nil {
		return clientErr
	}

	if err := store.DeleteToken(client, email); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("deleted", true),
		kv("email", email),
		kv("client", client),
	)
}
