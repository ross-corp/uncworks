## 1. Startup logger initialization

- [x] 1.1 In `cmd/apiserver/main.go`: add `initLogger()` function that calls `slog.SetDefault(...)` — JSON handler if `LOG_FORMAT=json` or non-TTY stdout, text handler otherwise; level from `LOG_LEVEL` env (default `info`)
- [x] 1.2 Call `initLogger()` as the very first statement in `cmd/apiserver/main.go`'s `main()`
- [x] 1.3 In `cmd/sidecar/main.go`: add the same `initLogger()` function and call it first in `main()`
- [x] 1.4 Migrate `cmd/apiserver/main.go`'s own `log.*` calls to `slog.*` (covers: K8s client init, Temporal connection, server startup, shutdown, API key warning, Temporal warning)
- [x] 1.5 Migrate `cmd/sidecar/main.go`'s own `log.*` calls to `slog.*` (covers: gateway start, shutdown, pi models config warnings)
- [x] 1.6 Remove `import "log"` from both cmd mains once all usages replaced; add `import "log/slog"`

## 2. `internal/server/traces.go`

- [x] 2.1 Replace `import "log"` with `import "log/slog"`
- [x] 2.2 Replace `log.Printf("failed to read spans file %s: %v", spansPath, err)` (×2) with `slog.Error("failed to read spans file", "file", spansPath, "err", err)`
- [x] 2.3 Replace `log.Printf("skipping malformed span line: %v", err)` with `slog.Debug("skipping malformed span line", "err", err)`

## 3. `internal/server/sse.go`

- [x] 3.1 Replace `import "log"` with `import "log/slog"`
- [x] 3.2 Replace `log.Printf("SSE: failed to marshal graph event: %v", err)` with `slog.Error("SSE: failed to marshal graph event", "err", err)`
- [x] 3.3 Replace `log.Printf("SSE: failed to marshal trace event: %v", err)` with `slog.Error("SSE: failed to marshal trace event", "err", err)`

## 4. `internal/server/classify.go`

- [x] 4.1 Replace `import "log"` with `import "log/slog"`
- [x] 4.2 Replace `log.Printf("WARNING: failed to list agent runs for classification: %v", err)` with `slog.Warn("failed to list agent runs for classification", "err", err)`
- [x] 4.3 Replace `log.Printf("ERROR: classify LLM call failed: %v", err)` with `slog.Error("classify LLM call failed", "err", err)`
- [x] 4.4 Replace `log.Printf("ERROR: failed to encode classify response: %v", err)` with `slog.Error("failed to encode classify response", "err", err)`

## 5. `internal/server/files.go`

- [x] 5.1 Replace `import "log"` with `import "log/slog"`
- [x] 5.2 Replace `log.Printf("exec ls in pod %s failed: %v, stderr: %s", podName, err, stderr)` with `slog.Error("exec ls in pod failed", "pod", podName, "err", err, "stderr", stderr)`
- [x] 5.3 Replace `log.Printf("failed to read directory %s: %v", diskPath, err)` with `slog.Error("failed to read directory", "path", diskPath, "err", err)`
- [x] 5.4 Replace `log.Printf("exec cat in pod %s failed: %v, stderr: %s", podName, err, stderr)` with `slog.Error("exec cat in pod failed", "pod", podName, "err", err, "stderr", stderr)`
- [x] 5.5 Replace `log.Printf("failed to read file %s: %v", diskPath, err)` with `slog.Error("failed to read file", "path", diskPath, "err", err)`
- [x] 5.6 Replace `log.Printf("Failed to stream logs for pod %s: %v", podName, err)` with `slog.Error("failed to stream logs for pod", "pod", podName, "err", err)`
- [x] 5.7 Replace `log.Printf("Failed to read logs for pod %s: %v", podName, err)` with `slog.Error("failed to read logs for pod", "pod", podName, "err", err)`
- [x] 5.8 Replace `log.Printf("failed to read log file %s: %v", logPath, err)` with `slog.Error("failed to read log file", "path", logPath, "err", err)`

## 6. `internal/server/grpc.go`

- [x] 6.1 Replace `import "log"` with `import "log/slog"`
- [x] 6.2 Replace all seven `log.Printf("WARNING: ...")` calls in the display-name LLM helper with `slog.Warn(...)` using appropriate structured fields (`"err"`, `"status"`, `"name"`)
- [x] 6.3 Ensure each `slog.Warn` call uses a concise message string (strip the `"WARNING: "` prefix) and puts variable data in key-value args

