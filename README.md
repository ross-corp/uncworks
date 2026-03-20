# UNCWORKS

**An agentic development environment.**

UNCWORKS is a Kubernetes-native platform that runs AI coding agents against git repositories. It uses a spec-driven pipeline (Plan, Execute, Verify) with two agent roles: **manage** (plans and verifies) and **implement** (writes code). Determinism is enforced through pi extension policies that constrain agent behavior.

---

## Key Features

- **Spec-driven pipeline** -- Plan, Execute, Verify stages with structured handoffs
- **OpenSpec integration** -- formal change proposals, designs, and task tracking
- **Agent role separation** -- manage agents plan and verify; implement agents write code
- **Determinism extension** -- pi policies constrain tool usage and model access
- **Real-time UI** -- activity feed, OpenTelemetry traces, file browser
- **LiteLLM model routing** -- centralized LLM proxy with per-agent budget and model controls
- **Workspace isolation** -- each agent run gets its own git worktree on a persistent volume

---

## Quick Start

See [docs/getting-started.md](docs/getting-started.md) for full setup instructions.

```bash
devbox shell          # enter Nix dev environment
task install          # install Go + Node.js dependencies
task k0s:setup        # initialize local k0s cluster
task k0s:crd          # apply AgentRun CRD
task build            # build all Go binaries
task dev:web          # start web dashboard
```

---

## Architecture

Everything runs inside Kubernetes, except cloud LLM providers.

```mermaid
graph LR
    User(("User"))
    Cloud["OpenRouter<br/>Cloud LLMs"]

    subgraph K8s["Kubernetes Cluster"]
        direction TB

        subgraph CP["Control Plane"]
            Web["Web UI :30300"]
            API["API Server :50055"]
            Ctrl["K8s Controller"]
            TW["Temporal Worker"]
        end

        subgraph Deps["Dependencies"]
            Temporal["Temporal :7233"]
            LiteLLM["LiteLLM :4000"]
            Ollama["Ollama :11434"]
        end

        subgraph DP["Data Plane — Agent Pod (1 per run)"]
            Init["init: Hydration"]
            Pi["pi-coding-agent"]
            Sidecar["RPC Gateway"]
            PVC[("/workspace PVC")]
        end
    end

    User --> Web
    Web --> API
    API --> Temporal
    Temporal --> TW
    Ctrl --> API
    TW -->|creates| DP
    Sidecar --> TW
    Pi --> LiteLLM
    LiteLLM --> Ollama
    LiteLLM --> Cloud
    Init --> PVC
    Pi --> PVC
    Sidecar --> PVC
```

| Section | Component | Description |
|---------|-----------|-------------|
| **Control Plane** | Web UI | React dashboard. Activity feed, file browser, traces, verification panel. Proxied to API via nginx. |
| | API Server | ConnectRPC endpoints: create, get, list, cancel runs. REST endpoints: structured logs, files, traces, thinking. |
| | K8s Controller | Watches `AgentRun` CRDs, triggers Temporal workflows, updates CRD status. |
| | Temporal Worker | Executes pipeline activities: provision keys, create pods, hydrate, plan, execute, verify, cleanup. |
| **Dependencies** | Temporal | Workflow orchestration with retries, compensation, signals (cancel, human input). |
| | LiteLLM | LLM proxy with model routing, budgets, fallback chains. Routes to local Ollama or cloud (OpenRouter). |
| | Ollama | Local model server (qwen3:8b, llama3.1:8b). |
| **Data Plane** | init: Hydration | Bare-clones repos, creates git worktrees at `/workspace/<repo>/`, sets up devbox. |
| | pi-coding-agent | LLM agent with determinism extension. `PI_ROLE=manage` (plan/verify): reads repo, runs openspec CLI, writes specs. `PI_ROLE=implement` (execute): reads specs, writes code, runs tests. |
| | RPC Gateway | Sidecar bridging agent to control plane: StartAgent, GetStatus, ExecCommand, SendInput. Streams JSONL logs to PVC. |
| | /workspace PVC | Persistent volume with repo worktrees, OpenSpec artifacts, agent logs, and traces. |

### Sequence: Spec-Driven Run

```mermaid
sequenceDiagram
    actor User
    participant Web as Web UI
    participant API as API Server
    participant Ctrl as Controller
    participant TW as Temporal Worker
    participant Pod as Agent Pod
    participant LLM as LiteLLM / OpenRouter

    User->>Web: Create run (repo + prompt)
    Web->>API: CreateAgentRun
    API->>API: Create AgentRun CRD
    Ctrl->>TW: Start workflow

    TW->>Pod: Create pod + PVC
    Note over Pod: Hydration: clone, worktree, devbox

    Note over TW,LLM: PLAN (manage agent)
    TW->>Pod: Scaffold openspec change
    TW->>Pod: StartAgent (plan prompt)
    Pod->>LLM: Read repo, write specs
    Pod->>TW: Complete
    TW->>Pod: openspec validate

    Note over TW,LLM: EXECUTE (implement agent)
    TW->>Pod: StartAgent (specs + prompt)
    Pod->>LLM: Read specs, write code, run tests
    Pod->>TW: Complete

    Note over TW,LLM: VERIFY (manage agent)
    TW->>Pod: Check tasks, validate specs, LLM judge
    alt Pass
        TW->>Pod: openspec archive
        TW-->>API: Succeeded
    else Fail
        TW->>Pod: Retry execute
    end

    TW->>Pod: Scale down
```

---

## Development

All commands use [Task](https://taskfile.dev/) (see `Taskfile.yml`):

```bash
task build            # build all Go binaries
task test             # run all tests (Go + web + extension)
task lint             # golangci-lint + TypeScript type checks
task dev:web          # start Vite dev server
task k0s:setup        # initialize local k0s cluster
```

---

## Deployment

```bash
helm install aot oci://ghcr.io/uncworks/charts/aot \
  --namespace aot --create-namespace \
  --set temporal.host=temporal:7233
```

See [`deploy/helm/aot/README.md`](deploy/helm/aot/README.md) for configuration reference.

---

## License

See repository root for license details.
