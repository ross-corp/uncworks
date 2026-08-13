# API reference

The API server serves ConnectRPC and REST on `:50055`. One mux serves both, and
every RPC is callable over gRPC and over HTTP with JSON.

`proto/aot/api/v1/api.proto` defines `AOTService`.
`proto/aot/agent/v1/agent.proto` defines the internal `AgentSidecarService`.

## `AOTService`

### `CreateAgentRun`

```proto
rpc CreateAgentRun(CreateAgentRunRequest) returns (CreateAgentRunResponse);
```

Creates an `AgentRun` resource. It generates an `ar-XXXXXX` id and an LLM-derived
display name, and sets the project, feature, repo, and tag labels.

These are the main `AgentRunSpec` fields.

| Field | Type | Notes |
|-------|------|-------|
| `backend` | `Backend` | `POD` is the only value |
| `repos[]` | `Repository` | At least one for most runs |
| `prompt` | `string` | Required for `SINGLE` and `MANUAL`. A spec-driven run derives it from `spec_content` |
| `model_tier` | `string` | Default `default` |
| `manage_model_tier`, `implement_model_tier` | `string` | Overrides `model_tier` for that role |
| `ttl_seconds` | `int32` | Default 3600 |
| `env_vars` | `map<string,string>` | Extra env on the agent |
| `spec_content` | `string` | CodeSpeak markdown. Setting it selects `SPEC_DRIVEN` |
| `spec_source` | `string` | `editor`, `webhook:github:...`, `ci-autofix:...` |
| `project_ref` | `string` | Every empty field inherits from this `Project` |
| `spec_ref` | `string` | Names a spec in the project's config repo |
| `orchestration_mode` | enum | One of `SINGLE`, `AUTO`, `MANUAL`, or `SPEC_DRIVEN` |
| `orchestration` | `Orchestration` | Task list for `MANUAL` |
| `pipeline_config` | `PipelineConfig` | Per-stage overrides |
| `max_budget` | `double` | Spend cap in USD. The LiteLLM virtual key enforces it |
| `auto_push`, `auto_pr`, `pr_base_branch` | | Push and pull-request automation |
| `approval_mode` | `string` | One of `hybrid`, which is also the empty default, `none`, `hitl`, or `llm-judge` |
| `openspec_change` | `string` | Turns on the task-completion gate in Verify |
| `parent_run_id`, `spec_run_id` | `string` | Links an orchestrated run to its group |
| `image`, `devbox_config`, `workspace_name` | `string` | Override the workspace defaults |

### `GetAgentRun`

Takes `{ id }` and returns an `AgentRun`. It merges the live Temporal query state
with the stored resource, and fills in `children[]`.

### `ListAgentRuns`

The filters are `phase_filter`, `spec_run_id`, `parent_run_id`, `stage_filter`,
`project_filter`, `feature_filter`, `tag_filter`, and `limit`. Results come back
newest first. Archived runs are hidden unless the request sets
`X-Include-Archived: true`.

### `WatchAgentRun`

Streams `AgentRunEvent` messages. It emits the current state first, then each
change until the run is terminal. The event types are `PHASE_CHANGED`, `LOG`,
`TOOL_CALL`, `WAITING_FOR_INPUT`, and `COMPLETED`.

### `CancelAgentRun`

Takes `{ id }` and cancels the Temporal workflow.

### `SendHumanInput`

Takes `{ agent_run_id, input }` and returns `{ accepted }`. It forwards the input
to a paused agent.

For a question, `input` is the answer. For an approval gate, `input` is
`approve`, `reject`, `deny`, or `no`. Any other value counts as approval, and the
input becomes the reject reason where one applies.

### `GetRunGraph`

Takes `{ id }` and returns a `RunGraph`. The graph holds the parent run and its
children, grouped by `aot.uncworks.io/spec-run-id`. Each node carries `name`,
`phase`, `role`, `started_at`, and `completed_at`. The role is `single`,
`senior`, or `junior`.

### `SearchPastWork`

Runs a vector search over the artifacts of past runs. It needs the brain and
embedder subsystems.

