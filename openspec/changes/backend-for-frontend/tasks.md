## 1. BFF Server Skeleton

- [ ] 1.1 Create `cmd/bff/main.go` with Go HTTP server, ConnectRPC client to apiserver, embedded static files via `go:embed`
- [ ] 1.2 Add SPA fallback handler (serve index.html for non-file routes)
- [ ] 1.3 Add health check endpoints (`/healthz`, `/readyz`)
- [ ] 1.4 Create `Dockerfile.bff` that builds Go binary + copies web dist into embed directory

## 2. Unified REST API Routes

- [ ] 2.1 `GET /api/v1/runs` — proxy to ConnectRPC ListAgentRuns, transform response to JSON
- [ ] 2.2 `POST /api/v1/runs` — proxy to ConnectRPC CreateAgentRun
- [ ] 2.3 `GET /api/v1/runs/{id}` — proxy to ConnectRPC GetAgentRun
- [ ] 2.4 `POST /api/v1/runs/{id}/cancel` — proxy to ConnectRPC CancelAgentRun
- [ ] 2.5 `POST /api/v1/runs/{id}/input` — proxy to ConnectRPC SendHumanInput
- [ ] 2.6 Proxy all existing REST endpoints (traces, logs, files, exec, debug, classify) to apiserver

## 3. WebSocket Proxy

- [ ] 3.1 Native WebSocket proxy for `/api/v1/runs/{id}/exec` (replace nginx map hack)
- [ ] 3.2 Handle WebSocket upgrade, bidirectional forwarding, timeout, cleanup

## 4. Middleware

- [ ] 4.1 Session middleware — cookie-based, in-memory store, configurable secret
- [ ] 4.2 CSRF middleware — token generation and validation for state-changing requests
- [ ] 4.3 Rate limiting middleware — per-session, configurable limit (default 100 req/s)
- [ ] 4.4 CORS middleware — configurable allowed origins
- [ ] 4.5 Request ID middleware — X-Request-Id header for tracing

## 5. Response Shaping

- [ ] 5.1 Span name remapping in traces response (displaySpanName server-side)
- [ ] 5.2 Cost estimation in traces response (compute from token metadata)
- [ ] 5.3 Span deduplication (move readSpansFile dedup logic into BFF)
- [ ] 5.4 Run response shaping — include computed fields (feature status, PR URL)

## 6. Caching

- [ ] 6.1 In-memory cache with TTL (use `sync.Map` or `github.com/patrickmn/go-cache`)
- [ ] 6.2 Phase-based TTL: 3s for running, 5min for terminal
- [ ] 6.3 Cache invalidation on write operations (create, cancel, input)

## 7. Deployment

- [ ] 7.1 Add `aot-bff` to Helm chart (deployment, service, configmap)
- [ ] 7.2 Update `dev-values.yaml` with BFF config (apiserver URL, session secret, auth mode)
- [ ] 7.3 Update Taskfile with `build:bff` target
- [ ] 7.4 Remove nginx deployment, Dockerfile.web, nginx.conf from Helm chart
- [ ] 7.5 Update frontend `apiFetch` to remove ConnectRPC dependency (all requests go to unified REST)

## 8. Tests

- [ ] 8.1 Unit test: SPA fallback serves index.html for non-file routes
- [ ] 8.2 Unit test: API proxy forwards requests and returns responses
- [ ] 8.3 Unit test: Response shaping transforms span names and computes costs
- [ ] 8.4 Unit test: Cache returns cached responses within TTL
- [ ] 8.5 Unit test: Rate limiter returns 429 when exceeded
- [ ] 8.6 Integration test: Full request flow through BFF to apiserver
- [ ] 8.7 Contract test: BFF REST API response matches frontend type expectations
