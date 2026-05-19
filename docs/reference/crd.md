# CRDs

Two: `AgentRun` (one run) and `Project` (organizational defaults).

Source: `api/v1alpha1/types.go`, `api/v1alpha1/project_types.go`.

## AgentRun

`aot.dev/v1alpha1` · Kind `AgentRun` · namespaced.

### Spec

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `backend` | `BackendType` | `Pod` | Only `Pod` is supported. |
| `repos` | `[]Repository` | — | At least one for most runs. |
| `prompt` | `string` | — | Required (single/manual); auto-derived for spec-driven when `specContent` set. |
| `modelTier` | `string` | `default` | LiteLLM model name. |
| `manageModelTier`, `implementModelTier` | `string` | — | Per-role override; fall back to `modelTier`. |
| `maxBudget` | `float64` | — | USD cap, enforced by LiteLLM virtual key. |
| `ttlSeconds` | `int32` | `3600` | Agent killed when exceeded. |
| `image` | `string` | — | Override default agent image. |
| `envVars` | `map[string]string` | — | Extra env on the agent. |
| `devboxConfig` | `string` | — | Path inside repo to devbox.json. |
| `orchestrationMode` | `OrchestrationMode` | `single` | See below. |
| `orchestration` | `*Orchestration` | — | Required for `manual`. |
| `specContent` | `string` | — | CodeSpeak `.cs.md` body. Auto-upgrades to spec-driven. |
| `specSource` | `string` | — | `editor` / `webhook:github:<repo>/<path>` / `github:<owner/repo/path>` / `ci-autofix:<owner/repo>#<sha>`. |
| `specRef` | `string` | — | Spec in project's config repo (requires `projectRef`). Resolves to `openspec/specs/{specRef}/spec.md`. |
| `projectRef` | `string` | — | `Project` to inherit defaults from. |
| `specRunID` | `string` | — | Groups orchestrated runs. |
| `parentRunID` | `string` | — | Child → parent link. |
| `displayName` | `string` | — | Auto-generated from prompt by LLM. |
| `workspaceName` | `string` | — | Workspace preset. |
| `pipelineConfig` | `*PipelineConfig` | — | Per-stage settings. |
| `autoPush` | `bool` | `false` | Push to `aot/<run-id>` on success. |
| `autoPR` | `bool` | `false` | Open PR; requires `autoPush`. |
| `prBaseBranch` | `string` | `main` | PR target. |
| `project`, `feature` | `string` | — | Labels for filtering. |
| `tags` | `[]string` | — | Comma-joined into the `aot.uncworks.io/tags` annotation. |
| `approvalMode` | `string` | `""` → `hybrid` | `none` / `hitl` / `llm-judge` / `hybrid`. See below. |
| `openspecChange` | `string` | — | When set, Verify runs `openspec list --change <name>` as a task-completion gate; ad-hoc runs without it skip the gate. |

### Repository

| Field | Notes |
|-------|-------|
| `url` | Required. HTTPS or SSH. |
| `branch` | Default branch if empty. |
| `path` | Defaults to repo name from URL. |

### Orchestration modes

| Mode | Behavior |
|------|----------|
| `single` | One agent, one prompt. Default when `specContent` not set. |
| `auto` | Senior decomposes. Currently falls back to single-run execution. |
| `manual` | Up to 7 explicit `orchestration.tasks[]`; each a junior agent. |
| `spec-driven` | Full Plan/Execute/Verify. Auto-selected when `specContent` set. |

`OrchestrationTask`: `{ name, prompt, repoUrls? }`.

### Approval modes

`approvalMode` controls what gates run before flipping to `Succeeded`. Default (empty) is `hybrid`.

| Mode | LLM judge | Human |
|------|-----------|-------|
| `none` | — | — |
| `llm-judge` | yes | — |
| `hitl` | — | yes |
| `hybrid` (default) | yes | yes (after judge) |

The judge always uses `deepseek-v3.1` (cheap, dedicated) regardless of the run's model.

