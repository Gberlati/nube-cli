# Bootstrap: Foundation Complete

## What was done

### Go module & dependencies

- Created `go.mod` (`github.com/gberlati/nube-cli`, Go 1.26)
- Dependencies: kong, termenv, keyring, json5, x/term

### Internal packages

#### `internal/ui/ui.go` — Terminal UI
- `UI` struct with `Out()` / `Err()` printers (stdout/stderr)
- `Printer` with `Printf`, `Println`, `Successf`, `Error`, `Errorf`, `Print`
- Color management via termenv (`auto|always|never`, respects `NO_COLOR`)
- Context helpers: `WithUI(ctx)` / `FromContext(ctx)`

#### `internal/outfmt/outfmt.go` — Output formatting
- `Mode{JSON, Plain}` with mutual-exclusion validation
- `FromFlags()`, `FromEnv()` (reads `NUBE_JSON` / `NUBE_PLAIN`)
- Context: `WithMode`, `FromContext`, `IsJSON`, `IsPlain`
- `WriteJSON(ctx, writer, value)` with indented encoding
- Payload helpers: `KeyValuePayload`, `KeysPayload`, `PathPayload`

#### `internal/errfmt/errfmt.go` — Error formatting
- `Format(err)` — user-friendly error messages
- `UserFacingError` type with `Message` + `Cause`
- Kong parse error formatting with hints
- `CredentialsMissingError` handling with Tienda Nube guidance
- Keyring `ErrKeyNotFound` handling

