# Milestone 2 Follow-up: Test Suite & README OAuth Docs

## What was built

27 test files covering every package with tests, plus README updates for OAuth documentation.

### Test files created (by batch)

#### Batch 1: Pure Value Types

| File | Coverage | Tests |
|------|----------|-------|
| `internal/api/errors_test.go` | — | Error() formatting for APIError, RateLimitError, NotFoundError, AuthError; Is* helpers (direct, wrapped, negative, nil) |
| `internal/outfmt/outfmt_test.go` | 92.0% | FromFlags (valid + conflict), FromEnv, context roundtrip, WriteJSON, payload helpers |
| `internal/errfmt/errfmt_test.go` | 69.8% | Format() for all error types including nil, wrapped, UserFacingError |
| `internal/cmd/exit_test.go` | — | ExitError.Error/Unwrap, ExitCode (nil, wrapped, bare, negative) |

#### Batch 2: Config Package

| File | Tests |
|------|-------|
| `internal/config/testhelpers_test.go` | Shared `setupConfigDir(t)` helper (sets XDG_CONFIG_HOME to t.TempDir()) |
| `internal/config/clients_test.go` | NormalizeClientName, NormalizeClientNameOrDefault |
| `internal/config/keys_test.go` | ParseKey, KeySpecFor, GetValue/SetValue/UnsetValue, KeyList, KeyNames |
| `internal/config/paths_test.go` | Dir, ConfigPath, ClientCredentialsPath/For, EnsureDir, ExpandPath |
| `internal/config/config_test.go` | ReadConfig (no file), Write/Read roundtrip, ConfigExists, JSON5 parsing (comments, trailing commas) |
| `internal/config/credentials_test.go` | Write/Read roundtrip, missing -> CredentialsMissingError, invalid JSON, missing fields |
| `internal/config/aliases_test.go` | NormalizeAccountAlias, Set/Resolve/Delete/List roundtrip, no config file, empty alias |
| `internal/config/list_credentials_test.go` | Empty dir, default only, multiple credentials (sorted), no dir -> nil |

Config package overall coverage: **78.9%**

#### Batch 3: UI Package

| File | Coverage | Tests |
|------|----------|-------|
| `internal/ui/ui_test.go` | 93.2% | New (invalid/valid colors), Printer output methods, ColorEnabled, chooseProfile (incl. NO_COLOR), context roundtrip |

#### Batch 4: API HTTP Tests

| File | Tests |
|------|-------|
| `internal/api/transport_test.go` | ensureReplayableBody, calculateBackoff, RoundTrip (200/400/429/5xx, retries, exhaust, context cancel, body replay, nil body GET); uses atomic counters, BaseDelay: time.Millisecond |
| `internal/api/client_test.go` | Get/Post/Put/Delete, Authentication header regression guard, User-Agent, URL construction, DecodeResponse[T], error responses (401->AuthError, 404->NotFoundError, 500->APIError) |
| `internal/api/pagination_test.go` | ParseLinkHeader (single/multiple/all four/empty/malformed/whitespace), HasNext, CollectAllPages (3 pages, single page, error on page 2) |

API package overall coverage: **88.0%**

#### Batch 5: OAuth Package

| File | Coverage | Tests |
|------|----------|-------|
| `internal/oauth/oauth_test.go` | 68.1% | extractCodeFromURL, exchangeCode (success/bad status/empty token), authorizeManual (remote step 1/2, missing auth-url), authorizeServer (full flow with mock browser, state mismatch, missing code, timeout), Authorize (delegates, credentials error, default timeout) |

Key test patterns:
- `mockTokenServer` with custom transport to redirect TokenURL to httptest server
- `mockBrowser` to capture URL and simulate callback
- `doCallbackRequest` helper with `http.NewRequestWithContext` (noctx compliant)
- authorizeServer tests cannot run in parallel (fixed port 8910)

#### Batch 6: Command Package

