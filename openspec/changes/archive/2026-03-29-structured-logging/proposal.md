## Why

The Go backend uses three different logging styles inconsistently. `internal/server/` (11 files) and `internal/sidecar/` use `log.Printf(...)` from the stdlib `log` package, including string-prefixed severity hacks like `log.Printf("WARNING: ...")` and `log.Printf("ERROR: ...")`. These produce unstructured text with no machine-readable fields, no run IDs, no paths, and no consistent severity levels. Downstream log aggregation (e.g., in k8s via structured JSON) cannot filter or correlate these messages. Go 1.21 introduced `log/slog` — a structured, leveled logger — and this project already runs Go 1.25. The project already uses `slog` in `ci/dagger.gen.go`.

## What Changes

- Replace all `log.Printf` / `log.Println` / `log.Fatalf` calls in `internal/server/` and `internal/sidecar/` with `slog` equivalents at the correct level
- Map `WARNING:` / `WARN:` prefixed messages to `slog.Warn`
- Map `ERROR:` prefixed messages to `slog.Error`
- Map informational messages to `slog.Info`
- Map debug/trace messages to `slog.Debug`
- Add structured key-value fields: `"err"` for errors, `"path"` for URL paths, `"run"` for run IDs, `"pod"` for pod names, `"branch"` for branch names, etc.
- Add a `slog.SetDefault(...)` call in `cmd/apiserver/main.go` and `cmd/sidecar/main.go` that sets JSON handler in production (when `LOG_FORMAT=json` or not TTY) and text handler in dev

## Capabilities

### Modified Capabilities
- `internal/server/`: All handler files switch from `import "log"` to `import "log/slog"`
- `internal/sidecar/gateway.go`: Same migration
- `cmd/apiserver/main.go`: Adds `slog.SetDefault(...)` at startup
- `cmd/sidecar/main.go`: Adds `slog.SetDefault(...)` at startup

### New Capabilities
- Structured log output: every error log includes `"err"` as a structured field parseable by log collectors
- Severity-aware filtering: operators can set `SLOG_LEVEL=warn` to reduce noise
- Run-ID correlation: key log sites in ci_autofix, webhook, sidecar include `"run"` field

## Non-Goals

- `internal/controller/` — controller-runtime's `logr`-based logger is idiomatic there; do NOT change it
- No new log aggregation infrastructure
- No distributed tracing integration (separate concern)
- No changes to test files

## Impact

- 11 files in `internal/server/`: traces.go, sse.go, classify.go, files.go, grpc.go, exec.go, debug.go, ci_autofix.go, webhook.go
- 1 file in `internal/sidecar/`: gateway.go
- 2 entrypoints: `cmd/apiserver/main.go`, `cmd/sidecar/main.go`
- No API changes, no proto changes, no frontend changes
- No behavior change for callers — purely logging internals