### PipelineConfig / StageConfig

```yaml
pipelineConfig:
  plan:    { model, timeoutSeconds, maxRetries, onFailure }
  execute: { model, timeoutSeconds, maxRetries, onFailure }
  verify:  { model, timeoutSeconds, maxRetries, onFailure }
```

Stage defaults: model `default-cloud`; timeouts 300 / 900 / 180 s; max retries 2 / 3 / 1; `onFailure` `fail` / `retry` / `fail`. `onFailure` ∈ `{retry, fail, skip}`.

### Status

| Field | Notes |
|-------|-------|
| `phase` | See enum below. |
| `message` | Human-readable status; updated as the workflow progresses. |
| `podName`, `deploymentName` | Pod handles. |
| `traceID` | OpenTelemetry trace id. |
| `worktreePath` | On-pod worktree path. |
| `startedAt`, `completedAt`, `retainUntil` | Timestamps. |
| `logOutput` | Persisted up to 1 MB before pod deletion. |
| `debugActive` | Debug session live. |
| `stage` | `planning` / `executing` / `verifying`; empty otherwise. |
| `retryCount` | Execute/verify retries so far. |
| `verificationResult` | JSON verdict from Verify; written even for non-spec-driven runs when the LLM judge runs. |
| `prUrl`, `parentPRUrl` | PR URLs. |
| `archived` | Hidden from default listings when true. |
| `totalCost`, `totalAdditions`, `totalDeletions` | Aggregates. |
| `ciFixAttempts`, `lastCIStatus` | CI autofix state. |
| `conditions` | Standard K8s conditions. |

### Phase

| | |
|---|---|
| `Pending` | CRD created, workflow not yet started. |
| `Running` | Active. Covers everything from pod provisioning through agent + approval gate. |
| `WaitingForInput` | Paused: HITL question, or final human approval. |
| `Succeeded` / `Failed` / `Cancelled` | Terminal. |

### Labels / annotations

Auto-set by the API server:

| Key | Value |
|-----|-------|
| `aot.uncworks.io/project` | `spec.project` |
| `aot.uncworks.io/feature` | `spec.feature` |
| `aot.uncworks.io/repo` | First repo name |
| `aot.uncworks.io/spec-run-id` | `spec.specRunID` |
| `aot.uncworks.io/tags` (annotation) | Comma-joined `spec.tags[]` |
| `aot.uncworks.io/pr-branch`, `ci-fix-sha`, `ci-fix-attempt` (annotations) | CI autofix state |

`kubectl get agentruns`: Backend, Phase, Age.

---

## Project

`aot.dev/v1alpha1` · Kind `Project` · namespaced.

### Spec

| Field | Notes |
|-------|-------|
| `displayName`, `description` | |
| `repos` | `Repository[]` — inherited via `projectRef`. |
| `devbox.packages[]` | `go@1.22`, `nodejs@20`, etc. |
| `defaults` | `ProjectDefaults` (below). Run-level fields override. |
| `ide` | `IDEConfig` (enable, image, idle timeout). |
| `ssh` | `SSHConfig` (enable, authorized keys). |

`ProjectDefaults`: `modelTier`, `manageModelTier`, `implementModelTier`, `ttlSeconds`, `orchestrationMode`, `autoPush`, `autoPR`, `prBaseBranch`.

### Status

| Field | Notes |
|-------|-------|
| `configRepoReady` | Soft-serve scaffold complete. |
| `configRepoURL` | e.g. `ssh://soft-serve:23231/project-<name>`. |
| `ideActive`, `idePodName` | |
| `runCount`, `lastRunId`, `lastRunAt`, `totalCost` | Aggregates. |
| `conditions` | |

`kubectl get projects`: DisplayName, Repos, ConfigReady, Age.

### Lifecycle

On create: finalizer `project.aot.dev/finalizer`, create soft-serve repo `project-<name>` with OpenSpec scaffolding, set `configRepoReady` + `configRepoURL`.

On delete: best-effort soft-serve repo deletion, then remove finalizer.
