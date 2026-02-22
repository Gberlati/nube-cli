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
- Global flag:
  - `--color=auto|always|never` (default `auto`)
  - `--json` (JSON output to stdout)
  - `--plain` (TSV output to stdout; stable/parseable; disables colors)
  - `--select` (comma-separated fields for JSON projection; supports dot paths, e.g. `--select id,name.en`)
  - `--force` (skip confirmations for destructive commands)
  - `--no-input` (never prompt; fail instead)
  - `--version` (print version)

Notes:

- We run `SilenceUsage: true` and print errors ourselves (colored when possible).
- `NO_COLOR` is respected.

## Output (TTY-aware colors)

- `github.com/muesli/termenv` is used to detect rich TTY capabilities and render colored output.
- Colors are enabled when:
  - output is a rich terminal and `--color=auto`, and `NO_COLOR` is not set; or
  - `--color=always`
- Colors are disabled when:
  - `--color=never`; or
  - `NO_COLOR` is set

Implementation: `internal/ui/ui.go`.


## Auth + secret storage

### OAuth client credentials (non-secret-ish)

- Stored on disk in the per-user config directory:
  - `$(os.UserConfigDir())/nube-cli/credentials.json` (default client)
  - `$(os.UserConfigDir())/nube-cli/credentials-<client>.json` (named clients)
- Written with mode `0600`.
- Command:
  - `nube auth credentials <credentials.json>`
  - `nube --client <name> auth credentials <credentials.json>`
  - `nube auth credentials list`

Implementation: `internal/config/*`.

### Access tokens (secrets)

- Stored in OS credential store via `github.com/99designs/keyring`.
- Key namespace is `nube-cli` (keyring `ServiceName`).
- Key format: `token:<client>:<email>` (default client uses `token:default:<email>`)
- Legacy key format: `token:<email>` (migrated on first read)
- Stored payload is JSON (access token).
- Fallback: if no OS credential store is available, keyring may use its encrypted "file" backend:
  - Directory: `$(os.UserConfigDir())/nube-cli/keyring/` (one file per key)
  - Password: prompts on TTY; for non-interactive runs set `NUBE_KEYRING_PASSWORD`

Current minimal management commands:

- `nube auth tokens list` (keys only)
- `nube auth tokens delete <email>`

Implementation: `internal/secrets/store.go`.

### OAuth flow

Two flows:

- **Broker (default)**: An OAuth broker (Cloudflare Worker) holds the app credentials. The CLI starts a local callback server, opens `{brokerURL}/start?port={CallbackPort}` in the browser, and receives the access token directly via `?token=...&user_id=...`. No local `credentials.json` needed. The broker URL can be overridden via `--broker-url` or `NUBE_AUTH_BROKER`.
- **Native (custom app)**: For developers with their own Tienda Nube app credentials. Stores `credentials.json` locally, opens the Tienda Nube authorization page, receives a `?code=...` callback, and exchanges it for an access token. Used when no broker URL is configured (i.e., `DefaultBrokerURL` is empty and `--broker-url` is not set).

## OAuth Broker (Cloudflare Worker)

The default broker flow uses a stateless Cloudflare Worker that holds the Tienda Nube app credentials (`CLIENT_ID`, `CLIENT_SECRET`) server-side. This way CLI users can authenticate without needing their own `credentials.json`.

### Architecture

- **Stateless**: no database or KV — each request is self-contained.
- **Secrets**: `CLIENT_ID` and `CLIENT_SECRET` are set via `wrangler secret put` (never checked into source).

### Endpoints

- `GET /start?port=<port>` — validates the port and redirects (302) to the Tienda Nube authorization page with `state=<port>`.
- `GET /callback?code=<code>&state=<port>` — exchanges the authorization code for an access token (server-to-server POST), then redirects (302) to `http://127.0.0.1:<port>/callback?token=<token>&user_id=<user_id>`.
- `GET /robots.txt` — returns `Disallow: /`.
- Everything else — 404.

