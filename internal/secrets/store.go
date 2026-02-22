package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/term"

	"github.com/gberlati/nube-cli/internal/config"
)

type Store interface {
	Keys() ([]string, error)
	SetToken(client string, email string, tok Token) error
	GetToken(client string, email string) (Token, error)
	DeleteToken(client string, email string) error
	ListTokens() ([]Token, error)
	GetDefaultAccount(client string) (string, error)
	SetDefaultAccount(client string, email string) error
}

type KeyringStore struct {
	ring keyring.Keyring
}

// Token represents a stored Tienda Nube access token.
// Unlike Google's refresh tokens, Tienda Nube tokens are permanent
// (invalidated only on re-auth or app uninstall).
type Token struct {
	Client      string    `json:"client,omitempty"`
	Email       string    `json:"email"`
	UserID      string    `json:"user_id,omitempty"` // Tienda Nube store ID
	Scopes      []string  `json:"scopes,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	AccessToken string    `json:"-"`
}

func keyringItem(key string, data []byte) keyring.Item {
	return keyring.Item{
		Key:   key,
		Data:  data,
		Label: config.AppName,
	}
}

const (
	keyringPasswordEnv = "NUBE_KEYRING_PASSWORD" //nolint:gosec // env var name, not a credential
	keyringBackendEnv  = "NUBE_KEYRING_BACKEND"  //nolint:gosec // env var name, not a credential
)

var (
	errMissingEmail        = errors.New("missing email")
	errMissingAccessToken  = errors.New("missing access token")
	errNoTTY               = errors.New("no TTY available for keyring file backend password prompt")
	errInvalidKeyringBkend = errors.New("invalid keyring backend")
	errKeyringTimeout      = errors.New("keyring connection timed out")
	openKeyringFunc        = openKeyring
	keyringOpenFunc        = keyring.Open
)

type KeyringBackendInfo struct {
	Value  string
	Source string
}

const (
	keyringBackendSourceEnv     = "env"
	keyringBackendSourceConfig  = "config"
	keyringBackendSourceDefault = "default"
	keyringBackendAuto          = "auto"
)

func ResolveKeyringBackendInfo() (KeyringBackendInfo, error) {
	if v := normalizeKeyringBackend(os.Getenv(keyringBackendEnv)); v != "" {
		return KeyringBackendInfo{Value: v, Source: keyringBackendSourceEnv}, nil
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return KeyringBackendInfo{}, fmt.Errorf("resolve keyring backend: %w", err)
	}

	if cfg.KeyringBackend != "" {
		if v := normalizeKeyringBackend(cfg.KeyringBackend); v != "" {
			return KeyringBackendInfo{Value: v, Source: keyringBackendSourceConfig}, nil
		}
	}

	return KeyringBackendInfo{Value: keyringBackendAuto, Source: keyringBackendSourceDefault}, nil
}

func allowedBackends(info KeyringBackendInfo) ([]keyring.BackendType, error) {
	switch info.Value {
	case "", keyringBackendAuto:
		return nil, nil
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}, nil
	case "file":
		return []keyring.BackendType{keyring.FileBackend}, nil
	default:
		return nil, fmt.Errorf("%w: %q (expected %s, keychain, or file)", errInvalidKeyringBkend, info.Value, keyringBackendAuto)
	}
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	password, passwordSet := os.LookupEnv(keyringPasswordEnv)

	if passwordSet {
		return keyring.FixedStringPrompt(password)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // fd conversion is safe
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("%w; set %s", errNoTTY, keyringPasswordEnv)
	}
}

func normalizeKeyringBackend(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

const keyringOpenTimeout = 5 * time.Second

func shouldForceFileBackend(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == keyringBackendAuto && dbusAddr == ""
}

func shouldUseKeyringTimeout(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == keyringBackendAuto && dbusAddr != ""
}

func openKeyring() (keyring.Keyring, error) {
	keyringDir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, fmt.Errorf("ensure keyring dir: %w", err)
	}

	backendInfo, err := ResolveKeyringBackendInfo()
	if err != nil {
		return nil, err
	}

	backends, err := allowedBackends(backendInfo)
	if err != nil {
		return nil, err
	}

	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")

	if shouldForceFileBackend(runtime.GOOS, backendInfo, dbusAddr) {
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	cfg := keyring.Config{
		ServiceName:              config.AppName,
		KeychainTrustApplication: false,
		AllowedBackends:          backends,
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	}

	if shouldUseKeyringTimeout(runtime.GOOS, backendInfo, dbusAddr) {
		return openKeyringWithTimeout(cfg, keyringOpenTimeout)
	}

	ring, err := keyringOpenFunc(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return ring, nil
}

type keyringResult struct {
	ring keyring.Keyring
	err  error
}

func openKeyringWithTimeout(cfg keyring.Config, timeout time.Duration) (keyring.Keyring, error) {
	ch := make(chan keyringResult, 1)

	go func() {
		ring, err := keyringOpenFunc(cfg)
		ch <- keyringResult{ring, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("open keyring: %w", res.err)
		}

		return res.ring, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w after %v (D-Bus SecretService may be unresponsive); "+
			"set NUBE_KEYRING_BACKEND=file and NUBE_KEYRING_PASSWORD=<password> to use encrypted file storage instead",
			errKeyringTimeout, timeout)
	}
}

func OpenDefault() (Store, error) {
	ring, err := openKeyringFunc()
	if err != nil {
		return nil, err
	}

	return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) Keys() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("list keyring keys: %w", err)
	}

	return keys, nil
}

type storedToken struct {
	AccessToken string    `json:"access_token"` //nolint:gosec // not a credential value
	UserID      string    `json:"user_id,omitempty"`
	Scopes      []string  `json:"scopes,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

func (s *KeyringStore) SetToken(client string, email string, tok Token) error {
	email = normalize(email)
	if email == "" {
		return errMissingEmail
	}

	if tok.AccessToken == "" {
		return errMissingAccessToken
	}

	normalizedClient, err := normalizeClient(client)
	if err != nil {
		return err
	}

	if tok.CreatedAt.IsZero() {
		tok.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(storedToken{
		AccessToken: tok.AccessToken,
		UserID:      tok.UserID,
		Scopes:      tok.Scopes,
		CreatedAt:   tok.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	if err := s.ring.Set(keyringItem(tokenKey(normalizedClient, email), payload)); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	if normalizedClient == config.DefaultClientName {
		if err := s.ring.Set(keyringItem(legacyTokenKey(email), payload)); err != nil {
			return fmt.Errorf("store legacy token: %w", err)
		}
	}

	return nil
}

func (s *KeyringStore) GetToken(client string, email string) (Token, error) {
	email = normalize(email)
	if email == "" {
		return Token{}, errMissingEmail
	}

	normalizedClient, err := normalizeClient(client)
	if err != nil {
		return Token{}, err
	}

	item, err := s.ring.Get(tokenKey(normalizedClient, email))
	if err != nil {
		if normalizedClient == config.DefaultClientName {
			if legacyItem, legacyErr := s.ring.Get(legacyTokenKey(email)); legacyErr == nil {
				item = legacyItem

				if migrateErr := s.ring.Set(keyringItem(tokenKey(normalizedClient, email), legacyItem.Data)); migrateErr != nil {
					return Token{}, fmt.Errorf("migrate token: %w", migrateErr)
				}
			} else {
				return Token{}, fmt.Errorf("read token: %w", err)
			}
		} else {
			return Token{}, fmt.Errorf("read token: %w", err)
		}
	}

	var st storedToken
	if err := json.Unmarshal(item.Data, &st); err != nil {
		return Token{}, fmt.Errorf("decode token: %w", err)
	}

	return Token{
		Client:      normalizedClient,
		Email:       email,
		UserID:      st.UserID,
		Scopes:      st.Scopes,
		CreatedAt:   st.CreatedAt,
		AccessToken: st.AccessToken,
	}, nil
}

func (s *KeyringStore) DeleteToken(client string, email string) error {
	email = normalize(email)
	if email == "" {
		return errMissingEmail
	}

	normalizedClient, err := normalizeClient(client)
	if err != nil {
		return err
	}

	if err := s.ring.Remove(tokenKey(normalizedClient, email)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("delete token: %w", err)
	}

	if normalizedClient == config.DefaultClientName {
		if err := s.ring.Remove(legacyTokenKey(email)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			return fmt.Errorf("delete legacy token: %w", err)
		}
	}

	return nil
}

func (s *KeyringStore) ListTokens() ([]Token, error) {
	keys, err := s.Keys()
	if err != nil {
		return nil, fmt.Errorf("list tokens: %w", err)
	}

	out := make([]Token, 0)
	seen := make(map[string]struct{})

	for _, k := range keys {
		client, email, ok := ParseTokenKey(k)
		if !ok {
			continue
		}

		key := client + "\n" + email
		if _, ok := seen[key]; ok {
			continue
		}

		tok, err := s.GetToken(client, email)
		if err != nil {
			return nil, fmt.Errorf("read token for %s: %w", email, err)
		}

		seen[key] = struct{}{}

		out = append(out, tok)
	}

	return out, nil
}

func ParseTokenKey(k string) (client string, email string, ok bool) {
	const prefix = "token:"
	if !strings.HasPrefix(k, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(k, prefix)

	if strings.TrimSpace(rest) == "" {
		return "", "", false
	}

	if !strings.Contains(rest, ":") {
		return config.DefaultClientName, rest, true
	}

	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// TokenKey returns the keyring key for a client+email pair.
func TokenKey(client string, email string) string {
	return fmt.Sprintf("token:%s:%s", client, email)
}

func tokenKey(client string, email string) string {
	return TokenKey(client, email)
}

func legacyTokenKey(email string) string {
	return fmt.Sprintf("token:%s", email)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normalizeClient(raw string) (string, error) {
	client, err := config.NormalizeClientNameOrDefault(raw)
	if err != nil {
		return "", fmt.Errorf("normalize client: %w", err)
	}

	return client, nil
}

const defaultAccountKey = "default_account"

func defaultAccountKeyForClient(client string) string {
	return fmt.Sprintf("default_account:%s", client)
}

func (s *KeyringStore) GetDefaultAccount(client string) (string, error) {
	normalizedClient, err := normalizeClient(client)
	if err != nil {
		return "", err
	}

	if normalizedClient != "" {
		if it, getErr := s.ring.Get(defaultAccountKeyForClient(normalizedClient)); getErr == nil {
			return string(it.Data), nil
		} else if !errors.Is(getErr, keyring.ErrKeyNotFound) {
			return "", fmt.Errorf("read default account: %w", getErr)
		}
	}

	it, err := s.ring.Get(defaultAccountKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", nil
		}

		return "", fmt.Errorf("read default account: %w", err)
	}

	return string(it.Data), nil
}

func (s *KeyringStore) SetDefaultAccount(client string, email string) error {
	email = normalize(email)
	if email == "" {
		return errMissingEmail
	}

	normalizedClient, err := normalizeClient(client)
	if err != nil {
		return err
	}

	if normalizedClient != "" {
		if err := s.ring.Set(keyringItem(defaultAccountKeyForClient(normalizedClient), []byte(email))); err != nil {
			return fmt.Errorf("store default account: %w", err)
		}
	}

	if err := s.ring.Set(keyringItem(defaultAccountKey, []byte(email))); err != nil {
		return fmt.Errorf("store default account: %w", err)
	}

	return nil
}
