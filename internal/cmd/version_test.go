package cmd

import (
	"context"
	"testing"

	"github.com/gberlati/nube-cli/internal/outfmt"
)

func TestVersionString(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		commit     string
		date       string
		wantPrefix string
	}{
		{"dev only", "dev", "", "", "dev"},
		{"version only", "v1.0.0", "", "", "v1.0.0"},
		{"version + commit", "v1.0.0", "abc123", "", "v1.0.0 (abc123)"},
		{"version + date", "v1.0.0", "", "2024-01-01", "v1.0.0 (2024-01-01)"},
		{"all set", "v1.0.0", "abc123", "2024-01-01", "v1.0.0 (abc123 2024-01-01)"},
		{"empty version defaults to dev", "", "", "", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origVersion, origCommit, origDate := version, commit, date
			version, commit, date = tt.version, tt.commit, tt.date
			defer func() { version, commit, date = origVersion, origCommit, origDate }()

			got := VersionString()
			if got != tt.wantPrefix {
				t.Errorf("VersionString() = %q, want %q", got, tt.wantPrefix)
			}
		})
	}
}

func TestVersionCmd_Run_JSON(t *testing.T) {
	sc := captureStdout(t)

	origVersion, origCommit, origDate := version, commit, date
	version, commit, date = "v1.2.3", "abc", "2024-01-01"
	defer func() { version, commit, date = origVersion, origCommit, origDate }()

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	cmd := &VersionCmd{}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := sc.String()
	if output == "" {
		t.Fatal("expected JSON output")
	}
}

func TestVersionCmd_Run_Text(t *testing.T) {
	sc := captureStdout(t)

	origVersion := version
	version = "v1.0.0"
	defer func() { version = origVersion }()

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})
	cmd := &VersionCmd{}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := sc.String()
	if output == "" {
		t.Fatal("expected text output")
	}
}
