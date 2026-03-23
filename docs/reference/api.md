# API Reference

The UNCWORKS API consists of a ConnectRPC service for run management and REST endpoints for files, traces, projects, and integrations. The API server listens on port 50055 by default.

## ConnectRPC Service: AOTService

Defined in `proto/aot/api/v1/api.proto`. All RPCs use ConnectRPC (gRPC-compatible, also callable via HTTP/JSON).

### CreateAgentRun

Creates a new agent run. Generates a human-readable display name from the prompt via LLM. Auto-sets labels for project, feature, repo, and tags.

```
rpc CreateAgentRun(CreateAgentRunRequest) returns (CreateAgentRunResponse)
```

**Request:**
```
CreateAgentRunRequest {
  spec: AgentRunSpec {
    backend: Backend               // POD (default, currently the only option)
    repos: [Repository]            // Git repositories to clone (at least one required)
    prompt: string                 // Task description (required)
    model_tier: string             // LiteLLM model name (default: "default")
    manage_model_tier: string      // Model for plan/verify stages
    implement_model_tier: string   // Model for execute stage
    ttl_seconds: int32             // Max lifetime in seconds (default: 3600)
    env_vars: map<string,string>   // Additional env vars for agent
    spec_content: string           // OpenSpec markdown content
    spec_source: string            // Origin: "editor", "webhook:github:...", "ci-autofix:..."
    project_ref: string            // Project CRD name for inheritance
    spec_ref: string               // Spec name in project config repo
    orchestration_mode: enum       // SINGLE, AUTO, MANUAL, SPEC_DRIVEN
    orchestration: Orchestration   // Task list for MANUAL mode
    pipeline_config: PipelineConfig // Per-stage config for spec-driven mode
    max_budget: double             // Max LLM spend in USD
    auto_push: bool                // Push changes to feature branch on success
    auto_pr: bool                  // Create GitHub PR on success (requires auto_push)
    pr_base_branch: string         // PR target branch (default: "main")
    project: string                // Project label for filtering
    feature: string                // Feature label for filtering
    tags: [string]                 // Freeform tags for cross-cutting filtering
    image: string                  // Override default agent container image
    devbox_config: string          // Path to devbox.json
    workspace_name: string         // Workspace preset name
    parent_run_id: string          // Links junior run to parent
    spec_run_id: string            // Groups runs from a single spec execution
  }
}
```

**Response:** `CreateAgentRunResponse { agent_run: AgentRun }`

### GetAgentRun

Retrieves the current state of a run by ID. Enriches with real-time Temporal workflow state via query. Populates `children` list for orchestrated runs.

```
rpc GetAgentRun(GetAgentRunRequest { id: string }) returns (AgentRun)
```

### ListAgentRuns

Lists runs with optional filters. Returns newest first. Excludes archived runs by default (include via `X-Include-Archived: true` header).

```
rpc ListAgentRuns(ListAgentRunsRequest) returns (ListAgentRunsResponse)
```

**Filters:**

| Field | Type | Description |
|-------|------|-------------|
| `phase_filter` | enum | Filter by phase (Pending, Running, Succeeded, etc.) |
| `spec_run_id` | string | Filter by spec execution group |
| `parent_run_id` | string | Filter by parent run (for child runs) |
| `stage_filter` | string | Filter by pipeline stage (planning, executing, verifying) |
| `project_filter` | string | Filter by project label |
| `feature_filter` | string | Filter by feature label |
| `tag_filter` | string | Filter by tag |
| `limit` | int32 | Maximum number of results |

**Response:** `ListAgentRunsResponse { agent_runs: [AgentRun] }`

### WatchAgentRun

Server-streaming RPC. Sends the current state as the initial event, then streams real-time updates via the event bus until the run reaches a terminal phase.

```
rpc WatchAgentRun(WatchAgentRunRequest { id: string }) returns (stream AgentRunEvent)
```

**Event types:** `PHASE_CHANGED`, `LOG`, `TOOL_CALL`, `WAITING_FOR_INPUT`, `COMPLETED`

### CancelAgentRun

Requests cancellation of a running agent. Sends a cancel signal to the Temporal workflow.

```
rpc CancelAgentRun(CancelAgentRunRequest { id: string }) returns (CancelAgentRunResponse)
```

### SendHumanInput

Provides human-in-the-loop input to a paused agent. Only valid when the run is in `WaitingForInput` phase. Signals the Temporal workflow which forwards to the sidecar.

```
rpc SendHumanInput(SendHumanInputRequest) returns (SendHumanInputResponse)
```

**Request:** `{ agent_run_id: string, input: string }`
**Response:** `{ accepted: bool }`

### GetRunGraph

Returns the parent/child run tree for a spec execution. Uses the `aot.uncworks.io/spec-run-id` label to find all related runs.

```
rpc GetRunGraph(GetRunGraphRequest { id: string }) returns (RunGraph)
```

**Response:**
```
RunGraph {
  nodes: [RunGraphNode {
    name: string
    phase: AgentRunPhase
    role: string        // "single", "senior", "junior"
    started_at: Timestamp
    completed_at: Timestamp
  }]
  edges: [RunGraphEdge {
    parent: string
    child: string
  }]
}
```

### SearchPastWork

