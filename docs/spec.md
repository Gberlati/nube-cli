# nube-cli spec

## Goal

Build a single, clean, modern Go CLI that talks to Tienda Nube's complete API and FTP Server.

## Non-Goals

- Running an MCP server (this is a CLI)

## Language/runtime

- Go `1.26` (see `go.mod`)

## CLI framework

- `github.com/alecthomas/kong`
- Root command: `nube`
- Global flags:
  - `--store` / `-s` — store profile name (env: `NUBE_STORE`)
  - `--json` / `-j` — JSON output to stdout
  - `--plain` / `-p` — TSV output (stable, parseable, no colors)
  - `--select` / `-S` — comma-separated fields for JSON projection (supports dot paths)
  - `--force` / `-y` — skip confirmations
  - `--no-input` — never prompt; fail instead
  - `--dry-run` / `-n` — show what would be done
  - `--verbose` / `-v` — debug logging
  - `--color` — `auto|always|never` (default `auto`)
  - `--enable-commands` — command allowlist
  - `--version` — print version

Notes:

- We run `SilenceUsage: true` and print errors ourselves (colored when possible).
- `NO_COLOR` is respected.

## Output (TTY-aware colors)

- `github.com/muesli/termenv` for TTY detection and colored output.
- Colors enabled: rich terminal + `--color=auto` + no `NO_COLOR`; or `--color=always`.
- Colors disabled: `--color=never`; or `NO_COLOR` is set.

Implementation: `internal/ui/ui.go`.

## Auth + credential storage

### Credential file

All credentials are stored in a single file:

- Path: `~/.config/nube-cli/credentials.json` (0600 permissions)
- Format:
  ```json
  {
    "default_store": "my-shop",
    "stores": {
      "my-shop": {
        "store_id": "1234567",
        "access_token": "abc123...",
        "email": "owner@myshop.com",
        "scopes": ["read_products", "write_products"],
        "created_at": "2025-01-15T10:30:00Z"
      }
    },
    "oauth_clients": {
      "default": {
        "client_id": "12345",
        "client_secret": "secret..."
      }
    }
  }
  ```

Store resolution priority: `--store` flag → `NUBE_STORE` env → `default_store` → single-store auto-select.

Implementation: `internal/credstore/credstore.go`.

### OAuth client credentials (optional)

- Stored in the `oauth_clients` section of `credentials.json`
- Only needed for native OAuth flow; broker flow requires none

### OAuth flow

Two flows:

- **Broker (default)**: A Cloudflare Worker holds the app credentials. The CLI starts a local callback server, opens `{brokerURL}/start?port={port}` in the browser, and receives the token via `?token=...&user_id=...`. No credentials file needed. Override via `NUBE_AUTH_BROKER`.
- **Native (custom app)**: For developers with their own Tienda Nube app. Requires OAuth client credentials in `credentials.json`. Opens the authorization page, receives `?code=...`, exchanges for a token.

Implementation: `internal/oauth/oauth.go`.

## OAuth Broker (Cloudflare Worker)

Stateless worker that holds Tienda Nube app credentials server-side.

### Endpoints

- `GET /start?port=<port>` — redirects to Tienda Nube authorization page
- `GET /callback?code=<code>&state=<port>` — exchanges code for token, redirects to local CLI
- `GET /robots.txt` — `Disallow: /`

### Deployment

```sh
cd broker
npm install
wrangler secret put CLIENT_ID
wrangler secret put CLIENT_SECRET
wrangler deploy
```

Code: `broker/src/index.js`.

## Config

- Base dir: `~/.config/nube-cli/`
- `config.json` (JSON5) — app config (currently only `client_domains`)
- `credentials.json` — store profiles + OAuth client credentials

Environment variables:

| Variable | Description |
|----------|-------------|
| `NUBE_ACCESS_TOKEN` | Access token (bypasses credential file) |
| `NUBE_USER_ID` | Store/user ID (with `NUBE_ACCESS_TOKEN`) |
| `NUBE_STORE` | Select store profile |
| `NUBE_AUTH_BROKER` | Override OAuth broker URL |
| `NUBE_JSON` | Default to JSON output |
| `NUBE_PLAIN` | Default to TSV output |
| `NUBE_COLOR` | Color mode: `auto` / `always` / `never` |
| `NUBE_ENABLE_COMMANDS` | Command allowlist |

## Commands

### Implemented

