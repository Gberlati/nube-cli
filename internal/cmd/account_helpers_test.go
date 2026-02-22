package cmd

import (
	"testing"

	"github.com/gberlati/nube-cli/internal/secrets"
)

func TestShouldAutoSelectAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"auto", true},
		{"Auto", true},
		{"AUTO", true},
		{"default", true},
		{"Default", true},
		{"prod", false},
		{"user@example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := shouldAutoSelectAccount(tt.input); got != tt.want {
				t.Errorf("shouldAutoSelectAccount(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveAccountAlias(t *testing.T) {
	setupConfigDir(t)

	// Before setting alias
	_, ok, err := resolveAccountAlias("prod")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if ok {
		t.Error("expected alias not found before setup")
	}

	// Emails are not aliases
	_, ok, _ = resolveAccountAlias("user@example.com")
	if ok {
		t.Error("emails should not resolve as aliases")
	}

	// Auto is not an alias
	_, ok, _ = resolveAccountAlias("auto")
	if ok {
		t.Error("auto should not resolve as alias")
	}
}

func TestRequireAccount_Flag(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)

	flags := &RootFlags{Account: "user@example.com"}
	got, err := requireAccount(flags)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if got != "user@example.com" {
		t.Errorf("got %q, want %q", got, "user@example.com")
	}
}

func TestRequireAccount_Env(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t)
	t.Setenv("NUBE_ACCOUNT", "env@example.com")

	flags := &RootFlags{}
	got, err := requireAccount(flags)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if got != "env@example.com" {
		t.Errorf("got %q, want %q", got, "env@example.com")
	}
}

func TestRequireAccount_SingleToken(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t, secrets.Token{
		Client:      "default",
		Email:       "only@example.com",
		AccessToken: "tok",
	})

	flags := &RootFlags{}
	got, err := requireAccount(flags)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if got != "only@example.com" {
		t.Errorf("got %q, want %q", got, "only@example.com")
	}
}

func TestRequireAccount_Error(t *testing.T) {
	setupConfigDir(t)
	setupMockStore(t) // Empty store

	flags := &RootFlags{}
	_, err := requireAccount(flags)
	if err == nil {
		t.Fatal("expected error with no account available")
	}
}
