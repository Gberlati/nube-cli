package cmd

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *ExitError
		want string
	}{
		{"with wrapped error", &ExitError{Code: 1, Err: errors.New("fail")}, "fail"},
		{"nil Err field", &ExitError{Code: 0, Err: nil}, ""},
		{"nil receiver", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitError_Unwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner")
	ee := &ExitError{Code: 2, Err: inner}

	if !errors.Is(ee, inner) {
		t.Error("errors.Is should find inner error")
	}

	wrapped := fmt.Errorf("wrap: %w", ee)
	var target *ExitError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should find ExitError through wrapping")
	}

	if target.Code != 2 {
		t.Errorf("Code = %d, want 2", target.Code)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"exit error code 2", &ExitError{Code: 2, Err: errors.New("e")}, 2},
		{"exit error code 0", &ExitError{Code: 0, Err: errors.New("e")}, 0},
		{"wrapped exit error", fmt.Errorf("wrap: %w", &ExitError{Code: 3, Err: errors.New("e")}), 3},
		{"bare error", errors.New("fail"), 1},
		{"negative code", &ExitError{Code: -1, Err: errors.New("e")}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
