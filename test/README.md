# Test Layers

AOT uses three test layers, each with increasing infrastructure requirements.

## 1. Unit / Contract Tests (`test/contract/`, `internal/`)

Fast tests that run without external services. Contract tests verify proto-to-CRD
boundary alignment, protovalidate rules, and type mappings.

```bash
# From uncworks/
go test ./internal/... ./test/contract/... -count=1

# Or via Taskfile (uncworks or aot-local):
task test:contract
```

**Infrastructure:** None. Runs anywhere with Go installed.

## 2. Integration Tests (`test/integration/`, `test/temporal/`)

Tests that require Docker (for testcontainers) or a running Temporal dev server.

```bash
# Integration tests (Docker required):
go test -tags integration ./test/integration/... -v

# Temporal workflow tests (temporal dev server required):
go test ./test/temporal/... -v

# Or via Taskfile:
task test:integration
```

**Infrastructure:** Docker daemon, optionally `temporal server start-dev`.

## 3. E2E / Smoke Tests (`e2e/`)

Full system tests that run against a live k0s cluster with all AOT services deployed.
Guarded by the `e2e` build tag so they are excluded from normal `go test` runs.

```bash
# Run all e2e tests (requires running cluster + port-forward):
go test -tags e2e ./e2e/... -v -timeout 30m

# Or via Taskfile:
task test:e2e
```

**Infrastructure:** k0s cluster with AOT deployed, Temporal, Ollama, kubeconfig
at `uncworks/kubeconfig`, API server reachable at `$AOT_API_URL` (default
`http://localhost:50055`).

### Smoke tests

The `e2e/smoke_*.go` files are lightweight end-to-end checks covering:

- **smoke_pipeline** -- spec-driven pipeline completes successfully
- **smoke_files** -- file listing during a running agent (no internal dirs exposed)
- **smoke_shell** -- WebSocket upgrade to the exec endpoint
- **smoke_traces** -- structured log tool_call counts match trace tool spans
- **smoke_validation** -- spec-driven validation step runs
