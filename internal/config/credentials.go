package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var errMissingClientID = errors.New("stored credentials.json is missing client_id/client_secret")

// ClientCredentials holds the OAuth client ID and secret for a Tienda Nube app.
// Unlike Google's nested format, Tienda Nube uses flat JSON with client_id + client_secret.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"` //nolint:gosec // not a credential value
}

func WriteClientCredentials(c ClientCredentials) error {
	return WriteClientCredentialsFor(DefaultClientName, c)
}

func WriteClientCredentialsFor(client string, c ClientCredentials) error {
	_, err := EnsureDir()
	if err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	path, err := ClientCredentialsPathFor(client)
	if err != nil {
		return fmt.Errorf("resolve credentials path: %w", err)
	}

	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials json: %w", err)
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

func ReadClientCredentials() (ClientCredentials, error) {
	return ReadClientCredentialsFor(DefaultClientName)
}

func ReadClientCredentialsFor(client string) (ClientCredentials, error) {
	path, err := ClientCredentialsPathFor(client)
	if err != nil {
		return ClientCredentials{}, fmt.Errorf("resolve credentials path: %w", err)
	}

	var b []byte

	if b, err = os.ReadFile(path); err != nil { //nolint:gosec // user-provided path
		if os.IsNotExist(err) {
			return ClientCredentials{}, &CredentialsMissingError{Path: path, Cause: err}
		}

		return ClientCredentials{}, fmt.Errorf("read credentials: %w", err)
	}

	var c ClientCredentials
	if err := json.Unmarshal(b, &c); err != nil {
		return ClientCredentials{}, fmt.Errorf("decode credentials: %w", err)
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		return ClientCredentials{}, errMissingClientID
	}

	return c, nil
}

type CredentialsMissingError struct {
	Path  string
	Cause error
}

func (e *CredentialsMissingError) Error() string {
	return "oauth credentials missing"
}

func (e *CredentialsMissingError) Unwrap() error {
	return e.Cause
}
