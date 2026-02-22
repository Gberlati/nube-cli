package config

import (
	"os"
	"testing"
)

func TestReadConfig_NoFile(t *testing.T) {
	setupConfigDir(t)

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if cfg.KeyringBackend != "" {
		t.Errorf("KeyringBackend = %q, want empty", cfg.KeyringBackend)
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	setupConfigDir(t)

	want := File{
		KeyringBackend: "file",
		AccountAliases: map[string]string{"prod": "prod@example.com"},
	}

	if writeErr := WriteConfig(want); writeErr != nil {
		t.Fatalf("WriteConfig() error = %v", writeErr)
	}

	got, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if got.KeyringBackend != want.KeyringBackend {
		t.Errorf("KeyringBackend = %q, want %q", got.KeyringBackend, want.KeyringBackend)
	}

	if got.AccountAliases["prod"] != "prod@example.com" {
		t.Errorf("AccountAliases[prod] = %q, want %q", got.AccountAliases["prod"], "prod@example.com")
	}
}

func TestConfigExists(t *testing.T) {
	setupConfigDir(t)

	exists, err := ConfigExists()
	if err != nil {
		t.Fatalf("ConfigExists() error = %v", err)
	}

	if exists {
		t.Error("ConfigExists() should be false before write")
	}

	writeErr := WriteConfig(File{})
	if writeErr != nil {
		t.Fatalf("WriteConfig() error = %v", writeErr)
	}

	exists, err = ConfigExists()
	if err != nil {
		t.Fatalf("ConfigExists() error = %v", err)
	}

	if !exists {
		t.Error("ConfigExists() should be true after write")
	}
}

func TestJSON5Parsing(t *testing.T) {
	setupConfigDir(t)

	// Ensure dir exists before writing directly
	if _, ensureErr := EnsureDir(); ensureErr != nil {
		t.Fatalf("EnsureDir() error = %v", ensureErr)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	// JSON5 with comments and trailing comma
	content := `{
  // This is a comment
  "keyring_backend": "file",
  "account_aliases": {
    "prod": "prod@example.com",
  },
}`

	writeErr := os.WriteFile(path, []byte(content), 0o600)
	if writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if cfg.KeyringBackend != "file" {
		t.Errorf("KeyringBackend = %q, want %q", cfg.KeyringBackend, "file")
	}

	if cfg.AccountAliases["prod"] != "prod@example.com" {
		t.Errorf("AccountAliases[prod] = %q", cfg.AccountAliases["prod"])
	}
}