### Deployment

```sh
cd broker
npm install
wrangler secret put CLIENT_ID
wrangler secret put CLIENT_SECRET
wrangler deploy
```

### Code location

`broker/` — see `broker/src/index.js`.

## Config layout

- Base config dir: `$(os.UserConfigDir())/nube-cli/`
- Files:
  - `config.json` (JSON5; comments and trailing commas allowed)
  - `credentials.json` (OAuth client id/secret; default client)
  - `credentials-<client>.json` (OAuth client id/secret; named clients)
- Secrets:
  - access tokens in keyring

We intentionally avoid storing access tokens in plain JSON on disk.

Environment:

- `NUBE_ACCOUNT=you@gmail.com` (email or alias; used when `--account` is not set; otherwise uses keyring default or a single stored token)
- `NUBE_CLIENT=work` (select OAuth client bucket; see `--client`)
- `NUBE_AUTH_BROKER=https://...` (override the default OAuth broker URL; see `--broker-url`)
- `NUBE_KEYRING_PASSWORD=...` (used when keyring falls back to encrypted file backend in non-interactive environments)
- `NUBE_KEYRING_BACKEND={auto|keychain|file}` (force backend; use `file` to avoid Keychain prompts and pair with `NUBE_KEYRING_PASSWORD` for non-interactive)
- `config.json` can also set `keyring_backend` (JSON5; env vars take precedence)
- `config.json` can also set `account_aliases` for `nube auth alias` (JSON5)
- `config.json` can also set `account_clients` (email -> client) and `client_domains` (domain -> client)

## Commands (current + planned)

### Implemented

- `nube config keys`
- `nube config get <key>`
- `nube config list`
- `nube config path`
- `nube config set <key> <value>`
- `nube config unset <key>`
- `nube version`
- `nube auth credentials <credentials.json>` — store OAuth client credentials
- `nube auth credentials list` — list stored credentials
- `nube --client <name> auth credentials <credentials.json>` — store named client credentials
- `nube auth add <email>` — OAuth flow to obtain and store access token
- `nube auth list` — list stored accounts
- `nube auth alias list` — list account aliases
- `nube auth alias set <alias> <email>` — create account alias
- `nube auth alias unset <alias>` — remove account alias
- `nube auth status` — show auth config, keyring backend, resolved account
- `nube auth remove <email>` — remove stored token (with confirmation)
- `nube auth tokens list` — list raw keyring keys
- `nube auth tokens delete <email>` — delete stored token (with confirmation)

### Planned

- `nube abandoned-checkout list [--since-id ID] [--created-at-max DATE] [--updated-at-max DATE] [--page N] [--per-page N] [--fields FIELDS]`
- `nube abandoned-checkout get <checkoutId>`
- `nube abandoned-checkout coupon add <cartId> --coupon-id ID`

- `nube billing plans create --code CODE [--external-reference REF] [--description DESC]`
- `nube billing plans update <planId> [--code CODE] [--external-reference REF] [--description DESC]`
- `nube billing plans delete <planId>`
- `nube billing subscriptions get <conceptCode> <serviceId>`
- `nube billing subscriptions update <conceptCode> <serviceId> [--amount-currency CURRENCY] [--amount-value VALUE] [--plan-id UUID] [--plan-external-id ID]`
- `nube billing charges create <serviceId> --description DESC --from-date DATE --to-date DATE --amount-value VALUE --amount-currency CURRENCY --concept-code CODE [--external-reference REF]`

- `nube blog posts list <blogId> [--page N]`
- `nube blog posts get <blogId> <postId>`
- `nube blog posts create <blogId> --metadata JSON [--content HTML] [--published] [--thumbnail URL]`
- `nube blog posts update <blogId> <postId> [--metadata JSON] [--content HTML] [--published] [--thumbnail URL]`
- `nube blog posts delete <blogId> <postId>`
- `nube blog posts publish <blogId> <postId>`
- `nube blog posts unpublish <blogId> <postId>`
- `nube blog posts media upload <blogId> --file PATH`
- `nube blog posts thumbnail upload <blogId> --file PATH`