`query` is required. The optional filters are `repo_url`, `source_filter`, which
takes `CODE` or `TRACE`, `created_after`, `created_before`, and `limit`. `limit`
defaults to 10 and caps at 100.

## REST

### Runs

| Method | Path | Returns |
|--------|------|---------|
| GET | `/api/v1/runs/{id}/files` | Directory listing |
| GET | `/api/v1/runs/{id}/files/content?path=` | File content |
| GET | `/api/v1/runs/{id}/logs` | The plain-text `agent.log` |
| GET | `/api/v1/runs/{id}/logs/structured` | `agent.jsonl` |
| GET | `/api/v1/runs/{id}/logs/thinking` | Reasoning blocks |
| GET | `/api/v1/runs/{id}/verification` | `VerificationResult` JSON |

### Traces

| Method | Path | Returns |
|--------|------|---------|
| GET | `/api/v1/runs/{id}/traces` | Spans |
| GET | `/api/v1/runs/{id}/traces/{spanId}/diff` | Per-span git diff |
| GET | `/api/v1/runs/{id}/traces/watch` | SSE stream |

### Archive

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/runs/{id}/archive` | Hide one run from the default listing |
| POST | `/api/v1/runs/bulk-archive` | Hide several runs at once |

### Debug and exec

| Method | Path | What it does |
|--------|------|---|
| POST/DELETE | `/api/v1/runs/{id}/debug` | Starts or stops a debug session, which scales the pod back to one replica |
| GET | `/api/v1/runs/{id}/connect` | WebSocket pod connect |
| GET | `/api/v1/runs/{id}/exec` | WebSocket shell |

### Projects

| Method | Path | |
|--------|------|---|
| GET/POST | `/api/v1/projects` | List or create |
| GET/DELETE | `/api/v1/projects/{name}` | Read or delete |
| GET | `/api/v1/projects/{name}/files` | Config repo listing |
| GET/PUT | `/api/v1/projects/{name}/files/{path...}` | Read, or write and commit |

Create body:

```json
{
  "name": "my-project",
  "displayName": "My Project",
  "description": "...",
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

Write body:

```json
{ "content": "...", "commitMessage": "update spec (optional)" }
```

### Specs

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/specs/push` | Push to GitHub |
| GET | `/api/v1/specs/pull` | Pull from GitHub |
| GET | `/api/v1/specs/{id}/graph` | Run graph for a spec execution |
| GET | `/api/v1/specs/{id}/graph/watch` | SSE |

### Misc

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/classify` | Classifies a prompt into a project, a feature, and tags |
| POST | `/api/v1/webhooks/github` | GitHub webhook |

Webhooks:

| Event | Action | Behavior |
|-------|--------|----------|
| `push` | none | Scans the commits for `.cs.md` files, and creates one `AgentRun` per spec at the pushed SHA. |
| `check_run` | `completed` with `failure` on an `aot/*` branch | Starts CI autofix. It debounces for 30 seconds, fetches the logs, condenses them, and creates a fix run. It allows 3 attempts per branch |
| `check_run` | `completed` with `success` | Updates `lastCIStatus` on the run that owns the branch |

Environment variables:

| Variable | Purpose |
|-----|---------|
| `GITHUB_WEBHOOK_SECRET` | The HMAC-SHA256 secret. When it is unset, the server does not validate the signature |
| `GITHUB_WEBHOOK_REPOS` | Comma-separated `owner/repo` allowlist. When it is unset, every repo is allowed |
| `CI_AUTOFIX_MAX_RETRIES` | Attempts per branch. Defaults to 3 |

## `AgentSidecarService`, internal

The Temporal worker calls this service on the agent pods. External clients MUST
NOT call it.

| RPC | What it does |
|-----|---|
| `StartAgent` | Spawns `pi` |
| `GetStatus` | Returns the process state |
| `StopAgent` | Sends SIGINT, then SIGKILL |
| `SendInput` | Writes the human's response |
| `ExecCommand` | Runs a shell command in the pod |
| `StreamOutput` | Streams stdout and stderr |
