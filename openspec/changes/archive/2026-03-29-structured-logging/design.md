## Context

The uncworks backend is a Go monorepo (Go 1.25) with three main server processes: `cmd/apiserver` (HTTP + Connect-RPC), `cmd/sidecar` (gRPC gateway inside agent pods), and `cmd/controller` (Kubernetes controller using controller-runtime). The controller already uses `logr` via controller-runtime — that pattern is idiomatic and stays untouched. The `internal/server/` and `internal/sidecar/` packages use raw `log.Printf` throughout. The project's Go version (1.25) has had `log/slog` since 1.21. No third-party structured logging library is needed.

## Goals / Non-Goals

**Goals:**
- Replace all `log.Printf`/`log.Println`/`log.Fatalf` in `internal/server/` and `internal/sidecar/` with `slog` calls at the correct level
- Include structured fields (`"err"`, `"path"`, `"run"`, `"pod"`, `"branch"`) at key call sites
- Configure a global `slog.Default()` logger at server startup: JSON in production, text in dev
- Remove string-prefixed severity markers (`"WARNING: "`, `"ERROR: "`, `"WARN: "`)

**Non-Goals:**
- Leave `internal/controller/` entirely untouched
- No new logging library dependencies
- No changes to test files (test log output style is irrelevant to operators)
- No distributed tracing

## Decisions

**1. `log/slog` (stdlib) over zerolog/zap/logrus**
Go 1.21+ stdlib `slog` is already used in `ci/dagger.gen.go`. Adding zerolog or zap would introduce a dependency for no practical gain at this codebase's scale. `slog` is the correct idiomatic choice for new Go code.

**2. Global default logger, not injected logger**
HTTP handlers in `internal/server/` are constructed with various typed handler structs (e.g., `FileHandler`, `ExecHandler`). Injecting a `*slog.Logger` field into each struct is a larger refactor than justified by this change. Instead, all call sites use `slog.Error(...)`, `slog.Warn(...)`, `slog.Info(...)` which route through the default logger set at startup. If a handler already has request context available, `slog.Default().With(...)` can be used for per-request fields; this is optional and can be done incrementally.

**3. Logger setup at startup**
Both `cmd/apiserver/main.go` and `cmd/sidecar/main.go` call `slog.SetDefault(...)` before any handler registration. Format selection:
- If `LOG_FORMAT=json` env var is set, or if stdout is not a TTY (`os.Stdout`), use `slog.NewJSONHandler`
- Otherwise use `slog.NewTextHandler` (human-readable for local dev)
- Level controlled by `LOG_LEVEL` env var (default: `info`; accepts `debug`, `info`, `warn`, `error`)

**4. Severity mapping**
| Old pattern | New call |
|---|---|
| `log.Printf("WARNING: ...")` | `slog.Warn("...", fields...)` |
| `log.Printf("WARN: ...")` | `slog.Warn("...", fields...)` |
| `log.Printf("ERROR: ...")` | `slog.Error("...", fields...)` |
| `log.Fatalf(...)` | `slog.Error("...", fields...); os.Exit(1)` |
| `log.Printf("...")` (informational) | `slog.Info("...", fields...)` |
| `log.Printf("...")` (loop/stream debug noise) | `slog.Debug("...", fields...)` |

`log.Fatalf` in `cmd/apiserver/main.go` maps to `slog.Error` + `os.Exit(1)` since the logger must be initialized before it can be used there; alternatively the existing `log.Fatal` in main (before slog init) can stay as-is — but any `log.Fatal` after `slog.SetDefault` should be migrated.

**5. Structured fields convention**
- Error field always: `"err", err`
- HTTP path: `"path", r.URL.Path`
- Run/resource ID: `"run", runID` or `"run", run.Name`
- Pod: `"pod", podName`
- Branch: `"branch", branch`
- Repo: `"repo", repo`
- File path: `"file", path`
- Attempt: `"attempt", n`

**6. `log.Fatalf` in main**
`cmd/apiserver/main.go` calls `log.Fatalf` in `init`-time setup (K8s client creation). These occur before `slog.SetDefault`, so they can remain as `log.Fatalf` OR be changed to `slog.Error + os.Exit(1)` after moving `slog.SetDefault` to be the very first call in `main()`. The latter is cleaner and is the preferred approach.

## Risks / Trade-offs

- **No behavior risk**: pure logging change, no business logic touched
- **Test output may change**: tests that capture stderr and assert on log output patterns may need updating — but there are no such tests in this codebase based on a grep of test files
- **`log.Fatalf` timing**: `slog.SetDefault` must be called before any `slog.*` usage; placing it as the first statement in `main()` ensures this

## Migration Plan

1. Add `slog.SetDefault(...)` helper function to both cmd mains — call it as first thing in `main()`
2. Migrate `cmd/apiserver/main.go` own `log.*` calls
3. Migrate `cmd/sidecar/main.go` own `log.*` calls
4. Migrate `internal/server/` file by file (11 files)
5. Migrate `internal/sidecar/gateway.go`
6. Verify: `go build ./...` passes, `go vet ./...` passes

## Open Questions

- Should per-request `slog.Logger` with a request-ID field be injected into handlers via `context.Context`? (Recommended: not in this change — keep it to global default; per-request context enrichment can be a follow-on)
- Should `LOG_FORMAT` auto-detect TTY or require explicit env var? (Recommended: auto-detect via `term.IsTerminal(int(os.Stdout.Fd()))` — requires `golang.org/x/term` which is already a transitive dep)
