# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

UNCWORKS is a Kubernetes-native platform for running AI coding agents. Users submit a prompt + git repo, and UNCWORKS provisions an isolated workspace, runs the agent, and streams results in real time. The core abstraction is the `AgentRun` CRD.

## Commands

All commands use [Task](https://taskfile.dev/) (see `Taskfile.yml`). Enter the dev environment first with `devbox shell`.

### Build
```bash
task build          # all Go binaries to ./bin/
task build:web      # web dashboard (Vite)
task build:app      # native macOS app (Wails v2, macOS only)
task build:uncworks # cross-compile uncworks CLI (linux/darwin amd64+arm64)
task proto:gen      # regenerate Go + TypeScript code from .proto files
task proto:lint     # lint protobuf definitions
task proto:breaking # check for breaking proto changes vs main
```

### Test
```bash
task test              # all tests in parallel (Go + web + extension + layer2)
task test:go           # Go unit + integration tests (api/... internal/...)
task test:unit         # Go unit only — fast, no Docker
task test:contract     # ConnectRPC + protovalidate contract tests
task test:temporal     # Temporal workflow tests
task test:layer2       # Layer 2 pipeline tests (LLM stubbed, no cluster)
task test:regression   # Regression suite — gates releases and PRs to main
task test:web          # Playwright tests for web dashboard
task test:extension    # pi-aot-extension TypeScript tests
task test:shared       # @aot/shared TypeScript tests
task test:e2e          # Go E2E tests (requires running cluster)
task test:e2e:full     # setup soft-serve → E2E → Playwright → teardown
```

Single Go test: `go test ./internal/server/... -run TestCreateAgentRun -count=1`

Controller tests require envtest (auto-resolved via `internal/testutil.EnsureEnvtestAssets()`). If `KUBEBUILDER_ASSETS` is not set, install with: `go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest`

### Lint
```bash
task lint           # golangci-lint + TypeScript type checks (all packages)
```

Linting uses [golangci-lint](https://golangci-lint.run/) v2 (config: `.golangci.yml`). Enabled linters: govet, errcheck, staticcheck, unused, ineffassign, gocritic, misspell. Formatter: gofmt. Generated code in `gen/go/` is excluded.

### Local Dev Cluster (colima-uncworks)
```bash
task dev:web          # start Vite HMR dev server for web dashboard
task dev:images       # build all images into colima-uncworks k8s.io namespace
task dev:deploy       # build images + kubectl rollout restart + status
task dev:install      # install all Go + npm workspace dependencies
task dev:hooks:install # install git hooks via lefthook
```

### Kubernetes / Cluster Operations
```bash
task k8s:crd          # apply AgentRun CRD to cluster
task k8s:deps         # deploy all infra deps (CRDs, storage, Ollama, LiteLLM, soft-serve)
task k8s:images       # build images via docker + import into k0s (sudo required)
task k8s:deploy:all   # build web + import images + rollout all deployments
task cluster:setup    # install systemd units + build/import images + start all services
task cluster:status   # show health of all UNCWORKS services and ports
task cluster:teardown # stop all UNCWORKS services and remove systemd units
task cluster:logs     # combined logs from all UNCWORKS services
task cluster:temporal:dev  # start Temporal dev server (SQLite, no external deps)
```

## Architecture

Two gRPC APIs define all communication:

- **`proto/api.proto`** — Client API (`AOTService` on `:50055`): CreateAgentRun, GetAgentRun, ListAgentRuns, WatchAgentRun (server-streaming), CancelAgentRun, SendHumanInput
- **`proto/agent.proto`** — Sidecar API (`AgentSidecarService` on `:50052`): StartAgent, StreamOutput, SendInput, GetStatus, StopAgent. Plus `AgentNotificationService` for sidecar→control-plane async events.

Generated code lives in `gen/go/`. Proto generation: `task proto:gen` (runs `buf generate`).

### Go binaries (`cmd/`)

| Binary | Role |
|--------|------|
| `apiserver` | ConnectRPC server + REST endpoints (`:50055`) |
| `controller` | K8s controller — watches AgentRun CRDs, creates pods |
| `hydration` | Init-container — git clone + devbox setup |
| `sidecar` | RPC Gateway — bridges agent process to control plane (`:50052`) |
| `temporal-worker` | Temporal activity worker — executes pipeline stages |
| `uncworks` | End-user CLI (`uncworks setup`, `uncworks open`, `uncworks tui`) |
| `aot` | Internal CLI — workspace tooling (`aot open`) |
| `bff` | BFF server for the macOS desktop app |
| `uncworks-app` | macOS desktop app (Wails v2) |

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

### Workspace Layout

Each agent run gets a persistent workspace on a PVC mounted at `/workspace`:

```
/workspace/
├── src/                    # Cloned repositories
├── .aot/
│   ├── logs/agent.log     # Agent stdout/stderr
│   ├── traces/spans.jsonl # Execution trace spans
│   └── metadata.json      # Run metadata snapshot
├── .devcontainer/
│   └── devcontainer.json  # VS Code Remote config
├── uncspace.yaml          # Workspace manifest
└── devbox.json            # Composed devbox config
```

- `src/` contains git clones of the specified repositories
- `.aot/logs/agent.log` is tee'd from agent stdout/stderr by the sidecar
- `.aot/traces/spans.jsonl` records tool calls, LLM interactions, and git diffs as JSONL
- `.aot/metadata.json` snapshots the run spec (repos, prompt, model tier, etc.)
- `.devcontainer/devcontainer.json` enables VS Code Remote attachment
- After completion (Deployment replicas=0), these files remain on the PVC and are served by the API

### Web dashboard (`web/`)

SolidJS + Vite. Connects to API server via WebSocket for real-time updates.

## Data Flow

1. Client calls `CreateAgentRun` via gRPC (or `kubectl apply`)
2. Controller sees new AgentRun CRD, creates Pod (init-container + agent + sidecar)
3. Init-container clones repo, creates worktree, runs `devbox install`
4. Agent container executes prompt in workspace
5. Sidecar streams output back to control plane via gRPC
6. Clients watch via `WatchAgentRun` (gRPC stream) or WebSocket

## Git Hooks & Releases

Git hooks are managed by [Lefthook](https://lefthook.dev/) (config: `lefthook.yml`). Hooks install automatically on `devbox shell` entry, or manually via `task dev:hooks:install`.

- **pre-commit**: gofmt, golangci-lint (new changes only), buf lint, TypeScript type checks
- **commit-msg**: Enforces [Conventional Commits](https://www.conventionalcommits.org/) via commitlint
- **pre-push**: Go tests, buf breaking change detection

Releases use [Release Please](https://github.com/googleapis/release-please). Conventional commit messages on `main` automatically generate changelogs and version bumps. The CI workflow (`ci.yml`) runs Release Please after each merge to `main`. Every passing push to `main` also auto-tags a pre-release: `vX.Y.Z-pre.YYYYMMDD.sha7`.

## Conventions

- **Diagrams**: Always use Mermaid in markdown. Never use ASCII box-drawing diagrams.
- **Testing**: Use Ginkgo/Gomega for BDD-style tests. Controller tests use envtest. gRPC tests use real listeners on `127.0.0.1:0`.
- **Go module**: `github.com/uncworks/aot`
- **CRD group**: `aot.uncworks.io/v1alpha1`
- **Labels**: `aot.uncworks.io/parent`, `aot.uncworks.io/role`, `aot.uncworks.io/managed`
- **Ports**: API server `:50055` (ConnectRPC + REST). Sidecar `:50052`.
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`, `chore:`).

## OpenSpec

OpenSpec is the change management system for this repo. It enforces a spec-driven workflow: propose → design → spec → implement → archive.

### Directory layout

```
openspec/
├── config.yaml          # schema and project context config
├── specs/               # global specs (source of truth, ~70+ domains)
└── changes/
    ├── <name>/          # active change
    │   ├── proposal.md
    │   ├── design.md
    │   ├── specs/
    │   └── tasks.md
    └── archive/         # completed changes
```

### Common commands

```bash
openspec list                    # list active changes
openspec new change <name>       # scaffold a new change
openspec status <name>           # show task completion for a change
openspec show <name>             # display full change details
openspec validate <name>         # validate artifacts
openspec archive <name>          # merge specs into openspec/specs/, move to archive
openspec view                    # interactive dashboard
```

### Workflow

1. **Propose** — `openspec new change <name>`, fill in `proposal.md`
2. **Design** — fill in `design.md` with technical decisions
3. **Spec** — add behavioral specs under `specs/`
4. **Apply** — implement via `tasks.md`; use `/opsx:apply` skill or work tasks manually
5. **Archive** — `openspec archive <name>` when all tasks are done

### Active changes (as of last update)

`add-rate-limiting`, `deployment-modes`, `frontend-quality`, `go-concurrency-fixes`, `keybindings`, `settings-wizard`, `ui-polish`

## Multi-Agent Claude Code Workflow

UNCWORKS uses Claude Code subagents for parallel exploration and implementation. Key principles:

### How subagents are used

- **Parallel exploration**: Spin up subagents to investigate different parts of the codebase simultaneously, then merge findings before writing code.
- **Thin vertical slices**: Each subagent works on a scoped, independently verifiable unit. Avoid big-bang changes.
- **Stop on invalidation**: If new information discovered by one subagent invalidates the plan, stop all others and re-plan.

### Agent roles in the platform

UNCWORKS itself runs two agent roles via `PI_ROLE` env var:

| Role | Responsibility |
|---|---|
| `manage` | PLAN stage: reads repo, runs `openspec` CLI, writes specs and tasks. VERIFY stage: checks task completion, validates implementation. |
| `implement` | EXECUTE stage: reads specs from workspace, writes code, runs tests. |

### Skills available for this repo

Invoke via `/skill-name` in Claude Code:

| Skill | When to use |
|---|---|
| `/uncworks-deploy` | Build images + rollout to colima-uncworks dev cluster |
| `/uncworks-image-push` | Build images into k8s.io namespace (no rollout) |
| `/uncworks-run-tests` | Choose and run the right test suite for what changed |
| `/uncworks-release` | Understand the release process (Release Please) |
| `/uncworks-new-change` | Create a new OpenSpec change |
| `/uncworks-audit-openspec` | List and categorize active OpenSpec changes |
| `/uncworks-rebuild-app` | Rebuild and reinstall the macOS desktop app |
