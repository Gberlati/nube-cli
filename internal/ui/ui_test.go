package ui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

func TestNew_InvalidColor(t *testing.T) {
	t.Parallel()

	_, err := New(Options{Color: "neon"})
	if err == nil {
		t.Fatal("expected error for invalid color")
	}

	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Errorf("expected *ParseError, got %T", err)
	}
}

func TestNew_ValidModes(t *testing.T) {
	t.Parallel()

	modes := []string{"auto", "always", "never", ""}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			u, err := New(Options{
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
				Color:  mode,
			})
			if err != nil {
				t.Fatalf("New(color=%q) error = %v", mode, err)
			}

			if u == nil {
				t.Fatal("expected non-nil UI")
			}
		})
	}
}

func TestPrinter_Output(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	u, err := New(Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Color:  "never",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	u.Out().Successf("ok %s", "done")

	if !strings.Contains(stdout.String(), "ok done") {
		t.Errorf("Successf: stdout = %q", stdout.String())
	}

	stdout.Reset()
	u.Out().Println("hello")

	if !strings.Contains(stdout.String(), "hello\n") {
		t.Errorf("Println: stdout = %q", stdout.String())
	}

	stdout.Reset()
	u.Out().Printf("count=%d", 42)

	if !strings.Contains(stdout.String(), "count=42") {
		t.Errorf("Printf: stdout = %q", stdout.String())
	}

	stdout.Reset()
	u.Out().Print("raw")

	if stdout.String() != "raw" {
		t.Errorf("Print: stdout = %q, want %q", stdout.String(), "raw")
	}

	u.Err().Error("bad")

	if !strings.Contains(stderr.String(), "bad") {
		t.Errorf("Error: stderr = %q", stderr.String())
	}

	stderr.Reset()
	u.Err().Errorf("fail %d", 1)

	if !strings.Contains(stderr.String(), "fail 1") {
		t.Errorf("Errorf: stderr = %q", stderr.String())
	}
}

func TestColorEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile termenv.Profile
		want    bool
	}{
		{"TrueColor", termenv.TrueColor, true},
		{"ANSI256", termenv.ANSI256, true},
		{"ANSI", termenv.ANSI, true},
		{"Ascii", termenv.Ascii, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &Printer{
				o:       termenv.NewOutput(&bytes.Buffer{}),
				profile: tt.profile,
			}

			if got := p.ColorEnabled(); got != tt.want {
				t.Errorf("ColorEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChooseProfile(t *testing.T) {
	// Cannot run in parallel due to NO_COLOR env var
	t.Run("never", func(t *testing.T) {
		got := chooseProfile(termenv.TrueColor, "never")
		if got != termenv.Ascii {
			t.Errorf("never mode: got %v, want Ascii", got)
		}
	})

	t.Run("always", func(t *testing.T) {
		got := chooseProfile(termenv.Ascii, "always")
		if got != termenv.TrueColor {
			t.Errorf("always mode: got %v, want TrueColor", got)
		}
	})

	t.Run("auto uses detected", func(t *testing.T) {
		got := chooseProfile(termenv.ANSI256, "auto")
		if got != termenv.ANSI256 {
			t.Errorf("auto mode: got %v, want ANSI256", got)
		}
	})

	t.Run("NO_COLOR override", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")

		got := chooseProfile(termenv.TrueColor, "always")
		if got != termenv.Ascii {
			t.Errorf("NO_COLOR: got %v, want Ascii", got)
		}
	})
}

func TestContextRoundtrip(t *testing.T) {
	t.Parallel()

	u, err := New(Options{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Color:  "never",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := WithUI(context.Background(), u)
	got := FromContext(ctx)

	if got != u {
		t.Error("expected same UI from context roundtrip")
	}
}

func TestFromContext_Empty(t *testing.T) {
	t.Parallel()

	got := FromContext(context.Background())
	if got != nil {
		t.Error("expected nil from empty context")
	}
}
