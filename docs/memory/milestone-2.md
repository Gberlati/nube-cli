# Milestone 2: API Client, OAuth & Auth Commands

## What was built

### API client — `internal/api/` (4 files)

#### `errors.go` — Typed API errors
- `APIError` (StatusCode, Code, Message, Body) for non-2xx responses
- `RateLimitError` (Limit, Remaining, Reset, Retries) for 429 after retries
- `NotFoundError` (Resource, ID) for 404
- `AuthError` (Message) for 401
- `Is*Error()` checkers using `errors.As`

#### `transport.go` — Retry transport
- `RetryTransport` wrapping `http.RoundTripper`
- Retries 429 up to 5 times, 5xx up to 2 times
- Tienda Nube rate limit headers: `X-Rate-Limit-Limit`, `X-Rate-Limit-Remaining`, `X-Rate-Limit-Reset`
- Exponential backoff with jitter; respects `Retry-After` header
- `ensureReplayableBody` for request body replay on retries

#### `client.go` — HTTP client
- `Client` struct with baseURL, storeID, accessToken, userAgent
- Auth header: `Authentication: bearer <token>` (Tienda Nube quirk — NOT `Authorization`)
- User-Agent required (API returns 400 if missing)
- Base URL: `https://api.tiendanube.com/v1/{store_id}/`
- Methods: `Get`, `Post`, `Put`, `Delete`
- Generic `DecodeResponse[T any]` for JSON decoding
- `parseErrorResponse` maps HTTP status to typed errors

#### `pagination.go` — Link header pagination
- `PageInfo` struct: Next, Prev, First, Last
- `ParseLinkHeader()` — RFC 5988 parser
- `CollectAllPages[T any]()` — follows next links to collect all items

### OAuth flow — `internal/oauth/` (2 files)

#### `oauth.go` — OAuth 2.0 authorization
- Three flows: browser-based (local HTTP server), manual interactive, remote 2-step
- Auth URL: `https://www.tiendanube.com/apps/{client_id}/authorize`
- Token exchange: POST `https://www.tiendanube.com/apps/authorize/token`
- CSRF state validation on browser flow
- `StepOneComplete` sentinel error for `--remote --step 1`
- Testable seams: `readClientCredentials`, `openBrowserFn` as package vars

#### `browser.go` — Cross-platform browser open
- `xdg-open` (Linux), `open` (macOS), `rundll32` (Windows)
- Uses `exec.CommandContext` with `context.Background()`

### Auth commands — `internal/cmd/auth.go`, `auth_alias.go`

#### `auth.go` — Full auth command tree
- `nube auth credentials <file>` / `nube auth credentials list`
- `nube auth add <email>` with `--manual`, `--remote`, `--step`, `--auth-url`, `--timeout`
- `nube auth list` — table with EMAIL, CLIENT, STORE ID, CREATED
- `nube auth status` — config path, keyring backend, resolved account
- `nube auth remove <email>` — with confirmation prompt
- `nube auth tokens list` / `nube auth tokens delete <email>`
- All commands support `--json` and `--plain` output modes
- Testable seam: `authorizeOAuth` as package var

#### `auth_alias.go` — Alias management
- `nube auth alias list` / `set <alias> <email>` / `unset <alias>`
- Validates: no `@` in alias, rejects reserved names (`auto`, `default`)

### Shared helpers — `internal/cmd/`

#### `output_helpers.go`
- `resultKV` struct + `kv()` constructor
- `tableWriter(ctx)` — returns `tabwriter` or raw stdout for `--plain`
- `writeResult(ctx, u, kvs...)` — JSON/TSV/UI output dispatch

#### `confirm.go`
- `confirmDestructive(flags, action)` — respects `--force` and `--no-input`
- TTY detection for non-interactive environments

#### `account_helpers.go`
- `requireAccount(flags)` — resolves account from: `--account` flag, `NUBE_ACCOUNT` env, keyring default, single stored token
- `resolveAccountAlias()` — alias lookup from config
- Testable seam: `openSecretsStore` as package var

### Config additions — `internal/config/`

#### `aliases.go`
- `NormalizeAccountAlias`, `ResolveAccountAlias`, `SetAccountAlias`, `DeleteAccountAlias`, `ListAccountAliases`
- Reads/writes `account_aliases` map in `config.json`

#### `list_credentials.go`
- `CredentialInfo` struct: Client, Path, Default
- `ListClientCredentials()` — scans config dir for `credentials.json` / `credentials-*.json`

### Modified files

| File | Changes |
|------|---------|
| `internal/cmd/root.go` | Added `DryRun bool` flag (short: `-n`), registered `Auth AuthCmd` |
| `internal/secrets/store.go` | Exported `TokenKey(client, email) string` |
| `internal/errfmt/errfmt.go` | Added formatting for `APIError`, `AuthError`, `RateLimitError`, `NotFoundError` |
| `docs/spec.md` | Auth commands moved to Implemented; added `api/*`, `oauth/*` to code layout |
| `README.md` | Fixed binary name, filled Quick Start, split features, added env vars, filled Security |

## Key bugs encountered & fixed

1. **DryRun duplicate flag**: `aliases:"dry-run"` conflicted with kong's auto-generated `--dry-run` from field name. Binary silently exited with code 1. Fix: removed the alias, kept only `short:"n"`.
2. **err113 lint**: Dynamic errors in oauth.go needed static sentinel errors (`errAuthorization`, `errTokenExchange`, `errNoAccessToken`).
3. **gosec false positives**: `TokenURL` constant flagged as credential, `AccessToken` JSON tag, SSRF on `httpClient.Do()`. Fixed with targeted nolint comments.
4. **noctx lint**: `exec.Command` needed `exec.CommandContext` in browser.go.