## 7. `internal/server/exec.go`

- [x] 7.1 Replace `import "log"` with `import "log/slog"`
- [x] 7.2 Replace `log.Printf("websocket upgrade failed: %v", err)` with `slog.Error("websocket upgrade failed", "err", err, "path", r.URL.Path)` (add `r` context if available)
- [x] 7.3 Replace `log.Printf("create clientset failed: %v", err)` with `slog.Error("create clientset failed", "err", err)`
- [x] 7.4 Replace `log.Printf("create SPDY executor failed: %v", err)` with `slog.Error("create SPDY executor failed", "err", err)`
- [x] 7.5 Replace `log.Printf("SPDY stream ended: %v", err)` with `slog.Debug("SPDY stream ended", "err", err)`
- [x] 7.6 Replace `log.Printf("websocket write error: %v", writeErr)` with `slog.Warn("websocket write error", "err", writeErr)`
- [x] 7.7 Replace `log.Printf("stdout read error: %v", err)` with `slog.Warn("stdout read error", "err", err)`
- [x] 7.8 Replace `log.Printf("websocket read error: %v", err)` with `slog.Warn("websocket read error", "err", err)`

## 8. `internal/server/debug.go`

- [x] 8.1 Replace `import "log"` with `import "log/slog"`
- [x] 8.2 Replace `log.Printf("failed to update deployment %s for debug: %v", deployName, err)` with `slog.Error("failed to update deployment for debug", "deployment", deployName, "err", err)`
- [x] 8.3 Replace `log.Printf("failed to update AgentRun %s status for debug: %v", runID, err)` with `slog.Error("failed to update AgentRun status for debug", "run", runID, "err", err)`
- [x] 8.4 Replace the two equivalent "stop debug" `log.Printf` calls with corresponding `slog.Error` calls

## 9. `internal/server/ci_autofix.go`

- [x] 9.1 Replace `import "log"` with `import "log/slog"`
- [x] 9.2 Replace `log.Printf("CI failure detected: %s on %s/%s (SHA %s)", ...)` with `slog.Info("CI failure detected", "check", payload.CheckRun.Name, "repo", repo, "branch", branch, "sha", sha[:8])`
- [x] 9.3 Replace `log.Printf("ERROR: failed to count fix attempts for %s: %v", branch, err)` with `slog.Error("failed to count fix attempts", "branch", branch, "err", err)`
- [x] 9.4 Replace `log.Printf("CI autofix: max retries (%d) reached for %s, posting comment", ci.MaxRetries, branch)` with `slog.Warn("CI autofix: max retries reached", "maxRetries", ci.MaxRetries, "branch", branch)`
- [x] 9.5 Replace `log.Printf("ERROR: failed to create fix run for %s: %v", branch, err)` with `slog.Error("failed to create fix run", "branch", branch, "err", err)`
- [x] 9.6 Replace `log.Printf("WARN: failed to fetch CI logs for %s: %v (proceeding without)", branch, err)` with `slog.Warn("failed to fetch CI logs, proceeding without", "branch", branch, "err", err)`
- [x] 9.7 Replace `log.Printf("CI autofix: created fix run %s for %s (attempt %d/%d)", run.Name, branch, attempt, ci.MaxRetries)` with `slog.Info("CI autofix: created fix run", "run", run.Name, "branch", branch, "attempt", attempt, "maxRetries", ci.MaxRetries)`
- [x] 9.8 Replace `log.Printf("WARN: failed to update CI status for %s: %v", run.Name, err)` with `slog.Warn("failed to update CI status", "run", run.Name, "err", err)`
- [x] 9.9 Replace `log.Printf("WARN: failed to resolve PR number for %s: %v", branch, err)` with `slog.Warn("failed to resolve PR number", "branch", branch, "err", err)`
- [x] 9.10 Replace `log.Printf("WARN: failed to post circuit breaker comment: %v", err)` with `slog.Warn("failed to post circuit breaker comment", "err", err)`
- [x] 9.11 Replace `log.Printf("CI autofix: posted circuit breaker comment on PR #%d", prNumber)` with `slog.Info("CI autofix: posted circuit breaker comment", "pr", prNumber)`

