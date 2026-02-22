package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	setupConfigDir(t)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}

	if !strings.HasSuffix(dir, AppName) {
		t.Errorf("Dir() = %q, want suffix %q", dir, AppName)
	}
}

func TestConfigPath(t *testing.T) {
	setupConfigDir(t)

	p, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	if !strings.HasSuffix(p, "config.json") {
		t.Errorf("ConfigPath() = %q, want suffix config.json", p)
	}
}

func TestClientCredentialsPath(t *testing.T) {
	setupConfigDir(t)

	p, err := ClientCredentialsPath()
	if err != nil {
		t.Fatalf("ClientCredentialsPath() error = %v", err)
	}

	if !strings.HasSuffix(p, "credentials.json") {
		t.Errorf("ClientCredentialsPath() = %q, want suffix credentials.json", p)
	}
}

func TestClientCredentialsPathFor(t *testing.T) {
	setupConfigDir(t)

	tests := []struct {
		name   string
		client string
		suffix string
	}{
		{"default", "default", "credentials.json"},
		{"empty defaults", "", "credentials.json"},
		{"named client", "myapp", "credentials-myapp.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ClientCredentialsPathFor(tt.client)
			if err != nil {
				t.Fatalf("ClientCredentialsPathFor(%q) error = %v", tt.client, err)
			}

			if !strings.HasSuffix(p, tt.suffix) {
				t.Errorf("path = %q, want suffix %q", p, tt.suffix)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	setupConfigDir(t)

	dir, err := EnsureDir()
	if err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if !info.IsDir() {
		t.Error("EnsureDir() did not create a directory")
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	home, _ := os.UserHomeDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty", "", ""},
		{"tilde alone", "~", home},
		{"tilde prefix", "~/foo", filepath.Join(home, "foo")},
		{"absolute", "/usr/bin", "/usr/bin"},
		{"relative", "foo/bar", "foo/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExpandPath(tt.path)
			if err != nil {
				t.Fatalf("ExpandPath(%q) error = %v", tt.path, err)
			}

			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
