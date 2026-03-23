# CRD Reference

UNCWORKS defines two custom resource definitions: `AgentRun` for individual agent executions and `Project` for organizational grouping with default configuration.

Source files: `api/v1alpha1/types.go`, `api/v1alpha1/project_types.go`

---

## AgentRun

**API Version:** `aot.dev/v1alpha1`
**Kind:** `AgentRun`
**Scope:** Namespaced

### Spec Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `backend` | `BackendType` | `Pod` | Execution backend. Currently only `Pod` is supported. |
| `repos` | `[]Repository` | -- | Git repositories to clone into the workspace. At least one required for most runs. |
| `prompt` | `string` | -- | Task description for the agent. Required for single/manual mode; auto-generated for spec-driven if `specContent` is provided. |
| `modelTier` | `string` | `default` | LiteLLM model name for LLM routing. Common values: `default` (Ollama local), `default-cloud` (OpenRouter), `premium` (Anthropic/OpenAI). |
| `manageModelTier` | `string` | -- | Model for plan/verify stages in spec-driven runs. Falls back to `modelTier` if empty. |
| `implementModelTier` | `string` | -- | Model for execute stage in spec-driven runs. Falls back to `manageModelTier`, then `modelTier` if empty. |
| `maxBudget` | `float64` | -- | Maximum LLM spend budget in USD for this run. Enforced via LiteLLM virtual key budget caps. |
| `ttlSeconds` | `int32` | `3600` | Maximum run lifetime in seconds. Agent is killed when TTL expires. |
| `image` | `string` | -- | Override the default agent container image. |
| `envVars` | `map[string]string` | -- | Additional environment variables passed to the agent process. |
| `devboxConfig` | `string` | -- | Path to a devbox.json configuration for workspace setup. |
| `orchestrationMode` | `OrchestrationMode` | `single` | Decomposition mode (see Orchestration Modes below). |
| `orchestration` | `*Orchestration` | -- | Task list for manual orchestration mode. |
| `specContent` | `string` | -- | CodeSpeak `.cs.md` spec body (markdown). When set, auto-upgrades to spec-driven mode. |
| `specSource` | `string` | -- | Origin tracking for the spec. Values: `editor`, `webhook:github:<repo>/<path>`, `github:<owner/repo/path>`, `ci-autofix:<owner/repo>#<sha>`. |
| `specRef` | `string` | -- | Name of a spec in the project's config repo (e.g., `add-comments`). Resolves to `openspec/specs/{specRef}/spec.md` in the project's soft-serve repo. Requires `projectRef`. |
| `projectRef` | `string` | -- | Name of the Project CRD this run belongs to. When set, empty run fields are inherited from the project's defaults. |
| `specRunID` | `string` | -- | Groups all runs from a single spec execution. Used by `GetRunGraph` to build the run tree. |
| `parentRunID` | `string` | -- | Links this junior run to its parent senior run in orchestrated execution. |
| `displayName` | `string` | -- | Human-readable name. Auto-generated from the prompt by the LLM at creation time. |
| `workspaceName` | `string` | -- | Name of the workspace preset used for this run. |
| `pipelineConfig` | `*PipelineConfig` | -- | Per-stage configuration for spec-driven runs. |
| `autoPush` | `bool` | `false` | Push changes to a feature branch (`aot/<run-id>`) after successful verification. |
| `autoPR` | `bool` | `false` | Create a GitHub PR after pushing changes. Requires `autoPush` to be true. |
| `prBaseBranch` | `string` | `main` | Base branch for the auto-created PR. |
| `project` | `string` | -- | Project label for filtering (stored as `aot.uncworks.io/project` label). |
| `feature` | `string` | -- | Feature/unit-of-work label for filtering (stored as `aot.uncworks.io/feature` label). |
| `tags` | `[]string` | -- | Freeform tags for cross-cutting filtering (stored as `aot.uncworks.io/tags` annotation, comma-separated). |

### Repository

| Field | Type | Description |
|-------|------|-------------|
| `url` | `string` | Git repository URL (required). HTTPS or SSH format. |
| `branch` | `string` | Branch to check out. Uses the repo's default branch if empty. |
| `path` | `string` | Directory name under `/workspace/`. Derived from the URL if empty (e.g., `https://github.com/foo/bar.git` becomes `bar`). |

### PipelineConfig

Per-stage configuration for the spec-driven pipeline. Each stage has independent settings.

