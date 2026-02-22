package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListClientCredentials_Empty(t *testing.T) {
	setupConfigDir(t)

	// Ensure dir exists but is empty
	if _, err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	creds, err := ListClientCredentials()
	if err != nil {
		t.Fatalf("ListClientCredentials() error = %v", err)
	}

	if len(creds) != 0 {
		t.Errorf("expected empty, got %d", len(creds))
	}
}

func TestListClientCredentials_DefaultOnly(t *testing.T) {
	setupConfigDir(t)

	if _, err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	dir, _ := Dir()
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	creds, err := ListClientCredentials()
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if len(creds) != 1 {
		t.Fatalf("expected 1, got %d", len(creds))
	}

	if !creds[0].Default {
		t.Error("expected Default=true")
	}

	if creds[0].Client != DefaultClientName {
		t.Errorf("Client = %q, want %q", creds[0].Client, DefaultClientName)
	}
}

func TestListClientCredentials_Multiple(t *testing.T) {
	setupConfigDir(t)

	if _, err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	dir, _ := Dir()
	for _, name := range []string{"credentials.json", "credentials-beta.json", "credentials-alpha.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	creds, err := ListClientCredentials()
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if len(creds) != 3 {
		t.Fatalf("expected 3, got %d", len(creds))
	}

	// Should be sorted: alpha, beta, default
	if creds[0].Client != "alpha" {
		t.Errorf("creds[0].Client = %q, want alpha", creds[0].Client)
	}

	if creds[1].Client != "beta" {
		t.Errorf("creds[1].Client = %q, want beta", creds[1].Client)
	}

	if creds[2].Client != DefaultClientName {
		t.Errorf("creds[2].Client = %q, want %q", creds[2].Client, DefaultClientName)
	}
}

func TestListClientCredentials_NoDir(t *testing.T) {
	setupConfigDir(t)

	creds, err := ListClientCredentials()
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if creds != nil {
		t.Errorf("expected nil for missing dir, got %v", creds)
	}
}
