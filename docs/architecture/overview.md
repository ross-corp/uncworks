# UNCWORKS Architecture Overview

UNCWORKS is a Kubernetes-native agentic development platform. It runs AI coding agents against software repositories using a spec-driven pipeline where a manage agent plans work as structured OpenSpec artifacts, an implement agent writes code, and the manage agent verifies the result -- all orchestrated by Temporal workflows inside Kubernetes.

## System Diagram

```mermaid
flowchart TD
    subgraph K8s["Kubernetes Cluster"]
        subgraph UI["UNCWORKS UI"]
            Web["Web Dashboard\nReact + Tailwind\n:30300"]
        end

        subgraph CP["UNCWORKS Control Plane"]
            API["API Server\nConnectRPC :50055"]
            Ctrl["K8s Controller\nwatches AgentRun CRD\nwatches Project CRD"]
            TW["Temporal Worker\npipeline activities"]
        end

        subgraph Deps["Dependencies"]
            Temporal["Temporal :7233"]
            LiteLLM["LiteLLM :4000"]
            Ollama["Ollama :11434"]
            SoftServe["Soft-Serve :23231"]
        end

        subgraph DP["Data Plane (per run)"]
            subgraph Pod["Agent Pod"]
                Init["init: hydration\ngit clone + devbox"]
                Agent["container: agent\nsleep infinity"]
                Sidecar["container: rpc-gateway\nsidecar :50052\nruns pi-coding-agent"]
                Vol[("volume: /workspace\nPVC, 2Gi default")]
            end
        end

        Web -->|nginx proxy| API
        CP -->|Signal/Query| Deps
        CP -->|creates pods| DP
    end

    K8s -->|LLM calls via LiteLLM| CloudLLMs["Cloud LLMs\nOpenRouter"]
    K8s -->|GitHub API push/PR| GitHub["GitHub\nRepos + CI"]
```

## Components

### UNCWORKS UI

| Component | Description |
|-----------|-------------|
| **Web Dashboard** | React + Tailwind SPA served via nginx. Includes run list, run detail with activity feed, file browser, trace timeline, verification panel, and project management. Proxies API calls to the API Server via nginx reverse proxy. Port 30300 (NodePort). |
| **BFF (Nginx)** | Reverse proxy bundled in the web image. Routes `/api/` and `/connect/` to the API Server. Serves static assets for the SPA. |

### UNCWORKS Control Plane

| Component | Description |
|-----------|-------------|
| **API Server** | ConnectRPC/gRPC server on port 50055. Handles run CRUD (CreateAgentRun, GetAgentRun, ListAgentRuns, CancelAgentRun, SendHumanInput, GetRunGraph, SearchPastWork). Also serves REST endpoints for traces, files, logs, projects, archives, debug sessions, webhooks, and spec push/pull. |
| **K8s Controller** | Watches `AgentRun` CRDs and starts Temporal workflows. Maps CRD spec fields to workflow input. Updates CRD status from workflow state. Also reconciles `Project` CRDs: scaffolds soft-serve config repos, manages finalizers, tracks run counts and costs. |
| **Temporal Worker** | Executes pipeline activities: provision LLM keys via LiteLLM, create agent pods, wait for hydration, start/poll/stop agents, run plan/execute/verify stages, push changes to feature branches, create GitHub PRs, enrich run tags, persist knowledge data, scale down deployments, revoke keys. |

### UNCWORKS Data Plane

| Component | Description |
|-----------|-------------|
| **Agent Pod** | One Deployment per run with a PVC at `/workspace`. Three containers: hydration init (clones repos, installs devbox packages), agent container (holds workspace alive via `sleep infinity`), rpc-gateway sidecar (runs `pi-coding-agent`, exposes ConnectRPC on port 50052 for the Temporal Worker). PVC persists across pod restarts for debug access. |

### Dependencies

| Component | Description |
|-----------|-------------|
| **Temporal Server** | Workflow orchestration engine (:7233). Manages workflow state, signals (cancel, human-input), queries (get-state), retries, and compensation (cleanup on any exit path). |
| **LiteLLM Proxy** | Centralized LLM routing (:4000). Routes model requests to local Ollama or cloud providers (OpenRouter). Manages per-run virtual API keys with budget caps and model access control. |
| **Ollama** | Local LLM inference server (:11434). Runs models like qwen3:8b for zero-cost local development. Supports CPU or GPU. |
| **Soft-Serve** | In-cluster Git server (:23231). Hosts per-project config repositories containing OpenSpec artifacts, specs, and project configuration. Created automatically when a Project CRD is reconciled. |

## Data Flow: Run Creation to Completion