- `nube business-rules callbacks set <storeId> <domain> --url URL --event EVENT`

- `nube cart get <cartId>`
- `nube cart line-items delete <cartId> <lineItemId>`
- `nube cart coupons delete <cartId> <couponId>`

- `nube category custom-fields create --name NAME [--description DESC] --value-type text_list|text|numeric|date [--read-only] [--values V1,V2,...]`
- `nube category custom-fields list`
- `nube category custom-fields get <id>`
- `nube category custom-fields update <id> [--values V1,V2,...]`
- `nube category custom-fields delete <id>`
- `nube category custom-fields owners <id>`
- `nube category <categoryId> custom-fields list`
- `nube category <categoryId> custom-fields values set [--id UUID --value VALUE]...`

- `nube category list [--since-id ID] [--language LANG] [--handle HANDLE] [--parent-id ID] [--created-at-min DATE] [--created-at-max DATE] [--updated-at-min DATE] [--updated-at-max DATE] [--page N] [--per-page N] [--fields FIELDS]`
- `nube category get <categoryId> [--fields FIELDS]`
- `nube category create --name NAME [--parent ID] [--google-shopping-category CAT]`
- `nube category update <categoryId> [--name NAME] [--parent ID] [--google-shopping-category CAT]`
- `nube category delete <categoryId>`

- `nube coupon list [--q CODE] [--valid true|false] [--status activated|deactivated] [--discount-type percentage|absolute|shipping] [--sort-by CRITERIA] [--page N] [--per-page N] [--fields FIELDS]`
- `nube coupon get <couponId>`
- `nube coupon create --code CODE --type percentage|absolute|shipping [--value V] [--max-uses N] [--start-date DATE] [--end-date DATE] [--min-price P] [--includes-shipping] [--first-consumer-purchase] [--combines-with-other-discounts] [--only-cheapest-shipping] [--categories IDs] [--products IDs]`
- `nube coupon update <couponId> [--code CODE] [--type TYPE] [--value V] [--valid true|false]`
- `nube coupon delete <couponId>`

- `nube customer custom-fields create --name NAME [--description DESC] --value-type text_list|text|numeric|date [--read-only] [--values V1,V2,...]`
- `nube customer custom-fields list`
- `nube customer custom-fields get <id>`
- `nube customer custom-fields update <id> [--values V1,V2,...]`
- `nube customer custom-fields delete <id>`
- `nube customer custom-fields owners <id>`
- `nube customer <customerId> custom-fields list`
- `nube customer <customerId> custom-fields values set [--id UUID --value VALUE]...`

- `nube customer list [--since-id ID] [--q TEXT] [--email EMAIL] [--created-at-min DATE] [--created-at-max DATE] [--updated-at-min DATE] [--updated-at-max DATE] [--page N] [--per-page N] [--fields FIELDS]`
- `nube customer get <customerId> [--fields FIELDS]`
- `nube customer create --name NAME --email EMAIL [--phone PHONE] [--addresses JSON] [--send-email-invite] [--password PASS]`
- `nube customer update <customerId> [--name NAME] [--email EMAIL] [--phone PHONE] [--note TEXT]`
- `nube customer delete <customerId>`

- `nube discount callbacks set --url URL`
- `nube discount callbacks update --url URL`

- `nube draft-order list`
- `nube draft-order get <draftOrderId>`
- `nube draft-order create --contact-name NAME --contact-lastname LASTNAME --contact-email EMAIL --payment-status unpaid|pending_confirmation|paid --products JSON [--contact-phone PHONE] [--note TEXT] [--discount VALUE] [--discount-type absolute|percentage] [--shipping JSON]`
- `nube draft-order confirm <draftOrderId>`
- `nube draft-order delete <draftOrderId>`

