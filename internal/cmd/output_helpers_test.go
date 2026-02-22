package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

func TestKV(t *testing.T) {
	t.Parallel()

	r := kv("name", "value")
	if r.Key != "name" || r.Value != "value" {
		t.Errorf("kv() = %+v", r)
	}
}

func TestWriteResult_JSON(t *testing.T) {
	sc := captureStdout(t)

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})

	u, _ := ui.New(ui.Options{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Color:  "never",
	})

	if err := writeResult(ctx, u, kv("email", "test@example.com"), kv("stored", true)); err != nil {
		t.Fatalf("error = %v", err)
	}

	output := sc.Bytes()
	var got map[string]any
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, string(output))
	}

	if got["email"] != "test@example.com" {
		t.Errorf("email = %v", got["email"])
	}
}

func TestWriteResult_Default(t *testing.T) {
	var stdout bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{})

	u, _ := ui.New(ui.Options{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Color:  "never",
	})

	if err := writeResult(ctx, u, kv("email", "test@example.com")); err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(stdout.String(), "email") {
		t.Errorf("output = %q, want containing 'email'", stdout.String())
	}
}
