# Repository Guidelines

## Project Structure

- `cmd/nube/`: CLI entrypoint.
- `internal/cmd/`: Kong command structs and handlers.
- `internal/api/`: HTTP client, retry transport, circuit breaker, typed errors.
- `internal/oauth/`: OAuth 2.0 flow (broker + native browser fallback).
- `internal/credstore/`: Credential file storage (`credentials.json`).
- `internal/config/`: App config (`config.json`).
- `internal/outfmt/`: Output mode (JSON/plain) + JSON encoder.
- `internal/errfmt/`: User-friendly error formatting.
- `internal/ui/`: Color + terminal printing.
- `broker/`: OAuth broker (Cloudflare Worker).

## Build & Dev Commands

- `make` / `make build`: build `bin/nube`.
- `make test` / `make lint` / `make fmt` / `make ci`: test, lint, format, full gate.
- `make tools`: install pinned dev tools into `.tools/`.
- `lefthook install`: pre-commit hooks (`.lefthook.yml`).

## Linting

- `bodyclose`: always close HTTP response bodies. Use `//nolint:bodyclose // reason` when closed by callback.
- `rowserrcheck` and `sqlclosecheck` are also enabled.

## Testing

### Helpers (`internal/cmd/testhelpers_test.go`)

- `setupCredStore(t, stores, defaultStore)` — write test credentials to temp dir
- `setupMockAPIClient(t, handler)` — inject mock HTTP handler
- `captureStdout(t)` / `captureStderr(t)` — pipe-based output capture
- `withStdin(t, input, fn)` — pipe-based stdin injection
- `setupConfigDir(t)` — temp `XDG_CONFIG_HOME`

### Conventions

- Table-driven tests with `t.Parallel()` where safe
- `t.TempDir()` for filesystem isolation
- `t.Setenv()` for environment variable isolation
- Package-level vars (`newAPIClient`, `authorizeOAuth`) swapped in tests via `t.Cleanup`
