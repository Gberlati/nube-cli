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

	"github.com/gberlati/nube-cli/internal/credstore"
	"github.com/gberlati/nube-cli/internal/oauth"
	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

var authorizeOAuth = oauth.Authorize

// AuthCmd is the top-level auth command.
type AuthCmd struct {
	Credentials AuthCredentialsCmd `cmd:"" name:"credentials" help:"Manage OAuth client credentials"`
	List        AuthListCmd        `cmd:"" name:"list" help:"List store profiles"`
	Status      AuthStatusCmd      `cmd:"" name:"status" help:"Show auth configuration"`
	Token       AuthTokenCmd       `cmd:"" name:"token" help:"Print access token for a store profile"`
	Default     AuthDefaultCmd     `cmd:"" name:"default" help:"Set default store profile"`
}

// --- Login (top-level) ---

type LoginCmd struct {
	Name      string        `arg:"" optional:"" name:"name" help:"Profile name (auto-generated if omitted)"`
	Timeout   time.Duration `name:"timeout" help:"Authorization timeout" default:"5m"`
	BrokerURL string        `name:"broker-url" help:"OAuth broker URL (overrides default)" env:"NUBE_AUTH_BROKER"`
}

func (c *LoginCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	tok, err := authorizeOAuth(ctx, oauth.AuthorizeOptions{
		Timeout:   c.Timeout,
		OAuthApp:  "default",
		BrokerURL: c.BrokerURL,
	})
	if err != nil {
		return err
	}

	userID := tok.UserID.String()

	var scopes []string
	if tok.Scope != "" {
		scopes = strings.Split(tok.Scope, " ")
	}

	name := strings.TrimSpace(c.Name)
	if name == "" {
		name = "store-" + userID
	}

	profile := credstore.StoreProfile{
		StoreID:     userID,
		AccessToken: tok.AccessToken,
		Scopes:      scopes,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := credstore.SetStore(name, profile); err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"stored":   true,
			"name":     name,
			"store_id": userID,
			"scopes":   scopes,
		})
	}

	u.Out().Printf("name\t%s", name)
	u.Out().Printf("store_id\t%s", userID)

	return nil
}

// --- Logout (top-level) ---

type LogoutCmd struct {
	Name string `arg:"" name:"name" help:"Profile name to remove"`
}

func (c *LogoutCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	name := strings.TrimSpace(c.Name)
	if name == "" {
		return usagef("profile name required")
	}

	if err := confirmDestructive(flags, fmt.Sprintf("remove store profile %q", name)); err != nil {
		return err
	}

	if err := credstore.RemoveStore(name); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("deleted", true),
		kv("name", name),
	)
}

// --- Credentials ---

type AuthCredentialsCmd struct {
	Set  AuthCredentialsSetCmd  `cmd:"" default:"withargs" help:"Store OAuth client credentials"`
	List AuthCredentialsListCmd `cmd:"" name:"list" help:"List stored OAuth client credentials"`
}

type AuthCredentialsSetCmd struct {
	Path string `arg:"" name:"credentials" help:"Path to credentials.json or '-' for stdin"`
}

func (c *AuthCredentialsSetCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	var (
		b   []byte
		err error
	)

	inPath := c.Path

	if inPath == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		inPath, err = expandPath(inPath)
		if err != nil {
			return err
		}

		b, err = os.ReadFile(inPath) //nolint:gosec // user-provided path
	}

	if err != nil {
		return err
	}

	var creds struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"` //nolint:gosec // field name
	}

	if parseErr := json.Unmarshal(b, &creds); parseErr != nil {
		return fmt.Errorf("parse credentials: %w", parseErr)
	}

	if creds.ClientID == "" || creds.ClientSecret == "" {
		return fmt.Errorf("credentials.json must contain client_id and client_secret")
	}

	if err := credstore.SetOAuthClient("default", credstore.OAuthClient{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
	}); err != nil {
		return err
	}

	credPath, _ := credstore.Path()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"saved": true,
			"path":  credPath,
		})
	}

	u.Out().Printf("path\t%s", credPath)

	return nil
}

// expandPath expands ~ at the beginning of a path to the user's home directory.
func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand home dir: %w", err)
		}

		if path == "~" {
			return home, nil
		}

		return home + path[1:], nil
	}

	return path, nil
}

