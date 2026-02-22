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
- **Auth** — OAuth authorization flow (browser, manual, remote), credential management, account aliases
- **Parseable output** — JSON (`--json`) and TSV (`--plain`) modes for scripting and automation

**Planned:**

- **Store** — get store info and general settings
- **Products** — list/search/get/create/update/delete products and variants; look up by SKU; bulk-update stock and price; manage product images
- **Categories** — list/get/create/update/delete categories; organize storefront navigation hierarchy
- **Customers** — list/search/get/create/update/delete customers; inspect contact info and purchase history
- **Orders** — list/search/get/create/update orders; open/close/cancel; view audit history; manage fulfillment orders and tracking events
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
# Check version
nube version

# Show config file path
nube config path

# List all configuration values
nube config list

# Set keyring backend
nube config set keyring_backend file

# JSON output for scripting
nube version --json
nube config list --json

# Extract specific fields from JSON output
nube product list --json --select id,name.en,variants
```

## Authentication

nube-cli uses OAuth to authenticate with the Tienda Nube API. You need OAuth client credentials (a `credentials.json` file containing `client_id` and `client_secret`) to get started.

### 1. Store OAuth credentials

```bash
# From a file
nube auth credentials set /path/to/credentials.json

# From stdin
cat credentials.json | nube auth credentials set -

# List stored credentials
nube auth credentials list
```

### 2. Authorize a store

**Browser flow** (default) — opens the browser for authorization:

```bash
nube auth add user@example.com
```

**Manual flow** — paste the authorization code in the terminal:

```bash
nube auth add user@example.com --manual
```

**Remote flow** — two-step process for headless servers:

```bash
# Step 1: print the authorization URL (copy to a browser)
nube auth add user@example.com --remote --step 1

# Step 2: exchange the redirect URL for a token
nube auth add user@example.com --remote --step 2 --auth-url "http://localhost:8910/callback?code=..."
```

### 3. Manage accounts

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

### 4. Account aliases

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

### 5. Multiple stores

Use `--client` to manage separate OAuth credential sets and token buckets:

```bash
# Store credentials for a named client
nube auth credentials set creds.json --client beta

# Authorize under the named client
nube auth add user@example.com --client beta

# Use the named client for API calls
nube <command> --account user@example.com --client beta
```

## Environment Variables

 - `NUBE_ACCOUNT` — Account email or alias for API commands (used when `--account` is not set)
 - `NUBE_CLIENT` — OAuth client bucket name (used when `--client` is not set)
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

## License

MIT

## Links

 - [Github Repository](https://github.com/gberlati/nube-cli)
 - [Tienda Nube Documentation](https://tiendanube.github.io/api-documentation)

## Credits

This project is inspired by Peter Steinberg's google CLI.
 - [gogcli](https://github.com/steipete/gogcli)