#### `internal/config/` — Configuration (5 files)
- **`paths.go`**: `AppName = "nube-cli"`, `Dir()`, `EnsureDir()`, `KeyringDir()`, `EnsureKeyringDir()`, `ConfigPath()`, `ClientCredentialsPath()`, `ClientCredentialsPathFor(client)`, `ExpandPath()`
- **`clients.go`**: `DefaultClientName = "default"`, `NormalizeClientName()`, `NormalizeClientNameOrDefault()`
- **`config.go`**: `File` struct (`KeyringBackend`, `AccountAliases`, `AccountClients`, `ClientDomains`), `ReadConfig()` (JSON5), `WriteConfig()`, `ConfigExists()`
- **`credentials.go`**: `ClientCredentials{ClientID, ClientSecret}` — flat JSON (not Google's nested format), `ReadClientCredentials()`, `WriteClientCredentials()`, `CredentialsMissingError`
- **`keys.go`**: `Key` type, `KeySpec` with Get/Set/Unset/EmptyHint, `ParseKey()`, `GetValue()`, `SetValue()`, `UnsetValue()`, `KeyNames()`, `KeyList()`. Currently has one key: `keyring_backend`

#### `internal/secrets/store.go` — Keyring store
- `Store` interface: `Keys()`, `SetToken()`, `GetToken()`, `DeleteToken()`, `ListTokens()`, `GetDefaultAccount()`, `SetDefaultAccount()`
- `Token` struct: `Client`, `Email`, `UserID` (store ID), `Scopes []string`, `CreatedAt`, `AccessToken` (tagged `json:"-"`)
- Key format: `token:<client>:<email>` with legacy `token:<email>` migration
- `KeyringStore` backed by `99designs/keyring`
- `OpenDefault()` factory
- Linux D-Bus timeout protection (5s), file backend fallback
- Env vars: `NUBE_KEYRING_PASSWORD`, `NUBE_KEYRING_BACKEND`

#### `internal/cmd/` — Command layer (5 files)
- **`exit.go`**: `ExitError{Code, Err}` + `ExitCode(err)` helper
- **`version.go`**: Linker-injected `version`, `commit`, `date`. `VersionString()`. `VersionCmd.Run()` with JSON/text output
- **`enabled_commands.go`**: `enforceEnabledCommands()` — allowlist enforcement for agent sandboxing
- **`config_cmd.go`**: `ConfigCmd` with subcommands: `path`, `keys`, `list`, `get <key>`, `set <key> <value>`, `unset <key>` — all support `--json`
- **`root.go`**: `Execute(args)` — parse → enforce enabled commands → slog → output mode → UI → bind context → run. `RootFlags` with `--color`, `--account`, `--client`, `--enable-commands`, `--json`, `--plain`, `--force`, `--no-input`, `--verbose`. `CLI` struct embedding `RootFlags` + `Version`, `Config`, `VersionCmd` commands. Env-var defaults: `NUBE_COLOR`, `NUBE_CLIENT`, `NUBE_ENABLE_COMMANDS`, `NUBE_JSON`, `NUBE_PLAIN`

### Entrypoint

- `cmd/nube/main.go` — calls `cmd.Execute(os.Args[1:])`, exits with `cmd.ExitCode(err)`

### Tooling & CI

- `.golangci.yml` — 35+ linters (revive, staticcheck, gosec, cyclop, etc.), exclusions for test/cmd files, gofumpt + goimports formatters
- `.github/workflows/ci.yml` — setup-go (go-version-file), `make tools`, `make fmt-check`, `go test ./...`, `make lint`

### Verification

All passing:
- `make build` → `bin/nube`
- `./bin/nube --help` — global flags + `config`, `version` commands
- `./bin/nube version` / `./bin/nube version --json`
- `./bin/nube config path` / `config keys` / `config list`
- `NUBE_ENABLE_COMMANDS=version ./bin/nube config path` → exit 2
- `make lint` → 0 issues
- `make test` → passes

## Key Tienda Nube differences from gogcli

| Aspect | gogcli (Google) | nube-cli (Tienda Nube) |
|--------|----------------|----------------------|
| Tokens | Refresh tokens (OAuth2 flow) | Permanent access tokens |
| Token struct field | `RefreshToken` | `AccessToken` |
| Stored payload field | `refresh_token` | `access_token` |
| Extra token metadata | `Services []string` | `UserID string` (store ID) |
| Credentials file format | Nested (`installed.client_id`) | Flat (`client_id`) |
| Auth header | `Authorization: Bearer` | `Authentication: bearer` |
| Env var prefix | `GOG_` | `NUBE_` |
| API client deps | `golang.org/x/oauth2`, `google.golang.org/api` | None yet (permanent tokens) |
| Config keys | `timezone`, `keyring_backend` | `keyring_backend` (timezone not needed) |

## Next steps

### High priority (auth & API client)

1. **`internal/api/client.go`** — HTTP client for Tienda Nube API
   - Base URL: `https://api.tiendanube.com/2025-03/{store_id}`
   - Auth header: `Authentication: bearer <access_token>` (note: NOT `Authorization`)
   - User-Agent: `nube-cli (email)` — 400 if missing
   - Rate limiting: leaky bucket, 2 req/s, 40-request bucket
   - Rate limit headers: `x-rate-limit-limit`, `x-rate-limit-remaining`, `x-rate-limit-reset`
   - Pagination: `page` + `per_page` params; `Link` headers with rel=next/prev/first/last

2. **`internal/cmd/auth.go`** — Auth commands
   - `nube auth credentials <file>` — install OAuth client credentials
   - `nube auth credentials list` — list installed credentials
   - `nube auth add <email>` — OAuth flow to get access token
   - `nube auth status` — show current auth state
   - `nube auth remove <email>` — remove stored token
   - `nube auth tokens list` / `nube auth tokens delete <email>`

3. **OAuth flow implementation** — Token exchange with Tienda Nube
   - Local HTTP redirect on ephemeral port
   - Browserless/manual flow for headless
   - Remote 2-step flow

### Medium priority (first resource commands)

4. **`nube store get`** — simplest API command, good end-to-end test
5. **`nube product list`** / **`nube product get <id>`** — first CRUD resource
6. **`nube order list`** / **`nube order get <id>`** — high-value resource

### Lower priority (polish)

7. **Tests** — unit tests for config, secrets, errfmt, outfmt packages
8. **`nube auth alias`** — account alias management
9. **Tab completion** — shell completion via kong
10. **Pagination helpers** — `Link` header parsing, `--all` flag for auto-pagination
11. **Table output** — `text/tabwriter` for human-friendly list output
