# Test layers

UNCWORKS has three test layers. Each one needs more infrastructure than the last.

## 1. Unit and contract tests, in `test/contract/` and `internal/`

These run without any external service. The contract tests check that the proto
definitions and the custom resources agree, that the protovalidate rules hold,
and that the type mappings are correct.

```bash
go test ./internal/... ./test/contract/... -count=1

# Or through Task:
task test:contract
```

They need no infrastructure. They run anywhere Go is installed.

Run the controller tests with `-p 1`. Parallel test binaries race to extract the
same envtest etcd binary, and the loser fails with `text file busy`.

## 2. Integration tests, in `test/integration/` and `test/temporal/`

These need Docker, which testcontainers drives, or a running Temporal dev server.

```bash
# Integration tests. These need Docker
go test -tags integration ./test/integration/... -v

# Temporal workflow tests. These need a Temporal dev server
go test ./test/temporal/... -v

# Or through Task:
task test:integration
```

They need a Docker daemon, and optionally `temporal server start-dev`.

## 3. End-to-end and smoke tests, in `e2e/`

These run against a live k0s cluster with every UNCWORKS service deployed. The
`e2e` build tag keeps them out of a normal `go test` run.

```bash
# Run every e2e test. This needs a running cluster and a port-forward
go test -tags e2e ./e2e/... -v -timeout 30m

# Or through Task:
task test:e2e
```

They need a k0s cluster with UNCWORKS deployed, Temporal, Ollama, a kubeconfig at
`uncworks/kubeconfig`, and an API server reachable at `$AOT_API_URL`, which
defaults to `http://localhost:50055`.

### Smoke tests

The `e2e/smoke_*.go` files are lightweight end-to-end checks.

- `smoke_pipeline` confirms the spec-driven pipeline completes.
- `smoke_files` confirms file listing works during a run, and that it exposes no
  internal directory.
- `smoke_shell` confirms the exec endpoint upgrades to a WebSocket.
- `smoke_traces` confirms the `tool_call` count in the structured log matches the
  tool spans in the trace.
- `smoke_validation` confirms the spec-driven validation step runs.