| File | Tests |
|------|-------|
| `internal/cmd/testhelpers_test.go` | `mockStore` (in-memory secrets.Store), `stdoutCapture` (pipe-based stdout capture with sync), `setupMockStore`, `setupConfigDir`, `captureStdout` |
| `internal/cmd/confirm_test.go` | Force=true skips, NoInput=true errors, nil flags skips |
| `internal/cmd/output_helpers_test.go` | kv() constructor, writeResult in JSON/default modes |
| `internal/cmd/enabled_commands_test.go` | enforceEnabledCommands with real kong parser (allowed, blocked, wildcard *, all, case insensitive) |
| `internal/cmd/version_test.go` | VersionString with various combos, VersionCmd.Run in JSON and text modes |
| `internal/cmd/account_helpers_test.go` | shouldAutoSelectAccount, resolveAccountAlias, requireAccount (flag, env, single token, error) |
| `internal/cmd/config_cmd_test.go` | Config subcommands via Execute(): path, path --json, keys, set/get roundtrip, unset, list, invalid key |
| `internal/cmd/auth_alias_test.go` | set/list roundtrip, unset, not found, @ validation, reserved names, JSON output |
| `internal/cmd/auth_test.go` | auth add (success, JSON, empty email, invalid step, step without remote, StepOneComplete, OAuth error), auth list, auth remove (--force), auth tokens list, auth status |
| `internal/cmd/root_test.go` | Execute with --help, version, invalid command, unknown flag, --json --plain conflict |

Cmd package overall coverage: **64.4%**

#### Batch 7: README.md Update

- Moved **Auth** from Planned to Implemented features
- Added **Authentication** section with:
  - Store OAuth credentials (`nube auth credentials set`)
  - Authorize a store (browser, manual, remote 2-step flows)
  - Manage accounts (list, status, remove, tokens)
  - Account aliases (set, list, unset)
  - Multiple stores with `--client`

### Coverage summary

| Package | Coverage |
|---------|----------|
| `internal/ui` | 93.2% |
| `internal/outfmt` | 92.0% |
| `internal/api` | 88.0% |
| `internal/config` | 78.9% |
| `internal/errfmt` | 69.8% |
| `internal/oauth` | 68.1% |
| `internal/cmd` | 64.4% |

## Key patterns used

- **Table-driven tests** with `t.Run` subtests
- **`t.Parallel()`** except for tests that capture stdout/stderr or mutate package vars
- **`t.TempDir()` + `t.Setenv("XDG_CONFIG_HOME", ...)`** for config isolation
- **Package-var seams**: save original -> set mock -> `t.Cleanup(restore)`
- **`httptest.NewServer`** with atomic counters for HTTP tests
- **`package foo_test`** (black-box) for exported-only testing; **`package foo`** (white-box) for unexported helpers
- **`stdoutCapture`** type with `.String()`/`.Bytes()` that close pipe and wait for goroutine before returning

## Bugs encountered & fixed

1. **Stdout capture race**: `captureStdout` returned `*bytes.Buffer` that goroutine hadn't finished reading when tests checked it. Fixed with `stdoutCapture` type that syncs on read.

2. **42 lint issues across all test files** — systematic fixes:
   - **err113**: Package-level sentinel errors instead of inline `errors.New` in test tables
   - **errorlint**: `errors.As` instead of type assertions
   - **govet shadow**: Renamed shadowed `err` variables
   - **noctx**: `http.NewRequestWithContext` instead of `http.Get`
   - **predeclared**: Renamed `cap` variable (shadows builtin)
   - **unparam**: Removed unused return values from helpers
   - **wsl_v5**: Blank lines between `}` and `if` statements (30+ instances)
   - **gofumpt**: Alignment in var blocks
   - **nolintlint**: Removed stale `//nolint` directives

## Verification

- `make ci` — passes clean (fmt + lint + test)
- `make lint` — 0 issues
- `make fmt` — clean
- `go test -cover ./...` — all packages covered

## Next steps

Priorities from milestone-2.md still apply:
1. **Integration tests** — guarded by `//go:build integration`, require real credentials
2. **First resource commands** — `nube store get`, `nube product list/get`, `nube order list/get`
3. **Polish** — shell completion, `--all` auto-pagination, tab completion for `--account`
