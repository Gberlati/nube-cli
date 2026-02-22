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
- Hooks: `lefthook install` enables pre-commit/pre-push checks.