## 10. `internal/server/webhook.go`

- [x] 10.1 Replace `import "log"` with `import "log/slog"`
- [x] 10.2 Replace `log.Println("WARNING: GITHUB_WEBHOOK_SECRET not set ...")` with `slog.Warn("GITHUB_WEBHOOK_SECRET not set — webhook signature validation is disabled")`
- [x] 10.3 Replace `log.Printf("ERROR: CI autofix handler: %v", err)` with `slog.Error("CI autofix handler error", "err", err)`
- [x] 10.4 Replace `log.Printf("webhook: failed to fetch %s/%s@%s: %v", repo, path, payload.After, err)` with `slog.Error("webhook: failed to fetch file", "repo", repo, "path", path, "sha", payload.After, "err", err)`
- [x] 10.5 Replace `log.Printf("webhook: failed to create AgentRun for %s/%s: %v", repo, path, err)` with `slog.Error("webhook: failed to create AgentRun", "repo", repo, "path", path, "err", err)`
- [x] 10.6 Replace `log.Printf("webhook: created AgentRun %s for %s/%s", name, repo, path)` with `slog.Info("webhook: created AgentRun", "run", name, "repo", repo, "path", path)`

## 11. `internal/sidecar/gateway.go`

- [x] 11.1 Replace `import "log"` with `import "log/slog"` (verify no other log usages remain)
- [x] 11.2 Replace `log.Printf("WARNING: failed to create log dir %s: %v", agentLogDir, err)` with `slog.Warn("failed to create log dir", "dir", agentLogDir, "err", err)`
- [x] 11.3 Replace `log.Printf("WARNING: failed to create trace dir %s: %v", traceDir, err)` with `slog.Warn("failed to create trace dir", "dir", traceDir, "err", err)`
- [x] 11.4 Replace `log.Printf("Debug mode — waiting for connections")` with `slog.Info("debug mode — waiting for connections")`
- [x] 11.5 Replace `log.Printf("RPC Gateway listening on :%d", g.port)` with `slog.Info("RPC Gateway listening", "port", g.port)`
- [x] 11.6 Replace `log.Printf("Debug mode active — skipping agent launch for run %s", ...)` with `slog.Info("debug mode: skipping agent launch", "run", req.Msg.AgentRunId)`
- [x] 11.7 Replace `log.Printf("Stopping previous agent before starting new one for run %s", ...)` with `slog.Info("stopping previous agent before starting new one", "run", req.Msg.AgentRunId)`
- [x] 11.8 Replace `log.Printf("Loop detected: tool call %q repeated %d times — killing agent", sig, repeatCount)` with `slog.Warn("loop detected: tool call repeated — killing agent", "tool", sig, "count", repeatCount)`
- [x] 11.9 Replace `log.Printf("WARNING: dropped %s output (subscriber buffer full)", outputType)` with `slog.Warn("dropped output: subscriber buffer full", "type", outputType)`
- [x] 11.10 Replace `log.Printf("Scanner error on %s: %v", outputType, err)` with `slog.Error("scanner error", "type", outputType, "err", err)`
- [x] 11.11 Replace `log.Printf("Agent process hit rate limit (attempt %d/%d), retrying in %v: %s", ...)` with `slog.Warn("agent process hit rate limit, retrying", "attempt", attempt, "maxAttempts", maxAttempts, "delay", delay)`
- [x] 11.12 Replace `log.Printf("Failed to restart agent process (attempt %d): %v", attempt, startErr)` with `slog.Error("failed to restart agent process", "attempt", attempt, "err", startErr)`

## 12. Verification

- [x] 12.1 Run `go build ./cmd/apiserver/...` — must pass with no errors
- [x] 12.2 Run `go build ./cmd/sidecar/...` — must pass with no errors
- [x] 12.3 Run `go vet ./internal/server/... ./internal/sidecar/...` — must pass with no warnings
- [x] 12.4 Run `go test ./internal/server/... ./internal/sidecar/...` — all tests must pass
- [x] 12.5 Confirm no remaining `import "log"` in `internal/server/` or `internal/sidecar/` (grep check)
- [ ] 12.6 Start server locally with `LOG_FORMAT=text` and confirm human-readable output appears for a test request
- [ ] 12.7 Start server locally with `LOG_FORMAT=json` and confirm JSON-formatted log lines appear
