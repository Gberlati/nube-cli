package cmd

import (
	"testing"
)

func TestExecute_Help(t *testing.T) {
	_ = captureStdout(t)

	// --help causes kong to call os.Exit(0) which is caught by the panic handler
	err := Execute([]string{"--help"})
	if err != nil {
		t.Fatalf("--help should not error, got: %v", err)
	}
}

func TestExecute_Version(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"version"})
	if err != nil {
		t.Fatalf("version error = %v", err)
	}

	if buf.String() == "" {
		t.Error("expected version output")
	}
}

func TestExecute_InvalidCommand(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for invalid command")
	}
}

func TestExecute_UnknownFlag(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestExecute_JSONPlainConflict(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"version", "--json", "--plain"})
	if err == nil {
		t.Fatal("expected error for --json --plain conflict")
	}
}
