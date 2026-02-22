# Milestone 4: First Resource Commands + Agent Friendliness

## What was done

### Phase 0: Shared Helpers + Agent Infrastructure
- **`exit.go`**: Renamed `ExitError` -> `ExitErr`. Added 12 stable exit codes (0-11) with `stableExitCode(err)` mapping API errors to specific codes. Execute() wraps errors with stable codes before returning.
- **`api_helpers.go`**: `newAPIClient` var (swappable for tests), `PaginationFlags` (--page, --per-page/--max), `addQueryParam`, `extractI18n` (multilingual field extraction with es>pt>en priority), `jsonStr` (safe map value extraction).
- **`agent.go`**: `AgentCmd` + `AgentExitCodesCmd` — prints exit code map as JSON or table.
- **`schema.go`**: `SchemaCmd` — introspects kong parser, emits JSON schema of all commands/flags/args.

### Phase 1-4: Resource Commands (read-only)
- **`store.go`**: `StoreCmd` + `StoreGetCmd` — GET /store, human output shows id/name/email/domain/plan.
- **`product.go`**: `ProductCmd` + `ProductListCmd`/`ProductGetCmd`/`ProductGetBySkuCmd` — full filtering, pagination, table output with variant count and price.
- **`order.go`**: `OrderCmd` + `OrderListCmd`/`OrderGetCmd` — full filtering including status/payment/shipping, aggregates support.
- **`category.go`**: `CategoryCmd` + `CategoryListCmd`/`CategoryGetCmd` — filtering, subcategory count in table.
- **`customer.go`**: `CustomerCmd` + `CustomerListCmd`/`CustomerGetCmd` — search, email filter.

### Phase 5: Registration + Desire Paths
- **`root.go`**: CLI struct expanded with all domain commands + desire paths (`shop`, `products`, `orders`, `status`, `login`). Command aliases: `prod`, `ord`, `cat`, `cust`, `help-json`.

### Tests
- `api_helpers_test.go` — PaginationFlags, addQueryParam, extractI18n
- `exit_test.go` — added `TestStableExitCode` covering all API error types
- `store_test.go` — JSON + human output via mock HTTP server
- `product_test.go` — list/get/get-by-sku with mock server
- `order_test.go` — list/get JSON + table output
- `category_test.go` — list/get JSON + table output
- `customer_test.go` — list/get JSON + table output

### Documentation
- `README.md` — updated features, quick start, command reference, exit codes table
- `docs/spec.md` — moved implemented commands, added stable exit codes + agent helpers sections

## Key patterns established

1. **`newAPIClient` var** — swappable in tests via `setupMockAPIClient(t, handler)`, matching `openSecretsStore` pattern.
2. **`map[string]any` responses** — no typed structs; --json passes raw API data; human output extracts specific keys via `jsonStr`/`extractI18n`.
3. **Pagination**: `--page 0` = fetch all pages (CollectAllPages), `--page N` = single page. `--per-page` controls page size.
4. **`decodeList`** — shared decoder for all list endpoints: `func(resp) ([]map[string]any, error)`.
5. **Table output** — all list commands: header row + tabwriter. All get commands: key-value via `writeResult`.
6. **Desire paths** — embedded command structs at root level with `name:""` kong tag.

## File inventory

| File | Lines | Purpose |
|------|-------|---------|
| `api_helpers.go` | ~107 | newAPIClient, PaginationFlags, helpers |
| `api_helpers_test.go` | ~85 | Helper tests |
| `agent.go` | ~48 | Agent exit-codes command |
| `schema.go` | ~125 | Schema introspection command |
| `store.go` | ~75 | Store get command |
| `store_test.go` | ~80 | Store tests |
| `product.go` | ~190 | Product list/get/get-by-sku |
| `product_test.go` | ~145 | Product tests |
| `order.go` | ~140 | Order list/get |
| `order_test.go` | ~115 | Order tests |
| `category.go` | ~135 | Category list/get |
| `category_test.go` | ~105 | Category tests |
| `customer.go` | ~125 | Customer list/get |
| `customer_test.go` | ~110 | Customer tests |
