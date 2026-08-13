# Custom resources

There are two custom resources. `AgentRun` describes one run. `Project` holds
organizational defaults.

The Go source is `api/v1alpha1/types.go` and `api/v1alpha1/project_types.go`.

## AgentRun

Group and version `aot.uncworks.io/v1alpha1`. Kind `AgentRun`. Namespaced.

### Spec

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `backend` | `BackendType` | `Pod` | Only `Pod` is supported. |
| `repos` | `[]Repository` | none | At least one for most runs. |
| `prompt` | `string` | none | Required for `single` and `manual`. A spec-driven run derives it from `specContent`. |
| `modelTier` | `string` | `default` | LiteLLM model name. |
| `manageModelTier`, `implementModelTier` | `string` | none | Overrides `modelTier` for that role. |
| `maxBudget` | `float64` | none | Spend cap in USD. The LiteLLM virtual key enforces it. |
| `ttlSeconds` | `int32` | `3600` | The platform kills the agent when it exceeds this. |
| `image` | `string` | none | Overrides the default agent image. |
| `envVars` | `map[string]string` | none | Extra environment variables for the agent. |
| `devboxConfig` | `string` | none | Path inside the repo to a `devbox.json`. |
| `orchestrationMode` | `OrchestrationMode` | `single` | See below. |
| `orchestration` | `*Orchestration` | none | Required for `manual`. |
| `specContent` | `string` | none | The body of a CodeSpeak `.cs.md` file. Setting it selects `spec-driven`. |
| `specSource` | `string` | none | One of `editor`, `webhook:github:<repo>/<path>`, `github:<owner/repo/path>`, or `ci-autofix:<owner/repo>#<sha>`. |
| `specRef` | `string` | none | Names a spec in the project's config repo, and needs `projectRef`. It resolves to `openspec/specs/{specRef}/spec.md`. |
| `projectRef` | `string` | none | The `Project` this run inherits defaults from. |
| `specRunID` | `string` | none | Groups orchestrated runs. |
| `parentRunID` | `string` | none | Links a child run to its parent. |
| `displayName` | `string` | none | An LLM generates it from the prompt. |
| `workspaceName` | `string` | none | Workspace preset. |
| `pipelineConfig` | `*PipelineConfig` | none | Per-stage settings. |
| `autoPush` | `bool` | `false` | Pushes to `aot/<run-id>` when the run succeeds. |
| `autoPR` | `bool` | `false` | Opens a pull request. It needs `autoPush`. |
| `prBaseBranch` | `string` | `main` | The branch the pull request targets. |
| `project`, `feature` | `string` | none | Labels for filtering. |
| `tags` | `[]string` | none | Joined with commas into the `aot.uncworks.io/tags` annotation. |
| `approvalMode` | `string` | empty, which means `hybrid` | One of `none`, `hitl`, `llm-judge`, or `hybrid`. See below. |
| `openspecChange` | `string` | none | When set, Verify runs `openspec list --change <name>` as a task-completion gate. A run that leaves it empty skips the gate. |

### Repository

| Field | Notes |
|-------|-------|
| `url` | Required. HTTPS or SSH. |
| `branch` | The repo's default branch when empty. |
| `path` | Defaults to the repo name taken from the URL. |

### Orchestration modes

| Mode | Behavior |
|------|----------|
| `single` | One agent and one prompt. This is the default when `specContent` is empty. |
| `auto` | A senior agent decomposes the work. This currently falls back to single-run execution. |
| `manual` | Up to 7 entries in `orchestration.tasks[]`. Each one gets a junior agent. |
| `spec-driven` | The full Plan, Execute, and Verify pipeline. Setting `specContent` selects it. |

An `OrchestrationTask` carries `name`, `prompt`, and an optional `repoUrls`.

### Approval modes

`approvalMode` decides which gates run before a run reaches `Succeeded`. An empty
value means `hybrid`.

| Mode | LLM judge | Human |
|------|-----------|-------|
| `none` | no | no |
| `llm-judge` | yes | no |
| `hitl` | no | yes |
| `hybrid`, the default | yes | yes, after the judge |

The judge always uses `deepseek-v3.1`, whatever model the run uses. This keeps
the judge's cost independent of the run's cost.

### PipelineConfig and StageConfig

