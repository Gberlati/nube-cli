# nube-cli

Fast, script-friendly CLI for managing Tienda Nube stores from the terminal.

## Features

- **Auth** — OAuth via broker (zero-setup) or native browser flow; multi-store profiles
- **Store** — get store info (name, email, domain, plan)
- **Products** — list/get with full filtering, pagination, and SKU lookup
- **Orders** — list/get with status, payment, shipping, and date filters
- **Categories** — list/get with filtering
- **Customers** — list/get with search and filtering
- **Output** — JSON (`--json`), TSV (`--plain`), field selection (`--select id,name.en`)
- **Agent helpers** — stable exit codes, machine-readable command schema
- **Shortcuts** — `nube shop`, `nube products`, `nube orders`, `nube status`, `nube login`
- **Command allowlist** — restrict top-level commands for sandboxed/agent runs

## Installation

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/gberlati/nube-cli/releases). Available for Linux, macOS, and Windows (amd64/arm64).

### Build from Source

```bash
git clone https://github.com/gberlati/nube-cli.git
cd nube-cli
make
./bin/nube --help
```

## Quick Start

```bash
# Log in (opens browser — no setup required)
nube login

# Check store info
nube shop --json

# List products
nube products --json --per-page 5

# List open orders
nube orders --json --status open

# Get a single product
nube product get 12345 --json

# Look up by SKU
nube product get-by-sku ABC-001 --json

# Multi-store
nube login dev-store
nube products --store dev-store --json
```

## Authentication

Two OAuth flows: **broker** (default, zero setup) and **native** (your own app credentials).

### Broker (default)

```bash
nube login
```

Opens the browser, you authorize, and the token is saved to `~/.config/nube-cli/credentials.json`. No credentials file needed.

Override the broker URL:

```bash
NUBE_AUTH_BROKER=https://my-broker.example.com nube login
```

### Native (custom app)

For developers with their own Tienda Nube app:

```bash
# Store OAuth client credentials
nube auth credentials set /path/to/credentials.json

# Log in (uses native flow when no broker is configured)
nube login
```

### Store Management

```bash
nube auth list              # List store profiles
nube auth status            # Show credential file + active store
nube auth token [name]      # Print access token
nube auth default <name>    # Set default profile
nube logout <name>          # Remove a profile
```

### CI / Non-interactive

```bash
# Option 1: Copy credentials.json to the CI machine
nube products --store my-shop --json

# Option 2: Environment variables (no credential file needed)
NUBE_ACCESS_TOKEN=abc123 NUBE_USER_ID=456 nube products --json
```

## Commands

### Shortcuts

| Command | Equivalent |
|---------|------------|
| `nube shop` | `nube store get` |
| `nube products` | `nube product list` |
| `nube orders` | `nube order list` |
| `nube status` | `nube auth status` |
| `nube login` | OAuth login flow |
| `nube logout` | Remove profile |

### Auth

- `nube login [name]` — authorize and save a store profile
- `nube logout <name>` — remove a store profile
- `nube auth list` — list store profiles
- `nube auth status` — show credential file path and active store
- `nube auth token [name]` — print access token
- `nube auth default <name>` — set default store profile
- `nube auth credentials set <path>` — store OAuth client credentials
- `nube auth credentials list` — list OAuth client credentials

### Resources

- `nube store get`
- `nube product list [flags]` / `get <id>` / `get-by-sku <sku>`
- `nube order list [flags]` / `get <id>`
- `nube category list [flags]` / `get <id>`
- `nube customer list [flags]` / `get <id>`

### Config & Agent

- `nube config list` / `path`
- `nube agent exit-codes`
- `nube schema`

### Aliases

`prod`, `ord`, `cat`, `cust`, `help-json`

## Global Flags

| Flag | Short | Env | Description |
|------|-------|-----|-------------|
| `--store` | `-s` | `NUBE_STORE` | Store profile name |
| `--json` | `-j` | `NUBE_JSON` | JSON output |
| `--plain` | `-p` | `NUBE_PLAIN` | TSV output (no colors) |
| `--select` | `-S` | | Field selection (e.g. `id,name.en`) |
| `--force` | `-y` | | Skip confirmations |
| `--no-input` | | | Never prompt; fail instead |
| `--dry-run` | `-n` | | Show what would be done |
| `--verbose` | `-v` | | Enable debug logging |
| `--color` | | `NUBE_COLOR` | `auto` / `always` / `never` |
| `--enable-commands` | | `NUBE_ENABLE_COMMANDS` | Command allowlist |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NUBE_ACCESS_TOKEN` | Access token (bypasses credential file; for CI) |
| `NUBE_USER_ID` | Store/user ID (used with `NUBE_ACCESS_TOKEN`) |
| `NUBE_STORE` | Store profile name |
| `NUBE_AUTH_BROKER` | Custom OAuth broker URL |
| `NUBE_JSON` | Default to JSON output |
| `NUBE_PLAIN` | Default to TSV output |
| `NUBE_COLOR` | Color mode: `auto` / `always` / `never` |
| `NUBE_ENABLE_COMMANDS` | Comma-separated command allowlist |

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | ok | Success |
| 1 | error | Generic error |
| 2 | usage | Invalid usage |
| 3 | auth_required | HTTP 401 |
| 4 | not_found | HTTP 404 |
| 5 | permission_denied | HTTP 403 |
| 6 | rate_limited | HTTP 429 |
| 7 | retryable | HTTP 5xx |
| 8 | config | Missing config or credentials |
| 9 | cancelled | User cancelled |
| 10 | payment_required | HTTP 402 |
| 11 | validation | HTTP 422 |

## Security

Credentials are stored in `~/.config/nube-cli/credentials.json` with `0600` permissions. Config directories use `0700`.

TLS 1.2+ is enforced for all API connections. A circuit breaker prevents cascading failures. Rate limiting is handled automatically with exponential backoff.

## Development

```bash
make          # build to ./bin/nube
make test     # run tests
make lint     # lint
make fmt      # format
make ci       # all of the above
lefthook install  # pre-commit hooks
```

## License

MIT

## Links

- [GitHub Repository](https://github.com/gberlati/nube-cli)
- [Tienda Nube API Documentation](https://tiendanube.github.io/api-documentation)
