## Why

The current web architecture uses nginx as a dumb reverse proxy — it forwards `/api/` to the Go apiserver and `/aot.api.v1.` to ConnectRPC, with no intelligence in between. This causes problems:

1. **No session management** — every request is stateless, auth is per-request via API key header. No CSRF protection, no session tokens, no refresh flow.
2. **No request validation** — malformed requests pass straight through to the backend. The apiserver does validation but errors come back as gRPC status codes that the frontend has to interpret.
3. **No response shaping** — the frontend gets raw gRPC/REST responses and must transform them (mapRun, roleFromSpanName, displaySpanName). This logic should live server-side.
4. **WebSocket fragility** — the nginx `map $http_upgrade` hack for exec WebSocket is brittle. A BFF can natively handle WebSocket upgrade.
5. **No rate limiting** — any client can flood the API. The BFF can enforce per-session rate limits.
6. **No aggregation** — the frontend makes multiple calls (list runs + get traces + get logs) that could be a single BFF endpoint.
7. **Dual protocol complexity** — the frontend talks ConnectRPC for CRUD and REST for files/logs/traces. A BFF can unify these behind a single REST API.
8. **No caching** — trace data and file listings are re-fetched on every poll. The BFF can cache and invalidate intelligently.

## What Changes

- Replace nginx reverse proxy with a **Go BFF server** that serves the static frontend AND proxies/shapes API requests
- The BFF runs as a sidecar to the apiserver (or replaces the web deployment entirely)
- All frontend requests go through the BFF — no direct apiserver access
- BFF handles: session management, request validation, response shaping, WebSocket upgrade, rate limiting, caching, CORS
- The existing apiserver remains unchanged — the BFF is a client of it, not a replacement

## Capabilities

### New Capabilities
- `bff-server`: Go HTTP server that serves static files, proxies API calls to the apiserver, and adds session management, validation, caching, and response shaping
- `unified-api`: Single REST API surface for the frontend — no more dual ConnectRPC + REST. All endpoints are `/api/v1/*` with JSON request/response
- `session-auth`: Cookie-based session with CSRF token, replacing per-request API key header. Sessions bound to a configurable auth provider (API key, OIDC, or open)
- `response-shaping`: BFF transforms backend responses before returning to frontend — applies displaySpanName remapping, role resolution, cost estimation, span deduplication server-side

### Modified Capabilities
- None (apiserver unchanged, BFF is additive)

## Impact

- **New binary**: `cmd/bff/main.go` — Go HTTP server
- **Helm chart**: Replace `aot-web` deployment (nginx + static files) with `aot-bff` deployment (Go server + static files)
- **Frontend**: Change `apiFetch` base URL to BFF, remove ConnectRPC client dependency, all requests go through unified REST API
- **Dockerfile**: New `Dockerfile.bff` that builds Go binary + embeds static web assets
- **Security**: Sessions, CSRF, rate limiting, CORS all handled by BFF
- **Performance**: Server-side caching of traces, file listings, run status. SSE push instead of polling.