- `nube fulfillment-order list <orderId> [--aggregates custom_fields]`
- `nube fulfillment-order get <orderId> <fulfillmentOrderId>`
- `nube fulfillment-order update <orderId> <fulfillmentOrderId> [--status UNPACKED|PACKED|DISPATCHED|READY_FOR_PICKUP|DELIVERED] [--tracking-code CODE] [--tracking-url URL] [--notify-customer]`
- `nube fulfillment-order delete <orderId> <fulfillmentOrderId>`
- `nube fulfillment-order tracking-events list <orderId> <fulfillmentOrderId>`
- `nube fulfillment-order tracking-events get <orderId> <fulfillmentOrderId> <trackingEventId>`
- `nube fulfillment-order tracking-events create <orderId> <fulfillmentOrderId> --status STATUS --description DESC [--address ADDR] [--happened-at DATE] [--estimated-delivery-at DATE]`
- `nube fulfillment-order tracking-events update <orderId> <fulfillmentOrderId> <trackingEventId> --status STATUS --description DESC [--address ADDR] [--happened-at DATE] [--estimated-delivery-at DATE]`
- `nube fulfillment-order tracking-events delete <orderId> <fulfillmentOrderId> <trackingEventId>`

- `nube location list`
- `nube location get <locationId>`
- `nube location create --name NAME --address JSON`
- `nube location update <locationId> [--name NAME] [--address JSON]`
- `nube location delete <locationId>`
- `nube location set-default <locationId>`
- `nube location priorities update [--id ID --priority N]...`
- `nube location inventory-levels <locationId> [--variant-id ID] [--page N] [--per-page N]`

- `nube metafield list <products|product_variants|categories|pages|orders|customers> [--owner-id ID] [--namespace NS] [--filter-key KEY] [--page N] [--per-page N] [--fields FIELDS]`
- `nube metafield get <metafieldId>`
- `nube metafield create --key KEY --value VALUE --namespace NS --owner-id ID --owner-resource RESOURCE [--description DESC]`
- `nube metafield update <metafieldId> [--value VALUE] [--description DESC]`
- `nube metafield delete <metafieldId>`

- `nube order custom-fields create --name NAME [--description DESC] --value-type text_list|text|numeric|date [--read-only] [--values V1,V2,...]`
- `nube order custom-fields list`
- `nube order custom-fields get <id>`
- `nube order custom-fields update <id> [--values V1,V2,...]`
- `nube order custom-fields delete <id>`
- `nube order custom-fields owners <id>`
- `nube order <orderId> custom-fields list`
- `nube order <orderId> custom-fields values set [--id UUID --value VALUE]...`

- `nube order list [--since-id ID] [--status any|open|closed|cancelled] [--payment-status any|pending|authorized|paid|abandoned|refunded|voided] [--shipping-status any|unpacked|unfulfilled|fulfilled] [--channels store|api|form|meli|pos] [--created-at-min DATE] [--created-at-max DATE] [--updated-at-min DATE] [--updated-at-max DATE] [--customer-ids IDs] [--q TEXT] [--page N] [--per-page N] [--fields FIELDS] [--aggregates fulfillment_orders|custom_fields]`
- `nube order get <orderId> [--fields FIELDS] [--aggregates fulfillment_orders]`
- `nube order history values <orderId> [--status PENDING|CANCELLED|PAID]`
- `nube order history editions <orderId>`
- `nube order create --gateway GATEWAY --products JSON --customer JSON --billing-address JSON --shipping-address JSON --shipping SHIPPING --shipping-option OPTION --shipping-pickup-type ship|pickup --shipping-cost-customer COST [--payment-status STATUS] [--currency CODE] [--language CODE] [--note TEXT] [--location-id ID]`
- `nube order update <orderId> [--owner-note TEXT] [--status open|closed|cancelled]`
- `nube order close <orderId>`
- `nube order open <orderId>`
- `nube order cancel <orderId> [--reason customer|inventory|fraud|other] [--email] [--restock]`

