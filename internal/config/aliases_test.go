package config

import "testing"

func TestNormalizeAccountAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Prod", "prod"},
		{"  staging  ", "staging"},
		{"DEV", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeAccountAlias(tt.input); got != tt.want {
				t.Errorf("NormalizeAccountAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetResolveDeleteAlias(t *testing.T) {
	setupConfigDir(t)

	// Set
	if err := SetAccountAlias("prod", "store@example.com"); err != nil {
		t.Fatalf("SetAccountAlias() error = %v", err)
	}

	// Resolve
	email, ok, err := ResolveAccountAlias("prod")
	if err != nil {
		t.Fatalf("ResolveAccountAlias() error = %v", err)
	}

	if !ok {
		t.Error("expected alias to be found")
	}

	if email != "store@example.com" {
		t.Errorf("email = %q, want %q", email, "store@example.com")
	}

	// List
	aliases, err := ListAccountAliases()
	if err != nil {
		t.Fatalf("ListAccountAliases() error = %v", err)
	}

	if len(aliases) != 1 {
		t.Errorf("len(aliases) = %d, want 1", len(aliases))
	}

	// Delete
	deleted, err := DeleteAccountAlias("prod")
	if err != nil {
		t.Fatalf("DeleteAccountAlias() error = %v", err)
	}

	if !deleted {
		t.Error("expected alias to be deleted")
	}

	// Verify deleted
	_, ok, err = ResolveAccountAlias("prod")
	if err != nil {
		t.Fatalf("ResolveAccountAlias() error = %v", err)
	}

	if ok {
		t.Error("expected alias not found after delete")
	}
}

func TestResolveAlias_NoConfigFile(t *testing.T) {
	setupConfigDir(t)

	_, ok, err := ResolveAccountAlias("prod")
	if err != nil {
		t.Fatalf("ResolveAccountAlias() error = %v", err)
	}

	if ok {
		t.Error("expected false when no config file")
	}
}

func TestResolveAlias_Empty(t *testing.T) {
	setupConfigDir(t)

	_, ok, err := ResolveAccountAlias("")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if ok {
		t.Error("expected false for empty alias")
	}
}

func TestDeleteAlias_NotFound(t *testing.T) {
	setupConfigDir(t)

	deleted, err := DeleteAccountAlias("nonexistent")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if deleted {
		t.Error("expected false for nonexistent alias")
	}
}

func TestListAliases_Empty(t *testing.T) {
	setupConfigDir(t)

	aliases, err := ListAccountAliases()
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if len(aliases) != 0 {
		t.Errorf("expected empty aliases, got %d", len(aliases))
	}
}
