## Why

Five concurrency and resource-safety bugs were identified in a code quality audit: a missing WaitGroup registration that allows data loss on WebSocket teardown, a mutex that is not held during field reads causing a TOCTOU race on agent process state, silently discarded Kubernetes status-update errors that prevent automatic requeue on transient failures, unbounded in-memory maps in BFF middleware that leak memory and create a DoS surface, and a `context.Background()` used inside a timer callback that ignores server shutdown signals. These are correctness bugs with real operational impact and should be fixed before further feature development compounds the exposure.

## What Changes

- **`internal/server/exec.go`**: Register the WebSocket reader goroutine in the WaitGroup before `wg.Wait()` so teardown cannot race the reader.
- **`internal/sidecar/gateway.go`**: Ensure all reads and writes of `AgentProcess.state` and `AgentProcess.exitError` are performed while holding `mu`.
- **`internal/controller/agentrun_controller.go`, `chain_controller.go`, `schedule_controller.go`**: Replace `_ = r.Status().Update(...)` with proper error handling that returns the error (triggering controller-runtime requeue on transient failures).
- **`internal/bff/middleware.go`**: Replace the three unbounded maps (session, CSRF token, rate-limit IP buckets) with TTL-evicting structures to eliminate the memory leak and reduce the DoS surface on the IP bucket.
- **`internal/server/ci_autofix.go`**: Thread the server's context into the `time.AfterFunc` callback instead of using `context.Background()`.

## Capabilities

### New Capabilities

<!-- None — this is a targeted correctness fix with no new user-visible capabilities. -->

### Modified Capabilities

<!-- No spec-level requirement changes. All fixes are implementation-only corrections
     to existing behaviour that was already specified or implied by existing specs. -->

## Impact

- **`internal/server/exec.go`**: goroutine lifecycle and WebSocket handling
- **`internal/sidecar/gateway.go`**: agent process state management
- **`internal/controller/`**: three controller files — requeue behaviour for status updates
- **`internal/bff/middleware.go`**: session, CSRF, and rate-limit middleware
- **`internal/server/ci_autofix.go`**: CI autofix runner shutdown path
- No API surface changes, no breaking changes, no dependency additions (a small TTL-map helper or `sync.Map`-with-expiry pattern will be used inline or via an existing stdlib-compatible approach)
