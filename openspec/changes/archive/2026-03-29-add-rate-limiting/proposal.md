## Why

The platform has no rate limiting on any HTTP endpoint. The existing IP bucket implementation in `internal/bff/middleware.go` is unbounded (identified in `go-concurrency-fixes`) and covers only the BFF layer — the apiserver, webhook, and classify endpoints are fully unprotected. A single misbehaving client or misconfigured runner can saturate the server, degrade LLM throughput, or rack up unexpected Ollama/LiteLLM costs.

## What Changes

- Add per-IP rate limiting middleware to the apiserver HTTP mux (sliding window, configurable RPS + burst)
- Add per-IP rate limiting to webhook endpoint (separate, tighter limit — webhook abuse is a common attack vector)
- Add per-model rate limiting on `/api/v1/classify` and `/api/v1/chat/stream` to protect LLM gateway throughput
- Replace the unbounded `sync.Map` IP bucket in `internal/bff/middleware.go` with a TTL-evicting implementation (coordinated with `go-concurrency-fixes`)
- Expose rate limit headers (`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`) on all rate-limited endpoints
- Add rate limit config to Helm values (`apiserver.rateLimit.*`)

## Capabilities

### New Capabilities
- `api-rate-limiting`: Per-IP and per-model rate limiting across apiserver endpoints with configurable limits, burst allowance, TTL eviction, and standard rate limit response headers

### Modified Capabilities
- (none — the BFF middleware fix is an implementation detail, not a spec-level requirement change)

## Impact

- `internal/server/` — new middleware, applied to HTTP mux in `cmd/apiserver/main.go`
- `internal/bff/middleware.go` — replace unbounded IP bucket (coordinate with `go-concurrency-fixes`)
- `deploy/helm/aot/values.yaml` + `templates/apiserver.yaml` — new `rateLimit` config section
- No breaking changes to API contracts — rate limit responses use standard HTTP 429