| Field | Type | Description |
|-------|------|-------------|
| `plan` | `StageConfig` | Planning stage configuration. |
| `execute` | `StageConfig` | Execution stage configuration. |
| `verify` | `StageConfig` | Verification stage configuration. |

### StageConfig

| Field | Type | Default (plan / execute / verify) | Description |
|-------|------|-----------------------------------|-------------|
| `model` | `string` | `default-cloud` / `default-cloud` / `default-cloud` | LiteLLM model name for this stage. |
| `timeoutSeconds` | `int32` | `300` / `900` / `180` | Stage timeout in seconds. |
| `maxRetries` | `int32` | `2` / `3` / `1` | Maximum retry attempts. |
| `onFailure` | `string` | `fail` / `retry` / `fail` | Behavior when retries exhausted: `retry`, `fail`, or `skip`. |

### Orchestration Modes

| Mode | Value | Description |
|------|-------|-------------|
| Single | `single` | One agent handles the entire task. Default for runs without `specContent`. |
| Auto | `auto` | Senior agent decomposes the task, spawns junior agents in parallel. Currently falls back to single-run execution (structured output collection pending). |
| Manual | `manual` | User defines an explicit subtask list in `orchestration.tasks[]`. Junior agents are spawned in parallel, max 7 tasks. |
| Spec-Driven | `spec-driven` | Full Plan/Execute/Verify pipeline. Auto-selected when `specContent` is provided. Uses OpenSpec CLI for artifact management. |

### Orchestration (Manual Mode)

| Field | Type | Description |
|-------|------|-------------|
| `tasks` | `[]OrchestrationTask` | List of sub-tasks (max 7). |

**OrchestrationTask:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Short kebab-case identifier for the task. Used in child workflow names. |
| `prompt` | `string` | Task description for the junior agent. |
| `repoUrls` | `[]string` | Optional subset of repos for this task. Inherits parent repos if empty. |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | `AgentRunPhase` | Current lifecycle phase (see Phase Enum below). |
| `message` | `string` | Human-readable status information. Updated throughout the workflow. |
| `podName` | `string` | Name of the provisioned pod (derived from deployment name). |
| `deploymentName` | `string` | Name of the Deployment managing the agent pod. |
| `traceID` | `string` | OpenTelemetry trace ID for this run. |
| `worktreePath` | `string` | Git worktree path on the agent. |
| `startedAt` | `*metav1.Time` | When the agent started running. |
| `completedAt` | `*metav1.Time` | When the agent finished. |
| `retainUntil` | `*metav1.Time` | When the pod retention expires and cleanup will run. |
| `logOutput` | `string` | Persisted agent log output (up to 1MB), collected before pod deletion. |
| `debugActive` | `bool` | Whether a debug session is currently active (pod scaled up for access). |
| `stage` | `string` | Current pipeline stage for spec-driven runs: `planning`, `executing`, `verifying`. Empty for non-spec-driven runs. |
| `retryCount` | `int32` | Number of execute/verify retry attempts completed. |
| `verificationResult` | `string` | JSON-encoded verification verdict from the verify stage. |
| `prUrl` | `string` | URL of the GitHub PR created by the pipeline (when autoPR is true). |
| `archived` | `bool` | Whether this run has been archived (hidden from default list views). |
| `totalCost` | `string` | Estimated total LLM cost of this run (e.g., `$0.12`). |
| `totalAdditions` | `int32` | Aggregate number of lines added across all diffs. |
| `totalDeletions` | `int32` | Aggregate number of lines deleted across all diffs. |
| `ciFixAttempts` | `int32` | Number of CI autofix attempts for this run's PR branch. |
| `lastCIStatus` | `string` | Most recent CI check status: `success` or `failure`. |
| `parentPRUrl` | `string` | URL of the PR this fix run is targeting (CI autofix runs). |
| `conditions` | `[]metav1.Condition` | Standard Kubernetes conditions. |

### Phase Enum

| Phase | Description |
|-------|-------------|
| `Pending` | CRD created, waiting for workflow to start pod provisioning. |
| `Running` | Agent actively working. In spec-driven mode, cycles through plan/execute/verify stages. |
| `WaitingForInput` | Agent paused, waiting for human-in-the-loop input via the `ask_user` tool. |
| `Succeeded` | Task completed successfully. Verification passed (spec-driven). |
| `Failed` | Unrecoverable error. TTL exceeded, all retries exhausted, or agent crashed. |
| `Cancelled` | Cancelled by user via CancelAgentRun. |

