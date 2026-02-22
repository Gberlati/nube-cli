package config

import (
	"errors"
	"testing"
)

func TestParseKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Key
		wantErr bool
	}{
		{"valid keyring_backend", "keyring_backend", KeyKeyringBackend, false},
		{"unknown key", "nonexistent", "", true},
		{"empty key", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("ParseKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestKeySpecFor(t *testing.T) {
	t.Parallel()

	spec, err := KeySpecFor(KeyKeyringBackend)
	if err != nil {
		t.Fatalf("KeySpecFor() error = %v", err)
	}

	if spec.Key != KeyKeyringBackend {
		t.Errorf("Key = %q, want %q", spec.Key, KeyKeyringBackend)
	}

	_, err = KeySpecFor("bogus")
	if err == nil {
		t.Error("expected error for bogus key")
	}
}

func TestGetSetUnsetValue(t *testing.T) {
	t.Parallel()

	var cfg File

	// Set
	if err := SetValue(&cfg, KeyKeyringBackend, "file"); err != nil {
		t.Fatalf("SetValue() error = %v", err)
	}

	// Get
	if got := GetValue(cfg, KeyKeyringBackend); got != "file" {
		t.Errorf("GetValue() = %q, want %q", got, "file")
	}

	// Unset
	if err := UnsetValue(&cfg, KeyKeyringBackend); err != nil {
		t.Fatalf("UnsetValue() error = %v", err)
	}

	if got := GetValue(cfg, KeyKeyringBackend); got != "" {
		t.Errorf("GetValue() after unset = %q, want empty", got)
	}
}

func TestSetValueInvalidKey(t *testing.T) {
	t.Parallel()

	var cfg File

	err := SetValue(&cfg, "invalid_key", "value")
	if err == nil {
		t.Error("expected error for invalid key")
	}

	if !errors.Is(err, errUnknownConfigKey) {
		t.Errorf("expected errUnknownConfigKey, got %v", err)
	}
}

func TestUnsetValueInvalidKey(t *testing.T) {
	t.Parallel()

	var cfg File

	err := UnsetValue(&cfg, "invalid_key")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestKeyList(t *testing.T) {
	t.Parallel()

	keys := KeyList()
	if len(keys) == 0 {
		t.Fatal("KeyList() returned empty")
	}

	if keys[0] != KeyKeyringBackend {
		t.Errorf("keys[0] = %q, want %q", keys[0], KeyKeyringBackend)
	}
}

func TestKeyNames(t *testing.T) {
	t.Parallel()

	names := KeyNames()
	if len(names) == 0 {
		t.Fatal("KeyNames() returned empty")
	}

	if names[0] != "keyring_backend" {
		t.Errorf("names[0] = %q, want %q", names[0], "keyring_backend")
	}
}

func TestGetValueUnknown(t *testing.T) {
	t.Parallel()

	got := GetValue(File{}, "bogus")
	if got != "" {
		t.Errorf("GetValue(unknown) = %q, want empty", got)
	}
}
