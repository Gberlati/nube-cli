# ☁️ nube-cli - a CLI for managing Tienda Nube stores.

![GitHub Repo Banner](https://ghrb.waren.build/banner?header=%E2%98%81%EF%B8%8F+nube-cli&bg=0055D4-001554&color=FFFFFF&headerfont=Google+Sans&watermarkpos=bottom-right)
<!-- Created with GitHub Repo Banner by Waren Gonzaga: https://ghrb.waren.build -->

Fast, agentic and script-friendly CLI for managing Tienda Nube stores from the terminal. JSON-first output, with support for multiple stores.

## Features

**Implemented:**

- **Configuration** — get/set/list/unset config values; inspect config paths and keyring backend
- **Version** — print version, commit, and build date
- **Multiple accounts** — manage multiple Tienda Nube stores simultaneously (with aliases)
- **Command allowlist** — restrict top-level commands for sandboxed/agent runs
- **Secure credential storage** — OS keyring or encrypted on-disk keyring (configurable)
- **Auth** — OAuth authorization via broker (zero-setup) or native browser flow (custom app credentials), credential management, account aliases
- **Parseable output** — JSON (`--json`) and TSV (`--plain`) modes for scripting and automation
- **Store** — get store info (name, email, domain, plan)
- **Products** — list/get products with full filtering and pagination; look up by SKU
- **Orders** — list/get orders with filtering by status, payment, shipping, and date ranges
- **Categories** — list/get categories with filtering
- **Customers** — list/get customers with search and filtering
- **Agent helpers** — stable exit codes (`nube agent exit-codes`), machine-readable command schema (`nube schema`)
- **Desire paths** — top-level shortcuts: `nube shop`, `nube products`, `nube orders`, `nube status`, `nube login`
- **Command aliases** — short forms: `prod`, `ord`, `cat`, `cust`, `help-json`

**Planned:**

- **Products (writes)** — create/update/delete products and variants; bulk-update stock and price; manage product images
- **Categories (writes)** — create/update/delete categories
- **Customers (writes)** — create/update/delete customers
- **Orders (writes)** — create/update orders; open/close/cancel; view audit history; manage fulfillment orders and tracking events
- **Draft Orders** — create/confirm/delete draft orders from outside channels
- **Abandoned Checkouts** — list/get abandoned checkouts; apply coupons to recover carts
- **Coupons & Discounts** — list/get/create/update/delete coupons; define cart-level promotion and tier discount rules
- **Transactions** — list/get/create transactions per order; post events to drive payment state transitions
- **Shipping** — manage carriers and rate options; manage fulfillment events per order
- **Locations** — list/get/create/update/delete store locations; set priorities and default; inspect inventory levels
- **Blog & Pages** — manage blog posts and static store pages; upload images; manage SEO metadata
- **Metafields** — manage namespaced key-value metafields scoped to any resource
- **Webhooks** — list/get/create/update/delete event subscriptions
- **Billing** — manage app plans, subscriptions, and charges
- **FTP Support** — manage store themes by connecting via FTP

## Installation

### Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/gberlati/nube-cli/releases) page. Pre-built binaries are available for linux, macOS, and Windows (amd64/arm64).

### Build from Source

```bash
git clone https://github.com/gberlati/nube-cli.git
cd nube-cli
make
```

Run:

```bash
./bin/nube --help
```

## Quick Start

```bash
# Authorize a store (opens browser — no setup required)
nube auth add user@example.com
# or use the shortcut:
nube login user@example.com

# Check store info
nube shop --json

# List products (first 5)
nube products --json --per-page 5

# List orders
nube orders --json --status open

# Get a single product
nube product get 12345 --json

# Look up by SKU
nube product get-by-sku ABC-001 --json

# List categories
nube category list --json

# JSON output for scripting
nube version --json
nube config list --json

# Agent helpers
nube agent exit-codes --json
nube schema --json
```

## Authentication

nube-cli supports two OAuth flows: a **broker flow** (default, zero setup) and a **native browser flow** (for developers with their own Tienda Nube app credentials).

### Default (Broker)

Just run:

```bash
nube auth add user@example.com
```

The browser opens, you authorize, and the token is stored automatically. No credentials file needed.

Override the broker URL if needed:

```bash
nube auth add user@example.com --broker-url https://my-broker.example.com
# or via environment variable
NUBE_AUTH_BROKER=https://my-broker.example.com nube auth add user@example.com
```

### Custom App (Native)

For developers with their own Tienda Nube app credentials:

**1. Store OAuth credentials**

```bash
# From a file
nube auth credentials set /path/to/credentials.json

# From stdin
cat credentials.json | nube auth credentials set -

# List stored credentials
nube auth credentials list
```

**2. Authorize a store**

When no broker URL is configured and credentials are present, the CLI uses the native browser flow:

```bash
nube auth add user@example.com
```

Use `--client` for named credential sets:

```bash
nube auth credentials set creds.json --client beta
nube auth add user@example.com --client beta
```

### Account Management

```bash
# List authorized accounts
nube auth list

# Check auth status and keyring backend
nube auth status

# Remove an account
nube auth remove user@example.com

# List stored token keys
nube auth tokens list
```

### Account Aliases

Aliases let you refer to accounts by short names instead of email addresses:

```bash
# Set an alias
nube auth alias set prod user@example.com

# Use an alias with any command
nube <command> --account prod

# List aliases
nube auth alias list

# Remove an alias
nube auth alias unset prod
```

### Token-based (Agents & CI)

For non-interactive environments (Docker, CI, agents), export a token from keyring
to a `.env` file, then use it without keyring access:

```bash
# Export token to .env file (one-time, on a machine with keyring access)
nube auth token user@example.com --export .env

# Then use the .env file anywhere:
docker run --env-file .env myimage nube products --json
# or:
source .env && nube products --json
# or:
env $(cat .env) nube products --json

# Inline per-command (no file needed):
NUBE_ACCESS_TOKEN=abc123 NUBE_USER_ID=456 nube products --json

# Multi-account: set per invocation
NUBE_ACCESS_TOKEN=$TOK1 NUBE_USER_ID=$ID1 nube shop --json
NUBE_ACCESS_TOKEN=$TOK2 NUBE_USER_ID=$ID2 nube shop --json
```

### Multiple Stores

Use `--client` to manage separate OAuth credential sets and token buckets:

```bash
# Store credentials for a named client
nube auth credentials set creds.json --client beta

# Authorize under the named client
nube auth add user@example.com --client beta

# Use the named client for API calls
nube <command> --account user@example.com --client beta
```

## CLI Command Reference

### Desire Paths (shortcuts)

- `nube shop` — show store info (alias for `nube store get`)
- `nube products [flags]` — list products (alias for `nube product list`)
- `nube orders [flags]` — list orders (alias for `nube order list`)
- `nube status` — show auth status (alias for `nube auth status`)
- `nube login <email>` — authorize account (alias for `nube auth add`)

### Implemented

- `nube version` — print version, commit, and build date
- `nube store get` — show store information (id, name, email, domain, plan)
- `nube product list [flags]` — list products with pagination and filters
- `nube product get <id>` — get a product by ID
- `nube product get-by-sku <sku>` — get a product by SKU
- `nube order list [flags]` — list orders with pagination and filters
- `nube order get <id>` — get an order by ID
- `nube category list [flags]` — list categories
- `nube category get <id>` — get a category by ID
- `nube customer list [flags]` — list customers
- `nube customer get <id>` — get a customer by ID
- `nube agent exit-codes` — print stable exit code map
- `nube schema` — machine-readable command schema (JSON)
- `nube config keys` — list valid config keys
- `nube config get <key>` — get a config value
- `nube config list` — list all config values
- `nube config path` — show config file path
- `nube config set <key> <value>` — set a config value
- `nube config unset <key>` — unset a config value
- `nube auth credentials set <path>` — store OAuth client credentials
- `nube auth credentials list` — list stored credentials
- `nube auth add <email>` — authorize and store access token
- `nube auth list` — list stored accounts
- `nube auth alias list` — list account aliases
- `nube auth alias set <alias> <email>` — create account alias
- `nube auth alias unset <alias>` — remove account alias
- `nube auth status` — show auth config and keyring backend
- `nube auth remove <email>` — remove stored token
- `nube auth token [email]` — print access token for an account
- `nube auth token [email] --export FILE` — export token to dotenv file
- `nube auth tokens list` — list raw keyring keys
- `nube auth tokens delete <email>` — delete stored token

### Command Aliases

- `nube prod` = `nube product`
- `nube ord` = `nube order`
- `nube cat` = `nube category`
- `nube cust` = `nube customer`
- `nube help-json` = `nube schema`

### Stable Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | ok | Success |
| 1 | error | Generic error |
| 2 | usage | Invalid usage / bad arguments |
| 3 | auth_required | Authentication required (HTTP 401) |
| 4 | not_found | Resource not found (HTTP 404) |
| 5 | permission_denied | Permission denied (HTTP 403) |
| 6 | rate_limited | Rate limited (HTTP 429) |
| 7 | retryable | Retryable server error (HTTP 5xx) |
| 8 | config | Missing config or credentials |
| 9 | cancelled | User cancelled |
| 10 | payment_required | Payment required (HTTP 402) |
| 11 | validation | Validation error (HTTP 422) |

## Environment Variables

 - `NUBE_ACCESS_TOKEN` — Access token for API commands (bypasses keyring; for agents and CI)
 - `NUBE_USER_ID` — Store/user ID (used with `NUBE_ACCESS_TOKEN`)
 - `NUBE_ACCOUNT` — Account email or alias for API commands (used when `--account` is not set)
 - `NUBE_CLIENT` — OAuth client bucket name (used when `--client` is not set)
 - `NUBE_AUTH_BROKER` — OAuth broker URL (overrides the default broker)
 - `NUBE_KEYRING_PASSWORD` — Password used as fallback to encrypt tokens if no OS keyring is available
 - `NUBE_KEYRING_BACKEND` — Force keyring backend: `auto` (default), `keychain`, or `file`
 - `NUBE_JSON` — Default to JSON output (`1`, `true`, `yes`)
 - `NUBE_PLAIN` — Default to plain/TSV output (`1`, `true`, `yes`)
 - `NUBE_COLOR` — Color mode: `auto` (default), `always`, or `never`
 - `NUBE_ENABLE_COMMANDS` — Comma-separated allowlist of top-level commands (e.g., `config,version`)

## Security

Access tokens are stored in the OS keyring (macOS Keychain, Linux SecretService/D-Bus) via `github.com/99designs/keyring`. When no OS keyring is available, tokens fall back to an encrypted file backend in `$XDG_CONFIG_HOME/nube-cli/keyring/` (protected with `NUBE_KEYRING_PASSWORD`).

All config and credential files are written with `0600` permissions. Config directories use `0700`.

TLS 1.2+ is enforced for all API connections. A circuit breaker prevents cascading failures when the API is unresponsive. API errors are typed and user-friendly messages are shown for common issues (auth failures, payment required, permission denied, validation errors).

## Development

### Build from source

```bash
make          # build binary to ./bin/nube
make tools    # install pinned dev tools
make fmt      # format code
make lint     # lint code
make test     # run tests
make fmt-check # check formatting (CI)
```

### Pre-commit hooks

```bash
lefthook install  # runs fmt-check, lint, test before each commit
```

## License

MIT

## Links

 - [Github Repository](https://github.com/gberlati/nube-cli)
 - [Tienda Nube Documentation](https://tiendanube.github.io/api-documentation)

## Credits

This project is inspired by Peter Steinberg's google CLI.
 - [gogcli](https://github.com/steipete/gogcli)