- `nube page list [--page N]`
- `nube page get <pageId>`
- `nube page create --title TITLE --content HTML [--seo-handle HANDLE] [--seo-title TITLE] [--seo-description DESC] [--language LANG] [--publish]`
- `nube page update <pageId> [--title TITLE] [--content HTML]`
- `nube page delete <pageId>`

- `nube product custom-fields create --name NAME [--description DESC] --value-type text_list|text|numeric|date [--read-only] [--values V1,V2,...]`
- `nube product custom-fields list`
- `nube product custom-fields get <id>`
- `nube product custom-fields update <id> [--values V1,V2,...]`
- `nube product custom-fields delete <id>`
- `nube product custom-fields owners <id>`
- `nube product <productId> custom-fields list`
- `nube product <productId> custom-fields values set [--id UUID --value VALUE]...`

- `nube product images list <productId> [--since-id ID] [--src URL] [--position N] [--page N] [--per-page N] [--fields FIELDS]`
- `nube product images get <productId> <imageId> [--fields FIELDS]`
- `nube product images create <productId> --src URL [--position N]`
- `nube product images upload <productId> --filename NAME --attachment BASE64 [--position N]`
- `nube product images update <productId> <imageId> [--position N]`
- `nube product images delete <productId> <imageId>`

- `nube product variant custom-fields create --name NAME [--description DESC] --value-type text_list|text|numeric|date [--read-only] [--values V1,V2,...]`
- `nube product variant custom-fields list`
- `nube product variant custom-fields get <id>`
- `nube product variant custom-fields update <id> [--values V1,V2,...]`
- `nube product variant custom-fields delete <id>`
- `nube product variant custom-fields owners <id>`
- `nube product variant <variantId> custom-fields list`
- `nube product variant <variantId> custom-fields values set [--id UUID --value VALUE]...`

- `nube product variants list <productId> [--since-id ID] [--created-at-min DATE] [--created-at-max DATE] [--updated-at-min DATE] [--updated-at-max DATE] [--page N] [--per-page N] [--fields FIELDS]`
- `nube product variants get <productId> <variantId> [--fields FIELDS]`
- `nube product variants create <productId> --values JSON [--price PRICE] [--stock N] [--sku SKU] [--weight W] [--barcode CODE] [--age-group newborn|infant|toddler|kids|adult] [--gender female|male|unisex]`
- `nube product variants update <productId> <variantId> [--price PRICE] [--promotional-price PRICE] [--stock N] [--sku SKU] [--weight W] [--barcode CODE]`
- `nube product variants replace <productId> [--values JSON --price PRICE --stock N]...`
- `nube product variants patch <productId> [--id ID --values JSON --price PRICE]...`
- `nube product variants delete <productId> <variantId>`
- `nube product variants stock update <productId> --action replace|variation --value N [--id variantId]`

- `nube product list [--ids IDs] [--since-id ID] [--q TEXT] [--handle HANDLE] [--category-id ID] [--published true|false] [--free-shipping true|false] [--created-at-min DATE] [--created-at-max DATE] [--updated-at-min DATE] [--updated-at-max DATE] [--sort-by CRITERIA] [--page N] [--per-page N] [--fields FIELDS]`
- `nube product get <productId> [--fields FIELDS]`
- `nube product get-by-sku <sku>`
- `nube product create --name NAME [--description HTML] [--variants JSON] [--images JSON] [--categories IDs] [--brand BRAND] [--tags TAGS] [--published] [--free-shipping] [--seo-title TITLE] [--seo-description DESC]`
- `nube product update <productId> [--name NAME] [--description HTML] [--published] [--categories IDs] [--tags TAGS]`
- `nube product delete <productId>`
- `nube product stock-price update [--id PRODUCT_ID --variant-id VARIANT_ID --price PRICE --stock N]...`

- `nube script list [--page N] [--per-page N]`
- `nube script get <scriptId>`
- `nube script associate <scriptId> [--query-params JSON]`
- `nube script update <scriptId> [--query-params JSON]`
- `nube script dissociate <scriptId>`

