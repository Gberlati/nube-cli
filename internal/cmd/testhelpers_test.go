package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/gberlati/nube-cli/internal/credstore"
)

// setupCredStore writes a credentials.json file for tests.
func setupCredStore(t *testing.T, stores map[string]credstore.StoreProfile, defaultStore string) {
	t.Helper()
	setupConfigDir(t) // XDG_CONFIG_HOME → temp dir
	f := credstore.File{DefaultStore: defaultStore, Stores: stores}
	if err := credstore.Write(f); err != nil {
		t.Fatalf("write test credentials: %v", err)
	}
}

// setupConfigDir sets XDG_CONFIG_HOME to a temp dir.
func setupConfigDir(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

// stdoutCapture holds the captured stdout buffer and a flush function.
type stdoutCapture struct {
	buf  bytes.Buffer
	w    *os.File
	done chan struct{}
}

// String closes the write end and waits for the reader goroutine to finish,
// then returns the captured output.
func (c *stdoutCapture) String() string {
	_ = c.w.Close()
	<-c.done
	return c.buf.String()
}

func (c *stdoutCapture) Bytes() []byte {
	_ = c.w.Close()
	<-c.done
	return c.buf.Bytes()
}

// captureStdout redirects os.Stdout to a buffer.
// Call .String() or .Bytes() on the returned value to flush and get output.
func captureStdout(t *testing.T) *stdoutCapture {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = origStdout
		_ = w.Close()
	})

	sc := &stdoutCapture{w: w, done: make(chan struct{})}
	go func() {
		defer close(sc.done)
		_, _ = sc.buf.ReadFrom(r)
	}()

	return sc
}

// stderrCapture holds the captured stderr buffer.
type stderrCapture struct { //nolint:unused // test infrastructure for future commands
	buf  bytes.Buffer
	w    *os.File
	done chan struct{}
}

func (c *stderrCapture) String() string { //nolint:unused // test infrastructure for future commands
	_ = c.w.Close()
	<-c.done

	return c.buf.String()
}

// captureStderr redirects os.Stderr to a buffer.
// Call .String() on the returned value to flush and get output.
func captureStderr(t *testing.T) *stderrCapture { //nolint:unused // test infrastructure for future commands
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	origStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = origStderr
		_ = w.Close()
	})

	sc := &stderrCapture{w: w, done: make(chan struct{})}
	go func() {
		defer close(sc.done)
		_, _ = sc.buf.ReadFrom(r)
	}()

	return sc
}

// withStdin temporarily replaces os.Stdin with a pipe containing the given input.
func withStdin(t *testing.T, input string, fn func()) { //nolint:unused // test infrastructure for future commands
	t.Helper()

	orig := os.Stdin

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdin = r

	_, _ = io.WriteString(w, input)
	_ = w.Close()

	fn()

	_ = r.Close()
	os.Stdin = orig
}

// runKong creates an isolated Kong parser for a command and runs it.
// Useful for testing individual commands without the full Execute() machinery.
func runKong(t *testing.T, cmd any, args []string, ctx context.Context, flags *RootFlags) (err error) { //nolint:unused // test infrastructure for future commands
	t.Helper()

	parser, err := kong.New(
		cmd,
		kong.Writers(io.Discard, io.Discard),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
	)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				if ep.code == 0 {
					err = nil

					return
				}

				err = &ExitErr{Code: ep.code, Err: errors.New("exited")}

				return
			}

			panic(r)
		}
	}()

	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}

	if ctx != nil {
		kctx.BindTo(ctx, (*context.Context)(nil))
	}

	if flags == nil {
		flags = &RootFlags{}
	}

	kctx.Bind(flags)

	return kctx.Run()
}
