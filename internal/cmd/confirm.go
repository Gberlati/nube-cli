package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func confirmDestructive(flags *RootFlags, action string) error {
	if flags == nil || flags.Force {
		return nil
	}

	// Never prompt in non-interactive contexts.
	if flags.NoInput || !term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // fd conversion is safe
		return usagef("refusing to %s without --force (non-interactive)", action)
	}

	fmt.Fprintf(os.Stderr, "Proceed to %s? [y/N]: ", action)
	line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')

	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return fmt.Errorf("read confirmation: %w", readErr)
	}

	ans := strings.TrimSpace(strings.ToLower(line))
	if ans == "y" || ans == "yes" {
		return nil
	}

	return &ExitError{Code: 1, Err: errors.New("cancelled")}
}