### Labels and Annotations

Labels set automatically by the API server:

| Label | Value |
|-------|-------|
| `aot.uncworks.io/project` | `spec.project` field |
| `aot.uncworks.io/feature` | `spec.feature` field |
| `aot.uncworks.io/repo` | Repository name derived from first repo URL |
| `aot.uncworks.io/spec-run-id` | `spec.specRunID` field (used for GetRunGraph) |

Annotations set automatically:

| Annotation | Value |
|------------|-------|
| `aot.uncworks.io/tags` | Comma-separated tags from `spec.tags[]` |
| `aot.uncworks.io/pr-branch` | Branch name (set by CI autofix) |
| `aot.uncworks.io/ci-fix-sha` | SHA being fixed (set by CI autofix) |
| `aot.uncworks.io/ci-fix-attempt` | Attempt number (set by CI autofix) |

### Print Columns

`kubectl get agentruns` displays: Backend, Phase, Age.

---

## Project

**API Version:** `aot.dev/v1alpha1`
**Kind:** `Project`
**Scope:** Namespaced

### Spec Fields

| Field | Type | Description |
|-------|------|-------------|
| `displayName` | `string` | Human-readable project name. |
| `description` | `string` | Short summary of the project. |
| `repos` | `[]Repository` | Application source code repositories (GitHub). Inherited by runs via `projectRef`. |
| `devbox` | `*DevboxConfig` | Packages to install in every workspace via devbox. |
| `defaults` | `*ProjectDefaults` | Default run configuration inherited by project runs. |
| `ide` | `*IDEConfig` | Browser-based IDE settings for the project. |
| `ssh` | `*SSHConfig` | SSH access settings for the project workspace. |

### DevboxConfig

| Field | Type | Description |
|-------|------|-------------|
| `packages` | `[]string` | Nix packages to install (e.g., `go@1.22`, `nodejs@20`, `python@3.12`). |

### ProjectDefaults

Defaults inherited by runs that set `projectRef` to this project's name. Run-level fields override project defaults when both are set.

| Field | Type | Description |
|-------|------|-------------|
| `modelTier` | `string` | Default model for runs. |
| `manageModelTier` | `string` | Default model for plan/verify stages. |
| `implementModelTier` | `string` | Default model for execute stages. |
| `ttlSeconds` | `int32` | Default run timeout. |
| `orchestrationMode` | `string` | Default orchestration mode. |
| `autoPush` | `bool` | Enable automatic git push after successful runs. |
| `autoPR` | `bool` | Enable automatic PR creation after successful runs. |
| `prBaseBranch` | `string` | Target branch for auto-created PRs. |

### IDEConfig

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | `bool` | Whether IDE pods can be created for this project. |
| `image` | `string` | Docker image for the IDE pod. |
| `idleTimeoutMinutes` | `int32` | Idle timeout before scaling the IDE pod to 0. |

### SSHConfig

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | `bool` | Whether SSH access is available. |
| `authorizedKeys` | `[]string` | SSH public keys allowed to connect. |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `configRepoReady` | `bool` | Whether the soft-serve config repo has been created and scaffolded. |
| `configRepoURL` | `string` | In-cluster URL for the project config repo (e.g., `ssh://soft-serve:23231/project-<name>`). |
| `ideActive` | `bool` | Whether the IDE pod is currently running. |
| `idePodName` | `string` | Name of the IDE pod if active. |
| `runCount` | `int32` | Total number of runs for this project. |
| `lastRunId` | `string` | ID of the most recent run. |
| `lastRunAt` | `*metav1.Time` | When the most recent run was created. |
| `totalCost` | `string` | Aggregated estimated cost across all runs. |
| `conditions` | `[]metav1.Condition` | Standard Kubernetes conditions. |

### Print Columns

`kubectl get projects` displays: Display Name, Repos (run count), Config Ready, Age.

### Controller Behavior

When a Project is created:
1. The controller adds a finalizer (`project.aot.dev/finalizer`)
2. Creates a soft-serve Git repo named `project-<name>`
3. Scaffolds the repo with OpenSpec directory structure
4. Sets `status.configRepoReady = true` and `status.configRepoURL`

When a Project is deleted:
1. The controller deletes the soft-serve repo (best-effort, does not block deletion)
2. Removes the finalizer to allow Kubernetes to complete deletion
