# Milestone 3-7: API Hardening, Error Handling, and Tooling

## What was built

### Milestone 3: Bug Fixes & Error Handling

**3A. Rate limit reset header fix (BUG B1)**
- `internal/api/transport.go`: Changed `X-Rate-Limit-Reset` parsing from seconds to milliseconds per API docs
- `internal/api/transport_test.go`: Updated test to use `3000` (ms) instead of `3` (seconds)

**3B. Error response parsing (BUG B2)**
- `internal/api/client.go`: Rewrote `parseErrorResponse()` to handle three API error formats:
  - `{"code","message","description"}` (business errors)
  - `{"error":"..."}` (400 parse errors)
  - `{"field":["msg"]}` (422 field validation)
- `internal/api/errors.go`: Added `ValidationError` type with `Fields map[string][]string`
- `internal/errfmt/errfmt.go`: Added `formatValidationError()` with sorted, deterministic field output

**3C. 402 Payment Required (BUG B3)**
- `internal/api/errors.go`: Added `PaymentRequiredError` type
- `internal/api/client.go`: Maps 402 to `PaymentRequiredError`
- `internal/errfmt/errfmt.go`: User-friendly message about subscription

### Milestone 4: API Client Hardening

**4A. Circuit Breaker**
- Created `internal/api/circuitbreaker.go`: `CircuitBreaker` struct with `sync.Mutex`, threshold=5, reset=30s
- Created `internal/api/circuitbreaker_test.go`: Tests for open/close cycle, threshold, timeout reset
- Modified `internal/api/transport.go`: Integrated into `RetryTransport` with `IsOpen()` check and `RecordSuccess()`/`RecordFailure()`
- Added `CircuitBreakerError` type to `errors.go`

**4B. TLS 1.2+ Enforcement**
- `internal/api/client.go`: Added `newBaseTransport()` that clones `http.DefaultTransport` and sets `TLSClientConfig.MinVersion = tls.VersionTLS12`

**4C. Default HTTP Timeout**
- `internal/api/client.go`: Added `defaultHTTPTimeout = 30s`, `WithTimeout()` option, set in `New()`

**4D. PermissionDeniedError (403)**
- `internal/api/errors.go`: Added `PermissionDeniedError` type
- `internal/api/client.go`: Maps 403 to `PermissionDeniedError`
- `internal/errfmt/errfmt.go`: Format case for permission denied

### Milestone 5: --select Flag

- `internal/outfmt/outfmt.go`: Added `JSONTransform`, `selectFields()`, `getAtPath()`, context helpers, updated `WriteJSON()`
- `internal/cmd/root.go`: Added `Select` field to `RootFlags`, wired into context
- `internal/outfmt/outfmt_test.go`: Tests for select on objects, arrays, nested paths, missing fields

### Milestone 6: Testing Infrastructure

- `internal/cmd/testhelpers_test.go`: Added `captureStderr()`, `withStdin()`, `runKong()`
- Created `internal/api/testhelpers_test.go`: `withRequestTracking()`, `chainMiddleware()`

### Milestone 7: Linting & Tooling

**7A. Linters**
- `.golangci.yml`: Added `bodyclose`, `rowserrcheck`, `sqlclosecheck`
- Fixed 3 bodyclose issues (pagination.go, client_test.go, transport_test.go)

**7B. Lefthook**
- Created `.lefthook.yml`: pre-commit runs fmt-check, lint, test in parallel

**7C. Goreleaser**
- Created `.goreleaser.yaml`: multi-platform builds (linux/darwin/windows, amd64/arm64)
- Created `.github/workflows/release.yml`: tag-triggered release workflow

### Milestone 8: Documentation

- `docs/spec.md`: Added API error handling, rate limiting, circuit breaker, HTTP defaults, --select, linting, release sections
- `AGENTS.md`: Added linting notes, testing section with helper docs
- `README.md`: Added releases installation, --select example, security notes

## Files created
- `internal/api/circuitbreaker.go`
- `internal/api/circuitbreaker_test.go`
- `internal/api/testhelpers_test.go`
- `.lefthook.yml`
- `.goreleaser.yaml`
- `.github/workflows/release.yml`
- `docs/memory/milestone-3.md`

## Files modified
- `.golangci.yml`
- `internal/api/client.go`
- `internal/api/client_test.go`
- `internal/api/errors.go`
- `internal/api/errors_test.go`
- `internal/api/pagination.go`
- `internal/api/transport.go`
- `internal/api/transport_test.go`
- `internal/cmd/root.go`
- `internal/cmd/testhelpers_test.go`
- `internal/errfmt/errfmt.go`
- `internal/errfmt/errfmt_test.go`
- `internal/outfmt/outfmt.go`
- `internal/outfmt/outfmt_test.go`
- `docs/spec.md`
- `AGENTS.md`
- `README.md`