- `nube shipping-carrier list`
- `nube shipping-carrier get <carrierId>`
- `nube shipping-carrier create --name NAME --callback-url URL --types ship|pickup|ship,pickup [--active]`
- `nube shipping-carrier update <carrierId> [--name NAME] [--callback-url URL] [--types ship|pickup|ship,pickup] [--active true|false]`
- `nube shipping-carrier delete <carrierId>`
- `nube shipping-carrier options list <carrierId>`
- `nube shipping-carrier options get <carrierId> <optionId>`
- `nube shipping-carrier options create <carrierId> --code CODE --name NAME [--additional-days N] [--additional-cost N] [--allow-free-shipping] [--active]`
- `nube shipping-carrier options update <carrierId> <optionId> [--additional-days N] [--additional-cost N] [--allow-free-shipping true|false] [--active true|false]`
- `nube shipping-carrier options delete <carrierId> <optionId>`
- `nube fulfillment list <orderId>`
- `nube fulfillment get <orderId> <fulfillmentId>`
- `nube fulfillment create <orderId> --status STATUS --description DESC [--city CITY] [--province PROVINCE] [--country COUNTRY] [--happened-at DATE] [--estimated-delivery-at DATE]`
- `nube fulfillment delete <orderId> <fulfillmentId>`

- `nube store get [--fields FIELDS]`

- `nube transaction list <orderId>`
- `nube transaction get <orderId> <transactionId>`
- `nube transaction create <orderId> --payment-provider-id ID --payment-method-type TYPE [--payment-method-id ID] --first-event-type TYPE --first-event-status STATUS --first-event-amount VALUE --first-event-currency CURRENCY [--first-event-happened-at DATE] [--external-id ID] [--external-url URL] [--info JSON]`
- `nube transaction events create <orderId> <transactionId> --type TYPE --status STATUS --happened-at DATE [--amount VALUE] [--currency CURRENCY] [--authorization-code CODE] [--failure-code CODE] [--info JSON]`

- `nube webhook list [--since-id ID] [--url URL] [--event EVENT] [--created-at-min DATE] [--created-at-max DATE] [--page N] [--per-page N] [--fields FIELDS]`
- `nube webhook get <webhookId> [--fields FIELDS]`
- `nube webhook create --event EVENT --url URL`
- `nube webhook update <webhookId> [--event EVENT] [--url URL]`
- `nube webhook delete <webhookId>`

## Output formats

Default: human-friendly tables (stdlib `text/tabwriter`).

- Parseable stdout:
  - `--json`: JSON objects/arrays suitable for scripting
  - `--plain`: stable TSV (tabs preserved; no alignment; no colors)
  - `--select`: JSON field projection with dot-notation (e.g. `--select id,name.en,variants`). When applied to arrays, projects each element. Requires `--json`.
- Human-facing hints/progress are written to stderr so stdout can be safely captured.
- Colors are only used for human-facing output and are disabled automatically for `--json` and `--plain`.

We avoid heavy table deps unless we decide we need them.

## Code layout (current)

- `cmd/nube/main.go` — binary entrypoint
- `internal/cmd/*` — kong command structs
- `internal/ui/*` — color + printing
- `internal/outfmt/*` — output mode (JSON/plain) + JSON encoder
- `internal/errfmt/*` — user-friendly error formatting
- `internal/config/*` — config paths + credential parsing/writing + aliases
- `internal/secrets/*` — keyring store
- `internal/api/*` — Tienda Nube API client (HTTP client, retry transport, circuit breaker, TLS enforcement, typed errors: `APIError`, `AuthError`, `NotFoundError`, `RateLimitError`, `ValidationError`, `PaymentRequiredError`, `PermissionDeniedError`, `CircuitBreakerError`, pagination)
- `internal/oauth/*` — OAuth 2.0 flow (broker default, native browser fallback)
- `broker/` — OAuth broker Cloudflare Worker (holds app credentials server-side)