type AuthCredentialsListCmd struct{}

func (c *AuthCredentialsListCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	f, err := credstore.Read()
	if err != nil {
		return err
	}

	if len(f.OAuthClients) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"clients": []any{}})
		}

		u.Err().Println("No OAuth client credentials stored")

		return nil
	}

	type entry struct {
		Name string `json:"name"`
	}

	entries := make([]entry, 0, len(f.OAuthClients))
	for k := range f.OAuthClients {
		entries = append(entries, entry{Name: k})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"clients": entries})
	}

	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "NAME")

	for _, e := range entries {
		_, _ = fmt.Fprintln(w, e.Name)
	}

	return nil
}

// --- Auth List ---

type AuthListCmd struct{}

func (c *AuthListCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	f, err := credstore.Read()
	if err != nil {
		return err
	}

	type item struct {
		Name      string   `json:"name"`
		StoreID   string   `json:"store_id"`
		Email     string   `json:"email,omitempty"`
		Scopes    []string `json:"scopes,omitempty"`
		CreatedAt string   `json:"created_at,omitempty"`
		Default   bool     `json:"default"`
	}

	items := make([]item, 0, len(f.Stores))
	for name, p := range f.Stores {
		items = append(items, item{
			Name:      name,
			StoreID:   p.StoreID,
			Email:     p.Email,
			Scopes:    p.Scopes,
			CreatedAt: p.CreatedAt,
			Default:   name == f.DefaultStore,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"stores": items})
	}

	if len(items) == 0 {
		u.Err().Println("No store profiles configured")
		return nil
	}

	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "NAME\tSTORE ID\tDEFAULT\tCREATED")

	for _, it := range items {
		def := ""
		if it.Default {
			def = "*"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", it.Name, it.StoreID, def, it.CreatedAt)
	}

	return nil
}

// --- Auth Status ---

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	credPath, err := credstore.Path()
	if err != nil {
		return err
	}

	credExists := false
	if st, statErr := os.Stat(credPath); statErr == nil && !st.IsDir() {
		credExists = true
	}

	storeName := ""
	storeID := ""

	if flags != nil {
		if name, profile, resolveErr := credstore.ResolveStore(flags.Store); resolveErr == nil {
			storeName = name
			storeID = profile.StoreID
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"credentials": map[string]any{
				"path":   credPath,
				"exists": credExists,
			},
			"store": map[string]any{
				"name":     storeName,
				"store_id": storeID,
			},
		})
	}

	u.Out().Printf("credentials_path\t%s", credPath)
	u.Out().Printf("credentials_exists\t%t", credExists)

	if storeName != "" {
		u.Out().Printf("store\t%s", storeName)
		u.Out().Printf("store_id\t%s", storeID)
	}

	return nil
}

// --- Auth Token ---

type AuthTokenCmd struct {
	Name string `arg:"" optional:"" name:"name" help:"Store profile name (uses default if omitted)"`
}

func (c *AuthTokenCmd) Run(ctx context.Context, flags *RootFlags) error {
	name := strings.TrimSpace(c.Name)

	// Use flag override if name not given as argument.
	flagStore := ""
	if flags != nil {
		flagStore = flags.Store
	}

	if name != "" {
		flagStore = name
	}

	resolvedName, profile, err := credstore.ResolveStore(flagStore)
	if err != nil {
		return &ExitErr{Code: ExitConfig, Err: err}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"access_token": profile.AccessToken,
			"store_id":     profile.StoreID,
			"name":         resolvedName,
		})
	}

	// Plain: just the token, suitable for $(nube auth token ...)
	fmt.Fprintln(os.Stdout, profile.AccessToken)

	return nil
}

// --- Auth Default ---

type AuthDefaultCmd struct {
	Name string `arg:"" name:"name" help:"Store profile name to set as default"`
}

func (c *AuthDefaultCmd) Run(ctx context.Context, _ *RootFlags) error {
	u := ui.FromContext(ctx)

	name := strings.TrimSpace(c.Name)
	if name == "" {
		return usagef("profile name required")
	}

	if err := credstore.SetDefault(name); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("default", name),
	)
}
