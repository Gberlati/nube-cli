# Repository Guidelines

## Project Structure

- `cmd/nube/`: CLI entrypoint.
- `internal/`: implementation (`cmd/`, Tienda Nube OAuth, config/secrets, output/UI).
- Tests: `*_test.go` next to code; opt-in integration suite in `internal/integration/` (build-tagged).
- `bin/`: build outputs; `docs/`: specs/releasing.

## Build, Test, and Development Commands

- `make` / `make build`: build `bin/nube`.
- `make tools`: install pinned dev tools into `.tools/`.
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate.
- Optional: `pnpm nube …`: build + run in one step.
- Hooks: `lefthook install` enables pre-commit/pre-push checks (`.lefthook.yml`).

## Linting

- `bodyclose` linter is enforced — always close HTTP response bodies. When the body is closed by a callback (e.g. `DecodeResponse`), use `//nolint:bodyclose // reason` with a comment explaining why.
- `rowserrcheck` and `sqlclosecheck` are also enabled.

## Testing

### Test helpers (`internal/cmd/testhelpers_test.go`)

- `captureStdout(t)` / `captureStderr(t)` — pipe-based stdout/stderr capture
- `withStdin(t, input, fn)` — pipe-based stdin injection for interactive testing
- `runKong(t, cmd, args, ctx, flags)` — isolated Kong parser for command testing
- `setupMockStore(t, tokens...)` — in-memory keyring mock
- `setupConfigDir(t)` — temp XDG_CONFIG_HOME

### HTTP test helpers (`internal/api/testhelpers_test.go`)

- `newTestClient(t, handler)` — creates a test API client with httptest server
- `withRequestTracking(t)` — middleware + getter for concurrent request tracking
- `chainMiddleware(handler, ...middleware)` — compose HTTP handlers

### Conventions

- Table-driven tests with `t.Parallel()` where safe
- `t.TempDir()` for filesystem isolation
- `t.Setenv()` for environment variable isolation