## Verification

All passing:
- `make build` — `bin/nube`
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — passes (no test files yet)
- `./bin/nube --help` — shows `auth` command group
- `./bin/nube auth --help` — shows all subcommands
- `./bin/nube auth status` / `--json`
- `./bin/nube version` / `--json`

---

## Next steps

### Priority 1: Complete test suite

The codebase currently has **zero test files**. This is the most critical gap. Every package needs tests before building more features.

#### `internal/api/errors_test.go`
- Error message formatting for each error type
- `Is*Error()` checker functions (positive + negative)
- Error wrapping/unwrapping with `errors.As`

#### `internal/api/transport_test.go`
- 429 retry with rate limit headers (X-Rate-Limit-Reset, Retry-After)
- 429 exhaust retries (MaxRateLimitRetries reached)
- 5xx retry and exhaust
- 2xx/4xx no-retry behavior
- `calculateBackoff()` with reset header, Retry-After, and exponential fallback
- `ensureReplayableBody()` for body replay
- Context cancellation during retry sleep
- Nil body requests

#### `internal/api/client_test.go`
- `httptest.Server` for all HTTP methods (Get, Post, Put, Delete)
- `Authentication` header format (NOT `Authorization`)
- User-Agent header sent
- URL construction with storeID
- `DecodeResponse[T]` generic decoding
- Error response parsing (401 → AuthError, 404 → NotFoundError, other → APIError)
- Options: WithBaseURL, WithUserAgent, WithHTTPClient

#### `internal/api/pagination_test.go`
- `ParseLinkHeader()` with next, prev, first, last
- Multiple rels in single header
- Empty/malformed headers
- `CollectAllPages()` with `httptest.Server` serving multiple pages
- Single page (no Link header)

#### `internal/oauth/oauth_test.go`
- `exchangeCode()` with mock HTTP server
- `extractCodeFromURL()` with full URL, bare code, empty
- `authorizeManual()` — step 1 prints URL, step 2 exchanges code, interactive paste
- `authorizeServer()` — full flow with test HTTP callback
- State mismatch error
- Missing code error
- Timeout behavior
- Use testable seams (`readClientCredentials`, `openBrowserFn`)

#### `internal/cmd/auth_test.go`
- Auth commands via kong parser: credentials set, list, add, list, status, remove, tokens
- Use testable seams (`authorizeOAuth`, `openSecretsStore`)
- JSON + plain + default output modes
- Confirmation prompts (force, no-input)
- Error cases: empty email, invalid step, missing credentials

#### `internal/cmd/auth_alias_test.go`
- Alias set/unset/list via kong parser
- Validation: `@` in alias, reserved names
- JSON output

#### `internal/cmd/output_helpers_test.go`
- `writeResult()` in JSON/TSV/UI modes
- `tableWriter()` with plain vs default
- `kv()` constructor

#### `internal/cmd/confirm_test.go`
- `confirmDestructive()` with --force (skip), --no-input (fail), TTY prompt

#### `internal/cmd/account_helpers_test.go`
- `requireAccount()` resolution chain: flag → env → default → single token
- `resolveAccountAlias()` with alias config
- `shouldAutoSelectAccount()` for "auto"/"default"

#### `internal/config/aliases_test.go`
- Normalize, resolve, set, delete, list aliases
- Roundtrip through config file

#### `internal/config/list_credentials_test.go`
- Scan dir with default + named credential files
- Empty dir, missing dir

#### `internal/errfmt/errfmt_test.go`
- Format each API error type (APIError, AuthError, RateLimitError, NotFoundError)
- Existing error types (ParseError, CredentialsMissingError, KeyNotFound)
- UserFacingError wrapping

#### Existing packages (from milestone 1, still untested)

- `internal/outfmt/outfmt_test.go` — Mode validation, FromFlags, FromEnv, context roundtrip, WriteJSON
- `internal/config/config_test.go` — ReadConfig, WriteConfig, JSON5 parsing
- `internal/config/credentials_test.go` — Read/write credentials, validation
- `internal/config/keys_test.go` — ParseKey, GetValue, SetValue, UnsetValue
- `internal/config/paths_test.go` — Dir, ConfigPath, CredentialsPath
- `internal/ui/ui_test.go` — New with color modes, Printer output, context roundtrip
- `internal/cmd/root_test.go` — Execute with valid/invalid args, flag parsing, exit codes
- `internal/cmd/version_test.go` — VersionString, VersionCmd JSON/text output
- `internal/cmd/config_cmd_test.go` — Config subcommands (path, keys, list, get, set, unset)
- `internal/cmd/enabled_commands_test.go` — Allowlist enforcement
- `internal/cmd/exit_test.go` — ExitError, ExitCode

#### Integration tests (separate suite)

- `internal/integration/` — guarded by `//go:build integration`
- Requires stored credentials + token in keyring
- Tests: `auth status`, `auth list`, `store get` (once implemented)
- Run: `NUBE_IT_ACCOUNT=you@email.com go test -tags=integration ./internal/integration`

### Priority 2: First resource commands

Once the test suite is solid, implement the first API resource commands:

1. `nube store get` — simplest API call, validates the full auth → API pipeline
2. `nube product list` / `nube product get <id>` — first CRUD resource with pagination
3. `nube order list` / `nube order get <id>` — high-value resource

### Priority 3: Polish

- Shell completion (kong supports `--completion-script-bash/zsh`)
- `--all` flag for auto-pagination on list commands
- Tab completion for `--account` values