Searches the knowledge base for relevant past work using vector similarity. Requires the knowledge system to be configured (PostgreSQL + embeddings).

```
rpc SearchPastWork(SearchPastWorkRequest) returns (SearchPastWorkResponse)
```

**Filters:**

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | Natural language search query (required) |
| `repo_url` | string | Filter by repository URL |
| `source_filter` | enum | `CODE`, `TRACE`, or unset (all) |
| `created_after` | Timestamp | Minimum creation date |
| `created_before` | Timestamp | Maximum creation date |
| `limit` | int32 | Max results (default: 10, max: 100) |

---

## REST Endpoints

### Run Files and Logs

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/runs/{id}/files` | List files in the run's workspace |
| `GET` | `/api/v1/runs/{id}/files/content?path=<path>` | Read file content from workspace |
| `GET` | `/api/v1/runs/{id}/logs` | Raw agent log output |
| `GET` | `/api/v1/runs/{id}/logs/structured` | Parsed structured log entries |
| `GET` | `/api/v1/runs/{id}/logs/thinking` | Extracted thinking/reasoning blocks |
| `GET` | `/api/v1/runs/{id}/verification` | Verification result JSON (spec-driven runs) |

### Traces

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/runs/{id}/traces` | List trace spans for a run |
| `GET` | `/api/v1/runs/{id}/traces/{spanId}/diff` | Get the git diff for a specific span |
| `GET` | `/api/v1/runs/{id}/traces/watch` | SSE stream of trace span updates |

### Archive

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/runs/{id}/archive` | Archive (soft-delete) a run |
| `POST` | `/api/v1/runs/bulk-archive` | Archive multiple runs at once |

### Debug

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/runs/{id}/debug` | Start a debug session (scales pod to 1) |
| `DELETE` | `/api/v1/runs/{id}/debug` | Stop a debug session |
| `GET` | `/api/v1/runs/{id}/connect` | WebSocket connection to the pod |

### Exec

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/runs/{id}/exec` | WebSocket shell exec into the agent pod |

### Projects

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/projects` | List all projects (newest first) |
| `POST` | `/api/v1/projects` | Create a new project |
| `GET` | `/api/v1/projects/{name}` | Get project details |
| `DELETE` | `/api/v1/projects/{name}` | Delete a project (and its soft-serve repo) |
| `GET` | `/api/v1/projects/{name}/files` | List files in the project's config repo |
| `GET` | `/api/v1/projects/{name}/files/{path...}` | Read a file from the config repo |
| `PUT` | `/api/v1/projects/{name}/files/{path...}` | Write a file to the config repo (with commit) |

**Create project request body:**
```json
{
  "name": "my-project",
  "displayName": "My Project",
  "description": "A short summary",
  "repos": [{"url": "https://github.com/owner/repo.git", "branch": "main"}],
  "devbox": {"packages": ["go@1.22", "nodejs@20"]},
  "defaults": {
    "modelTier": "default-cloud",
    "ttlSeconds": 1800,
    "autoPush": true,
    "autoPR": true,
    "prBaseBranch": "main"
  }
}
```

**Write file request body:**
```json
{
  "content": "file content here",
  "commitMessage": "update spec (optional, auto-generated if omitted)"
}
```

### Specs (GitHub Integration)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/specs/push` | Push spec content to a GitHub repository |
| `GET` | `/api/v1/specs/pull` | Pull spec content from a GitHub repository |
| `GET` | `/api/v1/specs/{id}/graph` | Get the run graph for a spec execution |
| `GET` | `/api/v1/specs/{id}/graph/watch` | SSE stream of run graph updates |

### Classification

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/classify` | Classify a prompt into project/feature/tags via LLM |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/webhooks/github` | GitHub webhook endpoint |

**Supported events:**

| Event | Action | Behavior |
|-------|--------|----------|
| `push` | -- | Scans commits for `.cs.md` files. Creates an AgentRun per spec file found. Fetches file content from GitHub API at the push SHA. |
| `check_run` | `completed` (failure) | Triggers CI autofix on `aot/*` branches. Debounces 30s, fetches CI logs, creates a fix run. Max 3 retry attempts per branch. |
| `check_run` | `completed` (success) | Updates `lastCIStatus` on the run associated with the branch. |

**Configuration (environment variables):**

| Variable | Description |
|----------|-------------|
| `GITHUB_WEBHOOK_SECRET` | HMAC-SHA256 secret for signature validation. If unset, all requests are accepted. |
| `GITHUB_WEBHOOK_REPOS` | Comma-separated allowlist of `owner/repo` strings. If unset, all repos are allowed. |
| `CI_AUTOFIX_MAX_RETRIES` | Max CI autofix attempts per branch (default: 3). |

---

## Internal Service: AgentSidecarService

Defined in `proto/aot/agent/v1/agent.proto`. Used by the Temporal Worker to communicate with agent pods. Not intended for external clients.

| RPC | Description |
|-----|-------------|
| `StartAgent` | Start the pi-coding-agent process |
| `GetStatus` | Get current agent process state |
| `StopAgent` | Send SIGINT/SIGKILL to agent |
| `SendInput` | Write HITL response to workspace |
| `ExecCommand` | Run a shell command in the pod |
| `StreamOutput` | Server-stream of stdout/stderr |
