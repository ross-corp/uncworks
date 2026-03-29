## Context

Five discrete correctness bugs were found across the server, sidecar, controller, BFF middleware, and CI-autofix packages. Each is independently fixable. None require new dependencies or architectural changes. The fixes are surgical: they touch only the erroneous lines, not the surrounding logic.

Current state of each bug:

1. **`exec.go` WaitGroup gap** — Two goroutines (SPDY stream, SPDY→WS writer) are registered with `wg.Add(1)` before launch. The third goroutine (WS reader, line 241) is launched with a bare `go func()` and never added to the WaitGroup. `wg.Wait()` at line 291 therefore returns before the reader exits, allowing the function's deferred cleanup and the caller's stack to be freed while the goroutine may still be calling `wsConn.ReadMessage`.

2. **`gateway.go` mutex gap** — `AgentProcess.mu` is defined as a `sync.Mutex`. `StartAgent` holds `g.mu` (the Gateway-level lock) when it reads `g.process.state` at line 169. But other callers (e.g. `GetAgentStatus`, stream goroutines) read `g.process.state` and `g.process.exitError` without holding either lock, producing a data race.

3. **Controller status-update errors** — `r.Status().Update(ctx, &obj)` can return a conflict or transient error. Assigning the result to `_` discards it. Controller-runtime interprets a nil return as success and does not requeue. The correct pattern is to return the error so controller-runtime applies its exponential-backoff retry.

4. **BFF unbounded maps** — `SessionMiddleware` uses `map[string]sessionEntry`, `CSRFMiddleware` uses `map[string]string`. Both grow without bound. `RateLimitMiddleware` uses `sync.Map` (also unbounded). The `sessionEntry` struct carries a `createdAt` field but it is never read for eviction. The session and CSRF maps are the primary leak vectors; the IP bucket map in `sync.Map` is a secondary DoS amplifier.

5. **`ci_autofix.go` context** — `CIAutofix` receives no server-lifetime context. The `time.AfterFunc` callback captures none and creates a `context.Background()`. If the server shuts down during the 30-second delay, the callback still fires and attempts to create a Kubernetes object against a client that may be torn down.

## Goals / Non-Goals

**Goals:**
- Eliminate the WaitGroup gap so `HandleExec` does not return while the WS reader goroutine is live.
- Lock all accesses to `AgentProcess.state` and `AgentProcess.exitError` so the race detector no longer fires.
- Return status-update errors in all three controllers so transient failures trigger automatic requeue.
- Add TTL eviction to the session and CSRF maps so memory is bounded on long-running servers; add a max-entries cap to the IP bucket map to limit DoS surface.
- Thread a server-lifetime context into `CIAutofix` so the timer callback respects shutdown.

**Non-Goals:**
- Replacing the in-memory session store with a persistent or distributed store.
- Changing the CSRF or session token formats or lifetimes.
- Adding metrics or structured logging.
- Any refactoring of surrounding code not directly related to these bugs.

## Decisions

### 1. WaitGroup registration for WS reader (`exec.go`)

Add `wg.Add(1)` immediately before the `go func()` at line 241 and add `defer wg.Done()` as the first statement inside the goroutine. This mirrors the existing pattern for the other two goroutines. No other changes are needed.

*Alternative considered*: use a separate done channel instead of the WaitGroup. Rejected — the existing WaitGroup is already the correct mechanism; adding a channel would be gratuitous complexity.

### 2. Locking `AgentProcess` fields (`gateway.go`)

Add accessor methods `State() agentv1.AgentProcessState` and `ExitError() string` on `AgentProcess` that lock `mu` internally. Replace all bare field reads/writes outside of already-locked sections with these accessors.

*Alternative considered*: promote the struct fields to be owned entirely by the Gateway-level `g.mu`. Rejected — `AgentProcess` outlives individual RPC calls and is accessed from stream goroutines that don't hold `g.mu`, so a per-process mutex is the right scope.

### 3. Status update error handling (controllers)

Replace `_ = r.Status().Update(ctx, &obj)` with `if err := r.Status().Update(ctx, &obj); err != nil { return ctrl.Result{}, err }`. Controller-runtime will then apply its standard exponential-backoff retry. No additional retry logic is needed.

### 4. TTL eviction for BFF maps (`middleware.go`)

Add a background goroutine inside each closure-based middleware that periodically sweeps the map and deletes entries older than a configurable TTL (default 24 h for sessions, 24 h for CSRF tokens). For the IP bucket `sync.Map`, add a sweep that deletes entries where `lastCheck` is older than 10 minutes (inactive IPs). The sweep interval will be 5 minutes for all three.

This approach uses only stdlib (`sync`, `time`) and keeps the changes self-contained within the existing closure. No new types or packages are introduced.

*Alternative considered*: replace maps with an LRU cache (e.g. `golang.org/x/exp/cache`). Rejected — introduces an external dependency for a change scoped to correctness. The sweep approach is sufficient and keeps the diff minimal.

### 5. Server-lifetime context in `CIAutofix` (`ci_autofix.go`)

Add a `ctx context.Context` field to `CIAutofix`. Populate it at construction time (the caller already has a server-lifetime context). In the `time.AfterFunc` callback, use `ci.ctx` instead of `context.Background()`. Gate the callback's work on `ci.ctx.Err() == nil`.

*Alternative considered*: pass context as a parameter to `scheduleFixRun`. Rejected — the callback is a closure already capturing `ci`; storing the context on the struct is idiomatic Go for long-lived objects that need a shutdown signal.

## Risks / Trade-offs

- **TTL eviction sweep goroutine leaks** — The sweep goroutine inside each middleware closure will run for the lifetime of the `http.Server`. If the server is restarted without full process exit (unusual), these goroutines remain. Mitigation: use `time.NewTicker` with a `select` on a stop channel tied to server shutdown (the same `ci.ctx` pattern from fix 5 could be applied here if needed; for now, process lifetime is sufficient).
- **Controller requeue storms** — Returning status errors will cause controller-runtime to requeue more aggressively on API server degradation. This is the correct behaviour but could increase noise in degraded clusters. Mitigation: none required — this is the documented controller-runtime contract.
- **`AgentProcess` accessor method churn** — Any future direct field access on `state` or `exitError` will bypass the lock. Mitigation: the fields can be unexported (they already are lowercase), so only methods on the same package type can access them — the access points are finite and auditable.

## Migration Plan

No data migrations. No API changes. No configuration changes. Deploy as a normal binary update. All fixes are backward-compatible.

Rollback: revert the commit. No state is stored externally by these components.
