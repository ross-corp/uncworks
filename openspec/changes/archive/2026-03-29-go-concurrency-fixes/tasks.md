## 1. Fix WaitGroup gap in WebSocket exec handler

- [x] 1.1 In `internal/server/exec.go`, add `wg.Add(1)` immediately before the `go func()` at line 241 (the WS reader goroutine)
- [x] 1.2 Add `defer wg.Done()` as the first statement inside that goroutine
- [x] 1.3 Verify with `go test -race ./internal/server/...` that no race is reported on exec teardown

## 2. Fix mutex gap on AgentProcess fields

- [x] 2.1 In `internal/sidecar/gateway.go`, add `State() agentv1.AgentProcessState` accessor method on `*AgentProcess` that locks and unlocks `mu`
- [x] 2.2 Add `ExitError() string` accessor method on `*AgentProcess` that locks and unlocks `mu`
- [x] 2.3 Replace all bare reads of `g.process.state` and `g.process.exitError` outside of already-locked sections with the new accessors
- [x] 2.4 Ensure all writes to `state` and `exitError` also occur under `mu`
- [x] 2.5 Run `go test -race ./internal/sidecar/...` and confirm no data race

## 3. Propagate controller status-update errors

- [x] 3.1 In `internal/controller/agentrun_controller.go`, replace each `_ = r.Status().Update(ctx, &obj)` with `if err := r.Status().Update(ctx, &obj); err != nil { return ctrl.Result{}, err }`
- [x] 3.2 Apply the same change in `internal/controller/chain_controller.go`
- [x] 3.3 Apply the same change in `internal/controller/schedule_controller.go`
- [x] 3.4 Run `go build ./internal/controller/...` and ensure no compilation errors

## 4. Add TTL eviction to BFF middleware maps

- [x] 4.1 In `internal/bff/middleware.go`, add a background sweep goroutine inside `SessionMiddleware` that runs every 5 minutes and deletes session entries with `createdAt` older than 24 hours
- [x] 4.2 Add a background sweep goroutine inside `CSRFMiddleware` that runs every 5 minutes and deletes token entries whose corresponding session entry is no longer present or older than 24 hours (key off age by storing a `createdAt` alongside each CSRF token)
- [x] 4.3 Add a background sweep goroutine inside `RateLimitMiddleware` that runs every 5 minutes and deletes `ipBucket` entries from the `sync.Map` where `lastCheck` is older than 10 minutes
- [x] 4.4 Run `go test ./internal/bff/...` and verify existing middleware tests still pass

## 5. Thread server-lifetime context into CIAutofix

- [x] 5.1 Add a `ctx context.Context` field to the `CIAutofix` struct in `internal/server/ci_autofix.go`
- [x] 5.2 Update the `CIAutofix` constructor or initialisation site to accept and store a server-lifetime context
- [x] 5.3 In the `time.AfterFunc` callback, replace `context.Background()` with `ci.ctx`
- [x] 5.4 Add a guard at the start of the callback body: `if ci.ctx.Err() != nil { return }` to skip execution on shutdown
- [x] 5.5 Run `go build ./internal/server/...` and ensure no compilation errors

## 6. Verification

- [x] 6.1 Run `go test -race ./internal/...` and confirm zero race detector warnings
- [x] 6.2 Run `go vet ./internal/...` and confirm no new issues
- [x] 6.3 Run the full unit test suite and confirm no regressions