```mermaid
flowchart TD
    User["User creates run via UI or webhook"] --> APIServer
    APIServer["API Server\nCreates AgentRun CRD in Kubernetes"] --> Controller
    Controller["Controller\nDetects new CRD\nStarts Temporal workflow"] --> Workflow

    subgraph Workflow["Temporal Workflow"]
        direction TB
        Provision["1. ProvisionLLMKey\nCreate virtual key in LiteLLM"]
        CreateDeploy["2. CreateDeployment\nPod with init + agent + sidecar"]
        WaitHydration["3. WaitForHydration\nInit container clones repos, devbox"]
        HydrateCtx["4. HydrateContext\nInject past work context (knowledge DB)"]

        Provision --> CreateDeploy --> WaitHydration --> HydrateCtx
        HydrateCtx --> ModeChoice{Mode?}

        subgraph Single["Single Mode"]
            S5["5. StartAgent\nSidecar launches pi with prompt"]
            S6["6. PollStatus\nEvery 5s until COMPLETED/FAILED"]
            S7["7. EnrichRunTags\nAuto-tag from git diff"]
            S5 --> S6 --> S7
        end

        subgraph Spec["Spec-Driven Mode"]
            SD5["5. PLAN stage\nManage agent creates OpenSpec change"]
            SD6["6. EXECUTE stage\nImplement agent writes code"]
            SD7["7. VERIFY stage\nManage agent checks specs + LLM judge"]
            SD8{"Verify\npassed?"}
            SD9["9. PushChanges\nPush to aot/run-id branch"]
            SD10["10. CreatePR\nOpen GitHub PR against base branch"]
            SD5 --> SD6 --> SD7 --> SD8
            SD8 -->|No| SD6
            SD8 -->|Yes| SD9 --> SD10
        end

        ModeChoice -->|Single| S5
        ModeChoice -->|Spec-Driven| SD5

        subgraph Cleanup["Cleanup (deferred, runs on every exit path)"]
            C1["PersistRunData\nSave to knowledge DB"]
            C2["EmbedRunData\nGenerate embeddings for search"]
            C3["RevokeLLMKey\nDelete virtual key from LiteLLM"]
            C4["ScaleDownDeployment\nScale deployment to 0 replicas"]
        end

        S7 --> Cleanup
        SD10 --> Cleanup
    end
```

## Project System

Projects provide organizational structure and default configuration for runs.

```mermaid
flowchart LR
    subgraph ProjectCRD["Project CRD spec"]
        P1["displayName"]
        P2["repos[]"]
        P3["devbox.packages[]"]
        P4["defaults:\nmodelTier, autoPush\nautoPR, prBaseBranch"]
    end

    subgraph SoftServe["Soft-Serve"]
        SS1["Creates repo:\nproject-name"]
        SS2["Scaffolds:\nopenspec/specs/\nopenspec/changes/\n.devcontainer/"]
    end

    ProjectCRD -->|creates| SoftServe
    ProjectCRD -->|projectRef on AgentRun| RunInherit

    RunInherit["Run Inheritance\nEmpty run fields are filled\nfrom project defaults\n(repos, model, TTL, autoPush, autoPR)"]
```

When a Project is created:
1. The controller creates a soft-serve Git repo named `project-<name>`
2. The repo is scaffolded with OpenSpec directory structure
3. Runs referencing the project via `projectRef` inherit default configuration
4. Specs can be stored in the config repo and referenced by `specRef`
5. Project status tracks run count, last run, and aggregated cost

## CI Autofix

UNCWORKS can automatically fix CI failures on branches it created.

```mermaid
flowchart TD
    CIFail["GitHub Actions CI fails on aot/* branch"] --> Webhook
    Webhook["GitHub Webhook\nPOST /api/v1/webhooks/github\nEvent: check_run\nAction: completed\nConclusion: failure"] --> Handler

    Handler["CI Autofix Handler\n1. Verify branch is aot/*\n2. Check retry count (max 3)\n3. Debounce 30s (coalesce failures)\n4. Fetch CI logs from GitHub Actions API\n5. Extract error lines\n6. Create fix AgentRun with CI error context"] --> FixRun

    FixRun["Fix Run\nSpec-driven run\nSkips PLAN stage\nPushes to same aot/* branch"] --> RetryCheck

    RetryCheck{"Max retries\nexhausted?"}
    RetryCheck -->|Yes| CircuitBreaker["Circuit Breaker\nPosts comment on PR:\nManual intervention required"]
    RetryCheck -->|No| CIFail
```

The autofix flow:
1. GitHub sends a `check_run` webhook when CI completes
2. Handler filters for `conclusion: failure` on `aot/*` branches
3. Multiple check_run events for the same SHA are debounced (30s window)
4. CI logs are fetched from GitHub Actions API, extracted from zip, and condensed to error-relevant lines
5. A new AgentRun is created with the CI errors as context
6. The fix run uses spec-driven mode but skips the PLAN stage
7. Changes are pushed to the same branch (AutoPush=true, AutoPR=false)
8. After 3 failed attempts, a circuit breaker comment is posted on the PR

## Webhook-Triggered Runs

Push events to repositories in the allowlist create runs automatically when `.cs.md` (CodeSpeak spec) files are added or modified.

```mermaid
flowchart TD
    Push["GitHub push event"] --> Handler

    Handler["Webhook Handler\n1. Validate HMAC-SHA256\n2. Check repo allowlist\n3. Scan commits for *.cs.md files\n4. Fetch file content from GitHub API\n5. Create AgentRun per spec file found"]
```
