## Architecture

### Current Architecture

```
Browser ──HTTP──▶ nginx:3000 ──proxy──▶ apiserver:50055
                    │                      │
                    ├─ /api/*    ─────────▶│ REST endpoints
                    ├─ /aot.api.v1.* ────▶│ ConnectRPC endpoints
                    ├─ /api/*/exec ──WS──▶│ WebSocket exec
                    └─ /* ─── static files │
```

Problems: no auth, no caching, no response shaping, dual protocol, WebSocket hacks.

### Proposed Architecture

```
Browser ──HTTP──▶ bff:3000 ──internal──▶ apiserver:50055
                    │                       │
                    ├─ /api/v1/* ──────────▶│ (ConnectRPC client)
                    │   ↳ validates         │
                    │   ↳ shapes response   │
                    │   ↳ caches            │
                    │   ↳ rate limits       │
                    │                       │
                    ├─ /api/v1/*/exec ─WS──▶│ (native WebSocket proxy)
                    │                       │
                    ├─ /api/v1/*/traces ───▶│ (cached, deduplicated)
                    │                       │
                    └─ /* ─── embedded      │
                         static files       │
                         (go:embed)         │
```

### BFF Server Design

```go
// cmd/bff/main.go
func main() {
    // 1. Connect to apiserver via ConnectRPC
    apiClient := connectrpc.NewClient(apiserverURL)

    // 2. Create session store (in-memory or Redis)
    sessions := session.NewMemoryStore()

    // 3. Create rate limiter
    limiter := ratelimit.New(100) // 100 req/s per session

    // 4. Register routes
    mux := http.NewServeMux()

    // Unified REST API (replaces ConnectRPC + REST split)
    mux.Handle("/api/v1/runs", bff.ListRuns(apiClient, cache))
    mux.Handle("/api/v1/runs/{id}", bff.GetRun(apiClient, cache))
    mux.Handle("/api/v1/runs", bff.CreateRun(apiClient)) // POST
    mux.Handle("/api/v1/runs/{id}/cancel", bff.CancelRun(apiClient))
    mux.Handle("/api/v1/runs/{id}/input", bff.SendInput(apiClient))

    // Pass-through with caching
    mux.Handle("/api/v1/runs/{id}/traces", bff.Traces(apiClient, cache))
    mux.Handle("/api/v1/runs/{id}/logs", bff.Logs(apiClient))
    mux.Handle("/api/v1/runs/{id}/files", bff.Files(apiClient))

    // WebSocket proxy (native, no nginx hack)
    mux.Handle("/api/v1/runs/{id}/exec", bff.ExecProxy(apiClient))

    // Classification
    mux.Handle("/api/v1/classify", bff.Classify(apiClient))

    // Static files (embedded)
    mux.Handle("/", http.FileServer(http.FS(staticFS)))

    // 5. Wrap with middleware
    handler := middleware.Chain(
        middleware.Session(sessions),
        middleware.CSRF(),
        middleware.RateLimit(limiter),
        middleware.CORS(allowedOrigins),
        middleware.RequestID(),
    )(mux)

    http.ListenAndServe(":3000", handler)
}
```

### Response Shaping

The BFF transforms backend responses before returning to the frontend:

```go
// Example: GET /api/v1/runs/{id}/traces
func Traces(apiClient *api.Client, cache *Cache) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        runID := r.PathValue("id")

        // 1. Check cache
        if cached, ok := cache.Get("traces:" + runID); ok {
            writeJSON(w, cached)
            return
        }

        // 2. Fetch from apiserver
        spans := apiClient.GetTraces(runID)

        // 3. Shape response (server-side, not client-side)
        for i := range spans {
            // Remap legacy names
            spans[i].Name = displaySpanName(spans[i].Name)
            // Resolve operation color hint
            spans[i].OperationHint = resolveOperation(spans[i])
            // Compute cost if tokens available
            if tokens, ok := spans[i].Metadata["gen_ai.usage.input_tokens"]; ok {
                spans[i].Metadata["cost_usd"] = estimateCost(tokens, ...)
            }
        }

        // 4. Cache (TTL based on run phase)
        ttl := 5 * time.Second
        if isTerminal(run.Phase) {
            ttl = 5 * time.Minute // completed runs don't change
        }
        cache.Set("traces:"+runID, spans, ttl)

        writeJSON(w, spans)
    }
}
```

### Session Management

```
POST /api/v1/auth/login     → create session, set cookie
POST /api/v1/auth/logout    → destroy session
GET  /api/v1/auth/me        → return current user info

Session cookie: HttpOnly, Secure, SameSite=Strict
CSRF token: returned in response header, required in request header
```

For local dev: sessions can be open (no login required). For production: pluggable auth provider (API key, OIDC).

### Static File Embedding

```go
//go:embed dist/*
var staticFS embed.FS

// Serves index.html for SPA routes (client-side routing)
func spaHandler(fs embed.FS) http.Handler {
    fileServer := http.FileServer(http.FS(fs))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try to serve the file directly
        if _, err := fs.Open(r.URL.Path); err == nil {
            fileServer.ServeHTTP(w, r)
            return
        }
        // Fall back to index.html for SPA routes
        r.URL.Path = "/"
        fileServer.ServeHTTP(w, r)
    })
}
```

### Deployment

```yaml
# Helm: replaces aot-web with aot-bff
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aot-bff
spec:
  containers:
    - name: bff
      image: aot-bff:{{ .Values.tag }}
      ports:
        - containerPort: 3000
      env:
        - name: APISERVER_URL
          value: "http://aot-apiserver:50055"
        - name: SESSION_SECRET
          value: {{ .Values.bff.sessionSecret }}
```

### Migration Path

1. Build BFF binary with all routes proxying to existing apiserver
2. Embed static web files via go:embed
3. Deploy alongside nginx initially (shadow mode)
4. Switch traffic from nginx to BFF
5. Remove nginx deployment and Dockerfile.web
6. Gradually add response shaping, caching, session management
