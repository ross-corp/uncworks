# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

AOT (Agent Orchestration Tool) is a Kubernetes-native platform for running AI coding agents. Users submit a prompt + git repo, and AOT provisions an isolated workspace, runs the agent, and streams results in real time. The core abstraction is the `AgentRun` CRD.

## Commands

All commands use [Task](https://taskfile.dev/) (see `Taskfile.yml`). Enter the dev environment first with `devbox shell`.

### Build
```bash
task build          # all 5 Go binaries to ./bin/
task build:web      # web dashboard (Vite)
task proto:gen      # regenerate Go code from .proto files
```

### Test
```bash
task test           # all tests (Go + web + extension + TUI)
task test:go        # Go unit + integration tests
task test:e2e       # E2E tests (requires running k0s cluster)
task test:web       # Playwright tests (web dashboard)
task test:extension # pi-aot-extension tests
task test:tui       # TUI renderer tests
task test:shared    # @aot/shared package tests
```

Single Go test: `go test ./internal/server/... -run TestCreateAgentRun -count=1`

Controller tests require envtest (auto-resolved via `internal/testutil.EnsureEnvtestAssets()`). If `KUBEBUILDER_ASSETS` is not set, install with: `go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest`

### Lint
```bash
task lint           # golangci-lint + TypeScript type checks
```

Linting uses [golangci-lint](https://golangci-lint.run/) v2 (config: `.golangci.yml`). Enabled linters: govet, errcheck, staticcheck, unused, ineffassign, gocritic, misspell. Formatter: gofmt. Generated code in `gen/go/` is excluded.

### Infrastructure
```bash
task k0s:setup      # initialize local k0s cluster (requires sudo)
task k0s:teardown   # tear down cluster
task k0s:crd        # apply AgentRun CRD to cluster
task dev:web        # start Vite dev server for web dashboard
```

## Architecture

Two gRPC APIs define all communication:

- **`proto/api.proto`** — Client API (`AOTService` on `:50051`): CreateAgentRun, GetAgentRun, ListAgentRuns, WatchAgentRun (server-streaming), CancelAgentRun, SendHumanInput
- **`proto/agent.proto`** — Sidecar API (`AgentSidecarService` on `:50052`): StartAgent, StreamOutput, SendInput, GetStatus, StopAgent. Plus `AgentNotificationService` for sidecar→control-plane async events.

Generated code lives in `gen/go/`. Proto generation: `task proto:gen` (runs `hack/proto-gen.sh`).

### Five Go binaries (`cmd/`)

| Binary | Role |
|--------|------|
| `apiserver` | gRPC server + WebSocket hub (`:50051` + `:8080`) |
| `controller` | K8s controller — watches AgentRun CRDs, creates pods |
| `hydration` | Init-container — git clone + devbox setup |
| `sidecar` | RPC Gateway — bridges agent process to control plane |
| `aot` | CLI tool (`aot open` finds/opens AOT worktrees) |

### Key Go packages (`internal/`)

- **`server/`** — gRPC `AOTService` implementation + WebSocket event hub
- **`controller/`** — K8s reconciler for AgentRun CRD. `multi_agent.go` handles `spawn_junior` child AgentRuns
- **`brain/`** — PostgreSQL state store (pgx). Agent state, metadata, priority queue
- **`hydration/`** — Git bare clone → worktree creation → devbox setup
- **`sidecar/`** — RPC Gateway running inside agent pods
- **`cli/`** — `aot open` implementation
- **`testutil/`** — Shared test helpers (auto-resolves envtest assets)

### CRD types (`api/v1alpha1/`)

`AgentRun` with `AgentRunSpec` (backend, repoURL, branch, prompt, devboxConfig, ttlSeconds, envVars, image) and `AgentRunStatus` (phase, message, podName, traceID). Phases: Pending → Running → Succeeded/Failed/Cancelled. WaitingForInput for HITL.

### TypeScript packages (`packages/`)

- **`@aot/shared`** — gRPC client wrapper + reactive agent state store
- **`@aot/pi-extension`** — Agent harness extension: `ask_human` tool (HITL), `spawn_junior` tool (multi-agent), OTel tracing
- **`@aot/tui`** — SolidJS TUI with ANSI renderer

### Web dashboard (`web/`)

SolidJS + Vite. Connects to API server via WebSocket for real-time updates.

## Data Flow

1. Client calls `CreateAgentRun` via gRPC (or `kubectl apply`)
2. Controller sees new AgentRun CRD, creates Pod (init-container + agent + sidecar)
3. Init-container clones repo, creates worktree, runs `devbox install`
4. Agent container executes prompt in workspace
5. Sidecar streams output back to control plane via gRPC
6. Clients watch via `WatchAgentRun` (gRPC stream) or WebSocket

## Conventions

- **Diagrams**: Always use Mermaid in markdown. Never use ASCII box-drawing diagrams.
- **Testing**: Use Ginkgo/Gomega for BDD-style tests. Controller tests use envtest. gRPC tests use real listeners on `127.0.0.1:0`.
- **Go module**: `github.com/uncworks/aot`
- **CRD group**: `aot.uncworks.io/v1alpha1`
- **Labels**: `aot.uncworks.io/parent`, `aot.uncworks.io/role`, `aot.uncworks.io/managed`
- **Ports**: API server `:50051` (gRPC), `:8080` (HTTP/WS). Sidecar `:50052`.

## OpenSpec

The project uses OpenSpec for change management. Active changes live in `openspec/changes/<name>/` with artifacts: proposal.md, design.md, specs/, tasks.md. Use `/opsx:explore` to investigate, `/opsx:propose` to create changes, `/opsx:apply` to implement.
