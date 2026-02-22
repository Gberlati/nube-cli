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

	if len(cfg.ClientDomains) != 0 {
		t.Errorf("ClientDomains should be empty, got %v", cfg.ClientDomains)
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	setupConfigDir(t)

	want := File{
		ClientDomains: map[string]string{"myapp": "myapp.example.com"},
	}

	if writeErr := WriteConfig(want); writeErr != nil {
		t.Fatalf("WriteConfig() error = %v", writeErr)
	}

	got, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if got.ClientDomains["myapp"] != "myapp.example.com" {
		t.Errorf("ClientDomains[myapp] = %q, want %q", got.ClientDomains["myapp"], "myapp.example.com")
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
  "client_domains": {
    "myapp": "myapp.example.com",
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

	if cfg.ClientDomains["myapp"] != "myapp.example.com" {
		t.Errorf("ClientDomains[myapp] = %q", cfg.ClientDomains["myapp"])
	}
}