- `nube login [name]` — OAuth flow, save store profile
- `nube logout <name>` — remove store profile
- `nube auth list` / `status` / `token [name]` / `default <name>`
- `nube auth credentials set <path>` / `list`
- `nube store get`
- `nube product list [flags]` / `get <id>` / `get-by-sku <sku>`
- `nube order list [flags]` / `get <id>`
- `nube category list [flags]` / `get <id>`
- `nube customer list [flags]` / `get <id>`
- `nube config list` / `path`
- `nube agent exit-codes`
- `nube schema`
- `nube version`
- Shortcuts: `nube shop`, `nube products`, `nube orders`, `nube status`
- Aliases: `prod`, `ord`, `cat`, `cust`, `help-json`

### Planned

Write operations for all resources (products, orders, categories, customers), plus:
abandoned checkouts, coupons, draft orders, fulfillment orders, locations, metafields,
webhooks, blog/pages, billing, shipping carriers, transactions, FTP support.

See the full planned command list in the codebase comments.

## Output formats

Default: human-friendly tables (stdlib `text/tabwriter`).

- `--json`: JSON objects/arrays for scripting
- `--plain`: stable TSV (no alignment, no colors)
- `--select`: JSON field projection with dot-notation (e.g. `--select id,name.en`). Requires `--json`.
- Human-facing hints/progress go to stderr so stdout can be captured.

## Code layout

- `cmd/nube/main.go` — binary entrypoint
- `internal/cmd/` — kong command structs and handlers
- `internal/api/` — HTTP client, retry transport, circuit breaker, TLS enforcement, typed errors, pagination
- `internal/oauth/` — OAuth 2.0 flow (broker + native)
- `internal/credstore/` — credential file storage (zero external deps)
- `internal/config/` — app config (JSON5)
- `internal/outfmt/` — output mode + JSON encoder
- `internal/errfmt/` — user-friendly error formatting
- `internal/ui/` — color + terminal printing
- `broker/` — OAuth broker Cloudflare Worker

## API error handling

The Tienda Nube API returns three error formats:

1. **Business errors** (`{"code", "message", "description"}`): structured API errors
2. **Parse errors** (`{"error": "..."}`): malformed request body (400)
3. **Field validation** (`{"field_name": ["error1"]}`): per-field validation (422)

Status code mapping:
- 401 → `AuthError`
- 402 → `PaymentRequiredError`
- 403 → `PermissionDeniedError`
- 404 → `NotFoundError`
- 422 → `ValidationError` or `APIError`
- 429 → retried; `RateLimitError` after exhaustion
- 5xx → retried; `APIError` after exhaustion

## Stable exit codes

| Code | Name | Condition |
|------|------|-----------|
| 0 | ok | Success |
| 1 | error | Generic error |
| 2 | usage | Invalid arguments |
| 3 | auth_required | HTTP 401 |
| 4 | not_found | HTTP 404 |
| 5 | permission_denied | HTTP 403 |
| 6 | rate_limited | HTTP 429 |
| 7 | retryable | HTTP 5xx or circuit breaker |
| 8 | config | Missing config or credentials |
| 9 | cancelled | User cancelled |
| 10 | payment_required | HTTP 402 |
| 11 | validation | HTTP 422 |

Machine-readable: `nube agent exit-codes --json`

## Rate limiting

Tienda Nube leaky bucket: 40 requests, 2 req/s leak rate.
Headers: `X-Rate-Limit-Limit`, `X-Rate-Limit-Remaining`, `X-Rate-Limit-Reset` (milliseconds).
`RetryTransport` retries 429 up to 5 times with exponential backoff.

## Circuit breaker

Opens after 5 consecutive failures. Resets after 30 seconds. All requests fail with `CircuitBreakerError` while open.

## HTTP client defaults

- TLS 1.2+ enforced
- Default timeout: 30 seconds

## Build & CI

### Formatting

Pinned tools in `.tools/`:
- `mvdan.cc/gofumpt`
- `golang.org/x/tools/cmd/goimports`
- `github.com/golangci/golangci-lint/cmd/golangci-lint`

### Commands

- `make fmt` / `make fmt-check` / `make lint` / `make test`
- `lefthook install` — pre-commit hooks

### CI

`.github/workflows/ci.yml`: setup-go, fmt-check, test, lint.

### Releases

`.github/workflows/release.yml`: tag-triggered, goreleaser, multi-platform builds.
