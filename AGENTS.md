# AGENTS.md

This file tells an AI coding agent how to work in this repository.

## Project overview

UNCWORKS is a Kubernetes-native platform for running AI coding agents. A user
submits a prompt and a git repository. The platform provisions an isolated
workspace, runs the agent, and streams the result in real time. The core
abstraction is the `AgentRun` custom resource.

## Commands

Every command runs through [Task](https://taskfile.dev/). See `Taskfile.yml` and
`tasks/`. Enter the toolchain first with `devbox shell`.

`tasks/homelab.yml` is machine-specific and is not tracked. Copy
`tasks/homelab.yml.example` if you need it. The include is optional, so `task`
works without it.

### Build

```bash
task build          # all Go binaries into ./bin/
task build:web      # web dashboard (Vite)
task build:app      # native macOS app (Wails v2, macOS only)
task build:uncworks # cross-compile the uncworks CLI (linux and darwin, amd64 and arm64)
task proto:gen      # regenerate Go and TypeScript code from .proto files
task proto:lint     # lint the protobuf definitions
task proto:breaking # check for breaking proto changes against main
```

`cmd/bff` embeds `cmd/bff/dist`. A tracked placeholder file satisfies the embed
pattern, so `go build ./...` works before you build the web bundle. Do not delete
it.

### Test

```bash
task test              # all tests in parallel (Go, web, extension, layer 2)
task test:go           # Go unit and integration tests (api/... internal/...)
task test:unit         # Go unit only. Fast, and needs no Docker
task test:contract     # ConnectRPC and protovalidate contract tests
task test:temporal     # Temporal workflow tests
task test:layer2       # Layer 2 pipeline tests (LLM stubbed, no cluster)
task test:regression   # Regression suite. Gates releases and PRs to main
task test:web          # Playwright tests for the web dashboard
task test:extension    # pi-aot-extension TypeScript tests
task test:shared       # @aot/shared TypeScript tests
task test:e2e          # Go E2E tests. Needs a running cluster
task test:e2e:full     # set up soft-serve, run E2E, run Playwright, tear down
```

Run one Go test with
`go test ./internal/server/... -run TestCreateAgentRun -count=1`.

Controller tests need envtest. `internal/testutil.EnsureEnvtestAssets()` resolves
the assets automatically. Set `KUBEBUILDER_ASSETS` yourself only if that fails.
Install the resolver with
`go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest`.

Run controller tests with `-p 1`. Parallel test binaries race to extract the same
envtest etcd binary, and the loser fails with `text file busy`.

### Lint

```bash
task lint           # golangci-lint plus TypeScript type checks
```

Linting uses [golangci-lint](https://golangci-lint.run/) v2. The config is
`.golangci.yml`. Enabled linters: govet, errcheck, staticcheck, unused,
ineffassign, gocritic, misspell, bodyclose, noctx, sqlclosecheck, rowserrcheck,
err113, and wrapcheck. The formatter is gofmt. Generated code under `gen/go/` is
excluded.

### Local dev cluster (colima-uncworks)

```bash
task dev:web          # start the Vite HMR dev server
task dev:images       # build all images into the colima-uncworks k8s.io namespace
task dev:deploy       # build images, restart the rollout, report status
task dev:install      # install all Go and npm dependencies
task dev:hooks:install # install git hooks through lefthook
```

### Kubernetes operations

```bash
task k8s:crd          # apply the AgentRun CRD
task k8s:deps         # deploy CRDs, storage, Ollama, LiteLLM, and soft-serve
task k8s:images       # build images with docker and import them into k0s (needs sudo)
task k8s:deploy:all   # build web, import images, roll out every deployment
task cluster:setup    # install systemd units, build and import images, start services
task cluster:status   # report the health of every service and port
task cluster:teardown # stop every service and remove the systemd units
task cluster:logs     # combined logs from every service
task cluster:temporal:dev  # start the Temporal dev server (SQLite, no external deps)
```

## Architecture

Two gRPC APIs define all communication.

`proto/api.proto` is the client API. `AOTService` listens on `:50055` and serves
CreateAgentRun, GetAgentRun, ListAgentRuns, WatchAgentRun (server-streaming),
CancelAgentRun, and SendHumanInput.

`proto/agent.proto` is the sidecar API. `AgentSidecarService` listens on `:50052`
and serves StartAgent, StreamOutput, SendInput, GetStatus, and StopAgent. The
same file defines `AgentNotificationService`, which carries asynchronous events
from the sidecar to the control plane.

Generated code lives in `gen/go/`. Regenerate it with `task proto:gen`, which
runs `buf generate`.

### Go binaries (`cmd/`)

| Binary | Role |
|--------|------|
| `apiserver` | ConnectRPC server and REST endpoints (`:50055`) |
| `controller` | Watches `AgentRun` resources and creates pods |
| `hydration` | Init container. Clones the repo and runs devbox setup |
| `sidecar` | RPC gateway. Bridges the agent process to the control plane (`:50052`) |
| `temporal-worker` | Temporal activity worker. Executes pipeline stages |
| `uncworks` | End-user CLI (`uncworks setup`, `uncworks open`, `uncworks tui`) |
| `aot` | Internal CLI for workspace tooling (`aot open`) |
| `bff` | Backend-for-frontend server for the macOS desktop app |
| `uncworks-app` | macOS desktop app (Wails v2) |

### Key Go packages (`internal/`)

- `server/` implements the gRPC `AOTService` and the WebSocket event hub.
- `controller/` reconciles the `AgentRun` resource. `multi_agent.go` handles
  `spawn_junior` child runs.
- `brain/` is the PostgreSQL state store (pgx). It holds agent state, metadata,
  and the priority queue.
- `hydration/` creates a bare git clone, then a worktree, then runs devbox setup.
- `sidecar/` is the RPC gateway that runs inside agent pods.
- `bff/` proxies for the desktop app, caches responses, and serves the SPA.
- `cli/` implements `aot open`.
- `embeddings/` generates embeddings for knowledge search through Ollama.
- `eventbus/` is an in-memory publish and subscribe bus for SSE and WebSocket
  events.
- `github/` is the GitHub App and PAT client. It handles webhooks and creates
  pull requests.
- `litellm/` is the LiteLLM admin API client. It provisions and revokes keys.
- `softserve/` is the Soft-Serve git client and project config repo scaffolder.
- `testutil/` holds shared test helpers and resolves envtest assets.

### Custom resources (`api/v1alpha1/`)

The resources are `AgentRun`, `Project`, `Chain`, `Schedule`, and `RunTemplate`.

`AgentRun` is the primary resource. Its spec carries repos, prompt, modelTier,
orchestrationMode, pipelineConfig, autoPush, and autoPR. Its status carries
phase, stage, verificationResult, prUrl, and totalCost. The phases are Pending,
Running, and then Succeeded, Failed, or Cancelled. A run waiting on a human sits
in WaitingForInput.

### TypeScript packages (`packages/`)

- `@aot/shared` is the gRPC client wrapper and the reactive agent state store.
- `@aot/pi-extension` is the agent harness extension. It provides the
  `ask_human` tool for human-in-the-loop, the `spawn_junior` tool for
  multi-agent runs, and OpenTelemetry tracing.

### Workspace layout

Each run gets a persistent workspace on a PVC mounted at `/workspace`.

```
/workspace/
├── <repo-name>/            # Git worktree (checked-out working copy)
├── .aot/
│   ├── logs/agent.log     # Agent stdout and stderr
│   ├── traces/spans.jsonl # Execution trace spans
│   └── metadata.json      # Run metadata snapshot
├── .devcontainer/
│   └── devcontainer.json  # VS Code Remote config
├── uncspace.yaml          # Workspace manifest
└── devbox.json            # Composed devbox config
```

The sidecar tees agent stdout and stderr into `.aot/logs/agent.log`.
`.aot/traces/spans.jsonl` records tool calls, LLM interactions, and git diffs as
JSONL. `.aot/metadata.json` snapshots the run spec.
`.devcontainer/devcontainer.json` lets VS Code Remote attach.

The files stay on the PVC after the run completes and the deployment scales to
zero replicas. The API serves them from there.

### Web dashboard (`web/`)

React 19, React Router 7, Vite, and Tailwind CSS. It reaches the API server over
ConnectRPC, and receives real-time updates over WebSocket and SSE.

## Data flow

1. A client calls `CreateAgentRun` over gRPC, or applies the resource with
   `kubectl`.
2. The controller sees the new `AgentRun` and creates a pod with an init
   container, an agent container, and a sidecar.
3. The init container clones the repo, creates a worktree, and runs
   `devbox install`.
4. The agent container executes the prompt inside the workspace.
5. The sidecar streams output back to the control plane over gRPC.
6. Clients watch through `WatchAgentRun` or over WebSocket.

## Git hooks and releases

[Lefthook](https://lefthook.dev/) manages the git hooks. The config is
`lefthook.yml`. Hooks install when you enter `devbox shell`, or when you run
`task dev:hooks:install`.

- pre-commit runs gofmt, golangci-lint against new changes only, buf lint, and
  the TypeScript type checks.
- commit-msg enforces [Conventional Commits](https://www.conventionalcommits.org/)
  through commitlint.
- pre-push runs the Go tests and buf breaking-change detection.

Run `git commit` from inside `devbox shell`. The hooks call `gofmt`,
`golangci-lint`, and `npx`, and none of them are on the bare PATH.

Releases use [Release Please](https://github.com/googleapis/release-please).
Conventional commit messages on `main` generate the changelog and the version
bump. The CI workflow runs Release Please after each merge to `main`. Every
passing push to `main` also tags a pre-release as
`vX.Y.Z-pre.YYYYMMDD.sha7`.

## Conventions

- Use Mermaid for every diagram in markdown. Never use ASCII box drawings.
- Write tests with Ginkgo and Gomega. Controller tests use envtest. gRPC tests
  use real listeners on `127.0.0.1:0`.
- The Go module is `github.com/uncworks/aot`.
- The resource group is `aot.uncworks.io/v1alpha1`.
- The labels are `aot.uncworks.io/parent`, `aot.uncworks.io/role`, and
  `aot.uncworks.io/managed`.
- The API server listens on `:50055` and the sidecar on `:50052`.
- Write commit messages as [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`, `chore:`.
- Write documentation in Simplified Technical English. Use one instruction per
  sentence, active voice, and one term for one concept. Do not use em-dashes, do
  not use emoji, and do not open a sentence or a list item with a bolded label.

## OpenSpec

OpenSpec is the change management system for this repository. It enforces a
spec-driven workflow: propose, design, spec, implement, archive.

### Directory layout

```
openspec/
├── config.yaml          # schema and project context
├── specs/               # the spec corpus, and the source of truth
└── changes/
    ├── <name>/          # an active change
    │   ├── proposal.md
    │   ├── design.md
    │   ├── specs/
    │   └── tasks.md
    └── archive/         # completed changes
```

### Commands

```bash
openspec list                    # list active changes
openspec new change <name>       # scaffold a new change
openspec status <name>           # show task completion for a change
openspec show <name>             # display the full change
openspec validate <name>         # validate the artifacts
openspec archive <name>          # merge specs into openspec/specs/ and archive
openspec view                    # interactive dashboard
```

### Workflow

`openspec/schema.yaml` is the authority. It extends openspec's built-in
`spec-driven` schema with two artifacts and tightens the authoring rules.
Templates live in `openspec/templates/`.

1. Propose. Run `openspec new change <name>` and fill in `proposal.md`. The
   `## Behavior` section carries the acceptance criteria, and it is the rubric
   every later artifact is reviewed against.
2. Pin the citations. Capture every external-factual claim into
   `citations.lock` with `uncworks cite capture`. Every change carries a lock,
   including one that cites nothing.
3. Design. Fill in `design.md`. Every Decision names a rejected alternative.
4. Spec. Add behavioral specs under `specs/`. Every requirement carries at least
   one declared-negative scenario.
5. Shape the tasks. Every non-Rollout phase declares `SHAPE loop` or
   `SHAPE graph`, and ends with an adversarial-review task.
6. Review. Record the adversarial review in `review.md`. The owner writes the
   decision, not the author.
7. Apply. Drive the work from `uncworks spec next`, not from the top of
   `tasks.md`.
8. Archive. Run `openspec archive <name>` when every task is done.

Run `openspec list` to see the active changes. Do not rely on a list in this
file, because it goes stale.

### Deterministic checks

```bash
uncworks spec check [<change>]   # the rubric lint. Exits 1 on an error finding
uncworks spec next  [<change>]   # the runnable set in the active phase
uncworks spec status             # task completion
uncworks spec graph <change>     # the phases and edges as Mermaid
uncworks spec rules              # what the rubric enforces in this build

uncworks cite capture <url> --id <id> --quote <text> --class <class>
uncworks cite verify [<lockdir>] # offline. No network, same verdict every time
uncworks cite recheck            # re-run the live checks over the pinned records
```

Both commands are pure functions of the files on disk, which is what lets them
gate a commit. Neither calls an LLM.

The four changes that were in flight when these rules landed predate them and do
not pass `uncworks spec check` yet.

### Skills

| Skill | When to use it |
|---|---|
| `spec-driven` | Drive the change workflow end to end |
| `citation-verification` | Pin an external-factual claim |
| `adversarial-review` | Run the critic loop at a phase's review gate |

## Multi-agent workflow

### How subagents are used

- Explore in parallel. Start subagents to investigate different parts of the
  codebase at the same time, then merge the findings before you write code.
- Work in thin vertical slices. Each subagent takes a scoped unit that can be
  verified on its own. Avoid one large change.
- Stop on invalidation. If one subagent finds something that invalidates the
  plan, stop the others and re-plan.

### Agent roles in the platform

UNCWORKS runs two agent roles, selected by the `PI_ROLE` environment variable.

| Role | Responsibility |
|---|---|
| `manage` | PLAN stage: read the repo, run the `openspec` CLI, write specs and tasks. VERIFY stage: check task completion and validate the implementation. |
| `implement` | EXECUTE stage: read the specs from the workspace, write code, run tests. |
