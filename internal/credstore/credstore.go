package credstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gberlati/nube-cli/internal/config"
)

// StoreProfile holds credentials and metadata for a single Tienda Nube store.
type StoreProfile struct {
	StoreID     string   `json:"store_id"`
	AccessToken string   `json:"access_token"` //nolint:gosec // G101: field name, not a credential
	Email       string   `json:"email,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

// OAuthClient holds the OAuth client ID and secret for a Tienda Nube app.
type OAuthClient struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"` //nolint:gosec // field name, not a credential
}

// File is the top-level credentials.json structure.
type File struct {
	DefaultStore string                  `json:"default_store,omitempty"`
	Stores       map[string]StoreProfile `json:"stores,omitempty"`
	OAuthClients map[string]OAuthClient  `json:"oauth_clients,omitempty"`
}

var (
	errNoStore        = errors.New("no store profile configured; run `nube login` first")
	errStoreNotFound  = errors.New("store profile not found")
	errAmbiguousStore = errors.New("multiple store profiles exist; use --store to select one")
)

// Path returns the path to credentials.json.
func Path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}

	return filepath.Join(dir, "credentials.json"), nil
}

// Read loads the credential file. Returns an empty File (not error) if it doesn't exist.
func Read() (File, error) {
	path, err := Path()
	if err != nil {
		return File{}, err
	}

	b, err := os.ReadFile(path) //nolint:gosec // credential file path
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}

		return File{}, fmt.Errorf("read credentials: %w", err)
	}

	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse credentials %s: %w", path, err)
	}

	return f, nil
}

// Write persists the credential file atomically with 0600 permissions.
func Write(f File) error {
	dir, err := config.EnsureDir()
	if err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	path := filepath.Join(dir, "credentials.json")

	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}

	b = append(b, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("commit credentials: %w", err)
	}

	return nil
}

// SetStore adds or updates a named store profile.
// If it's the only store, it automatically becomes the default.
func SetStore(name string, profile StoreProfile) error {
	f, err := Read()
	if err != nil {
		return err
	}

	if f.Stores == nil {
		f.Stores = make(map[string]StoreProfile)
	}

	f.Stores[name] = profile

	if len(f.Stores) == 1 {
		f.DefaultStore = name
	}

	return Write(f)
}

// GetStore returns a named store profile.
func GetStore(name string) (StoreProfile, error) {
	f, err := Read()
	if err != nil {
		return StoreProfile{}, err
	}

	p, ok := f.Stores[name]
	if !ok {
		return StoreProfile{}, fmt.Errorf("%w: %s", errStoreNotFound, name)
	}

	return p, nil
}

// RemoveStore deletes a named store profile and clears default if it matched.
func RemoveStore(name string) error {
	f, err := Read()
	if err != nil {
		return err
	}

	if _, ok := f.Stores[name]; !ok {
		return fmt.Errorf("%w: %s", errStoreNotFound, name)
	}

	delete(f.Stores, name)

	if f.DefaultStore == name {
		f.DefaultStore = ""

		if len(f.Stores) == 1 {
			for k := range f.Stores {
				f.DefaultStore = k
			}
		}
	}

	return Write(f)
}

// ResolveStore resolves the active store profile using the priority chain:
// --store flag → NUBE_STORE env → default_store → single-store auto-select.
// Returns (name, profile, error).
func ResolveStore(flagValue string) (string, StoreProfile, error) {
	name := flagValue
	if name == "" {
		name = os.Getenv("NUBE_STORE")
	}

	f, err := Read()
	if err != nil {
		return "", StoreProfile{}, err
	}

	if len(f.Stores) == 0 {
		return "", StoreProfile{}, errNoStore
	}

	if name != "" {
		p, ok := f.Stores[name]
		if !ok {
			return "", StoreProfile{}, fmt.Errorf("%w: %s", errStoreNotFound, name)
		}

		return name, p, nil
	}

	if f.DefaultStore != "" {
		if p, ok := f.Stores[f.DefaultStore]; ok {
			return f.DefaultStore, p, nil
		}
	}

	if len(f.Stores) == 1 {
		for k, v := range f.Stores {
			return k, v, nil
		}
	}

	return "", StoreProfile{}, errAmbiguousStore
}

// ListStores returns all store profile names, sorted.
func ListStores() ([]string, error) {
	f, err := Read()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(f.Stores))
	for k := range f.Stores {
		names = append(names, k)
	}

	sort.Strings(names)

	return names, nil
}

// SetDefault sets the default store profile name.
func SetDefault(name string) error {
	f, err := Read()
	if err != nil {
		return err
	}

	if _, ok := f.Stores[name]; !ok {
		return fmt.Errorf("%w: %s", errStoreNotFound, name)
	}

	f.DefaultStore = name

	return Write(f)
}

// GetOAuthClient returns a named OAuth client credential set.
func GetOAuthClient(name string) (OAuthClient, error) {
	f, err := Read()
	if err != nil {
		return OAuthClient{}, err
	}

	if f.OAuthClients == nil {
		return OAuthClient{}, &OAuthClientMissingError{Name: name}
	}

	c, ok := f.OAuthClients[name]
	if !ok {
		return OAuthClient{}, &OAuthClientMissingError{Name: name}
	}

	return c, nil
}

// SetOAuthClient adds or updates a named OAuth client.
func SetOAuthClient(name string, client OAuthClient) error {
	f, err := Read()
	if err != nil {
		return err
	}

	if f.OAuthClients == nil {
		f.OAuthClients = make(map[string]OAuthClient)
	}

	f.OAuthClients[name] = client

	return Write(f)
}

// OAuthClientMissingError is returned when no OAuth client credentials are found.
type OAuthClientMissingError struct {
	Name string
}

func (e *OAuthClientMissingError) Error() string {
	return "oauth client credentials missing"
}
