package config

import (
	"errors"
	"os"
	"testing"
)

func TestWriteReadCredentialsRoundtrip(t *testing.T) {
	setupConfigDir(t)

	want := ClientCredentials{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}

	if err := WriteClientCredentials(want); err != nil {
		t.Fatalf("WriteClientCredentials() error = %v", err)
	}

	got, err := ReadClientCredentials()
	if err != nil {
		t.Fatalf("ReadClientCredentials() error = %v", err)
	}

	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q", got.ClientID, want.ClientID)
	}

	if got.ClientSecret != want.ClientSecret {
		t.Errorf("ClientSecret = %q, want %q", got.ClientSecret, want.ClientSecret)
	}
}

func TestWriteReadCredentialsForNamedClient(t *testing.T) {
	setupConfigDir(t)

	want := ClientCredentials{
		ClientID:     "named-id",
		ClientSecret: "named-secret",
	}

	if err := WriteClientCredentialsFor("myapp", want); err != nil {
		t.Fatalf("WriteClientCredentialsFor() error = %v", err)
	}

	got, err := ReadClientCredentialsFor("myapp")
	if err != nil {
		t.Fatalf("ReadClientCredentialsFor() error = %v", err)
	}

	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q", got.ClientID, want.ClientID)
	}
}

func TestReadCredentialsMissing(t *testing.T) {
	setupConfigDir(t)

	_, err := ReadClientCredentials()
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}

	var credErr *CredentialsMissingError
	if !errors.As(err, &credErr) {
		t.Fatalf("expected CredentialsMissingError, got %T: %v", err, err)
	}

	if credErr.Path == "" {
		t.Error("CredentialsMissingError.Path should not be empty")
	}
}

func TestReadCredentialsInvalidJSON(t *testing.T) {
	setupConfigDir(t)

	if _, ensureErr := EnsureDir(); ensureErr != nil {
		t.Fatalf("EnsureDir: %v", ensureErr)
	}

	path, pathErr := ClientCredentialsPath()
	if pathErr != nil {
		t.Fatalf("path: %v", pathErr)
	}

	writeErr := os.WriteFile(path, []byte("{invalid json}"), 0o600)
	if writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}

	_, err := ReadClientCredentials()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadCredentialsMissingFields(t *testing.T) {
	setupConfigDir(t)

	if _, ensureErr := EnsureDir(); ensureErr != nil {
		t.Fatalf("EnsureDir: %v", ensureErr)
	}

	path, pathErr := ClientCredentialsPath()
	if pathErr != nil {
		t.Fatalf("path: %v", pathErr)
	}

	// Valid JSON but missing fields
	writeErr := os.WriteFile(path, []byte(`{"client_id": "id"}`), 0o600)
	if writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}

	_, err := ReadClientCredentials()
	if err == nil {
		t.Fatal("expected error for missing client_secret")
	}
}