```yaml
pipelineConfig:
  plan:    { model, timeoutSeconds, maxRetries, onFailure }
  execute: { model, timeoutSeconds, maxRetries, onFailure }
  verify:  { model, timeoutSeconds, maxRetries, onFailure }
```

Every stage defaults to model `default-cloud`. The timeouts are 300, 900, and 180
seconds. The retry limits are 2, 3, and 1. The `onFailure` defaults are `fail`,
`retry`, and `fail`. `onFailure` accepts `retry`, `fail`, or `skip`.

### Status

| Field | Notes |
|-------|-------|
| `phase` | See enum below. |
| `message` | Plain-text status. The workflow updates it as it progresses. |
| `podName`, `deploymentName` | Pod handles. |
| `traceID` | OpenTelemetry trace id. |
| `worktreePath` | On-pod worktree path. |
| `startedAt`, `completedAt`, `retainUntil` | Timestamps. |
| `logOutput` | Up to 1 MB of logs, persisted before the pod is deleted. |
| `debugActive` | True while a debug session is open. |
| `stage` | One of `planning`, `executing`, or `verifying`. Empty at any other time. |
| `retryCount` | How many Execute and Verify retries have run. |
| `verificationResult` | The JSON verdict from Verify. It is written whenever the LLM judge runs, including on runs that are not spec-driven. |
| `prUrl`, `parentPRUrl` | Pull request URLs. |
| `archived` | Hidden from default listings when true. |
| `totalCost`, `totalAdditions`, `totalDeletions` | Aggregates. |
| `ciFixAttempts`, `lastCIStatus` | CI autofix state. |
| `conditions` | Standard K8s conditions. |

### Phase

| | |
|---|---|
| `Pending` | The resource exists and the workflow has not started. |
| `Running` | Active. This covers pod provisioning, the agent's work, and the approval gate. |
| `WaitingForInput` | Paused on a question to the human, or on the final approval. |
| `Succeeded`, `Failed`, `Cancelled` | Terminal states. |

### Labels and annotations

The API server sets these.

| Key | Value |
|-----|-------|
| `aot.uncworks.io/project` | `spec.project` |
| `aot.uncworks.io/feature` | `spec.feature` |
| `aot.uncworks.io/repo` | First repo name |
| `aot.uncworks.io/spec-run-id` | `spec.specRunID` |
| `aot.uncworks.io/tags`, an annotation | `spec.tags[]` joined with commas |
| `aot.uncworks.io/pr-branch`, `ci-fix-sha`, `ci-fix-attempt`, all annotations | CI autofix state |

`kubectl get agentruns` prints Backend, Phase, and Age.

---

## Project

Group and version `aot.uncworks.io/v1alpha1`. Kind `Project`. Namespaced.

### Spec

| Field | Notes |
|-------|-------|
| `displayName`, `description` | |
| `repos` | `Repository[]`, inherited through `projectRef`. |
| `devbox.packages[]` | Devbox package names, such as `go@1.22` and `nodejs@20`. |
| `defaults` | A `ProjectDefaults`, described below. A field set on the run wins. |
| `ide` | An `IDEConfig`, with enable, image, and idle timeout. |
| `ssh` | An `SSHConfig`, with enable and authorized keys. |

A `ProjectDefaults` carries `modelTier`, `manageModelTier`, `implementModelTier`,
`ttlSeconds`, `orchestrationMode`, `autoPush`, `autoPR`, and `prBaseBranch`.

### Status

| Field | Notes |
|-------|-------|
| `configRepoReady` | True once the soft-serve scaffold is complete. |
| `configRepoURL` | For example `ssh://soft-serve:23231/project-<name>`. |
| `ideActive`, `idePodName` | |
| `runCount`, `lastRunId`, `lastRunAt`, `totalCost` | Aggregates. |
| `conditions` | |

`kubectl get projects` prints DisplayName, Repos, ConfigReady, and Age.

### Lifecycle

On create, the controller adds the `project.aot.dev/finalizer` finalizer, creates
the soft-serve repo `project-<name>` with the OpenSpec scaffolding, and sets
`configRepoReady` and `configRepoURL`.

On delete, the controller tries to delete the soft-serve repo, then removes the
finalizer. A failed deletion does not block the delete.