## API error handling

The Tienda Nube API returns three different error response formats:

1. **Business errors** (`{"code": "...", "message": "...", "description": "..."}`): Used for structured API errors (e.g. 422 business logic failures, 5xx errors with messages).
2. **Parse errors** (`{"error": "..."}`): Used for 400 errors when request body is malformed JSON.
3. **Field validation** (`{"field_name": ["error1", "error2"]}`): Used for 422 validation errors with per-field error messages.

Status code mapping:
- **401**: `AuthError` — authentication failed
- **402**: `PaymentRequiredError` — store subscription suspended
- **403**: `PermissionDeniedError` — insufficient permissions/scope
- **404**: `NotFoundError` — resource not found
- **422**: `ValidationError` (field format) or `APIError` (business format)
- **429**: Retried by `RetryTransport`; `RateLimitError` after exhausting retries
- **5xx**: Retried by `RetryTransport`; `APIError` after exhausting retries

## Rate limiting

The Tienda Nube API uses a leaky bucket rate limiter:
- Bucket size: 40 requests
- Leak rate: 2 requests/second
- `X-Rate-Limit-Limit`: bucket size
- `X-Rate-Limit-Remaining`: remaining requests
- `X-Rate-Limit-Reset`: **milliseconds** until bucket refills

The `RetryTransport` automatically retries 429 responses with exponential backoff, using `X-Rate-Limit-Reset` (ms) when available. Maximum 5 retries for rate limits, 2 retries for 5xx errors.

## Circuit breaker

The API client includes a circuit breaker that prevents cascading failures:
- Opens after 5 consecutive failures (5xx responses, transport errors, or exhausted retries)
- While open, all requests fail immediately with `CircuitBreakerError`
- Automatically resets after 30 seconds, allowing a probe request
- Closes on the first successful response

## HTTP client defaults

- TLS 1.2+ enforced for all API connections
- Default timeout: 30 seconds (configurable via `WithTimeout` option)

## Formatting, linting, tests

### Formatting

Pinned tools, installed into local `.tools/` via `make tools`:

- `mvdan.cc/gofumpt@v0.9.2`
- `golang.org/x/tools/cmd/goimports@v0.42.0`
- `github.com/golangci/golangci-lint/cmd/golangci-lint@v2.10.1`

Commands:

- `make fmt` — applies `goimports` + `gofumpt`
- `make fmt-check` — formats and fails if Go files or `go.mod/go.sum` change

### Lint

- `golangci-lint` with config in `.golangci.yml`
- `make lint`
- Notable linters: `bodyclose` (HTTP response body leak detection), `rowserrcheck`, `sqlclosecheck`

### Pre-commit hooks

- `.lefthook.yml` — runs `make fmt-check`, `make lint`, `make test` in parallel before each commit
- Install: `lefthook install`

### Tests

- stdlib `testing` (+ `httptest` when we add OAuth/API tests)
- `make test`

### Integration tests (local only)

There is an opt-in integration test suite guarded by build tags (not run in CI).

- requires:
  - stored `credentials.json` (or `credentials-<client>.json`) via `nube auth credentials ...`
  - refresh token in keyring via `nube auth add <email>`
- Run:
  - `NUBE_IT_ACCOUNT=you@gmail.com go test -tags=integration ./internal/integration`

## CI (GitHub Actions)

Workflow: `.github/workflows/ci.yml`

- runs on push + PR
- uses `actions/setup-go` with `go-version-file: go.mod`
- runs:
  - `make tools`
  - `make fmt-check`
  - `go test ./...`
  - `golangci-lint` (pinned `v2.10.1`)

## Releases

Workflow: `.github/workflows/release.yml`

- triggered on tag push (`v*`)
- uses `goreleaser-action` with `.goreleaser.yaml`
- builds multi-platform binaries: linux/darwin/windows (amd64/arm64)
- darwin builds use `CGO_ENABLED=1` (for Keychain support)
- creates GitHub release with archives and checksums

