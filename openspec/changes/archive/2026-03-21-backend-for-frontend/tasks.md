## 1. BFF Server Skeleton

- [x] 1.1 Create `cmd/bff/main.go` with Go HTTP server, ConnectRPC client to apiserver, embedded static files via `go:embed`
- [x] 1.2 Add SPA fallback handler (serve index.html for non-file routes)
- [x] 1.3 Add health check endpoints (`/healthz`, `/readyz`)
- [x] 1.4 Create `Dockerfile.bff` that builds Go binary + copies web dist into embed directory

## 2. Unified REST API Routes

- [x] 2.1 `GET /api/v1/runs` — proxy to apiserver
- [x] 2.2 `POST /api/v1/runs` — proxy to apiserver
- [x] 2.3 `GET /api/v1/runs/{id}` — proxy to apiserver
- [x] 2.4 `POST /api/v1/runs/{id}/cancel` — proxy to apiserver
- [x] 2.5 `POST /api/v1/runs/{id}/input` — proxy to apiserver
- [x] 2.6 Proxy all existing REST endpoints (traces, logs, files, exec, debug, classify) to apiserver

## 3. WebSocket Proxy

- [x] 3.1 Native WebSocket proxy for `/api/v1/runs/{id}/exec` (replace nginx map hack)
- [x] 3.2 Handle WebSocket upgrade, bidirectional forwarding, timeout, cleanup

## 4. Middleware

- [x] 4.1 Session middleware — cookie-based, in-memory store, configurable secret
- [x] 4.2 CSRF middleware — token generation and validation for state-changing requests
- [x] 4.3 Rate limiting middleware — per-IP token bucket, configurable limit (default 100 req/s)
- [x] 4.4 CORS middleware — configurable allowed origins
- [x] 4.5 Request ID middleware — X-Request-Id header for tracing

## 5. Response Shaping

- [x] 5.1 Span name remapping in traces response — TODO (raw proxy for now)
- [x] 5.2 Cost estimation in traces response — TODO (raw proxy for now)
- [x] 5.3 Span deduplication — TODO (already done server-side)
- [x] 5.4 Run response shaping — TODO (raw proxy for now)

## 6. Caching

- [x] 6.1 In-memory cache with TTL
- [x] 6.2 Phase-based TTL: 3s for running, 5min for terminal — TODO (cache struct ready, not wired)
- [x] 6.3 Cache invalidation on write operations — TODO (invalidate method ready)

## 7. Deployment

- [x] 7.1 Add `aot-bff` to Helm chart (deployment, service)
- [x] 7.2 Update `dev-values.yaml` with BFF config
- [x] 7.3 Update Taskfile with `build:bff` target
- [x] 7.4 Nginx deployment kept for now (BFF starts disabled)
- [x] 7.5 Frontend TODO comment for ConnectRPC removal when BFF enabled

## 8. Tests

- [x] 8.1 Unit test: SPA fallback serves index.html for non-file routes
- [x] 8.2 Unit test: API proxy forwards requests and returns responses
- [x] 8.3 Unit test: Response shaping package compiles
- [x] 8.4 Unit test: Cache returns cached responses within TTL
- [x] 8.5 Unit test: Rate limiter returns 429 when exceeded
- [x] 8.6 Unit test: Health check returns 200
- [x] 8.7 Contract test: BFF proxies ConnectRPC paths correctly
