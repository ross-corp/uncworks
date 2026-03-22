# AgentRun CRD Reference

The `AgentRun` custom resource (`agentruns.aot.dev/v1alpha1`) is the primary API object for UNCWORKS. It represents a single agent execution.

Source: `api/v1alpha1/types.go`

## Spec Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `backend` | `BackendType` | `Pod` | Execution backend: `Pod` |
| `repos` | `[]Repository` | -- | Git repositories to clone (required) |
| `prompt` | `string` | -- | Task description for the agent (required) |
| `modelTier` | `string` | `default` | LiteLLM model name for LLM routing |
| `ttlSeconds` | `int32` | `3600` | Maximum run lifetime in seconds |
| `image` | `string` | -- | Override default agent container image |
| `envVars` | `map[string]string` | -- | Additional environment variables |
| `devboxConfig` | `string` | -- | Path to devbox.json configuration |
| `orchestrationMode` | `OrchestrationMode` | `single` | Decomposition mode (see below) |
| `orchestration` | `*Orchestration` | -- | Task list for manual mode |
| `specContent` | `string` | -- | OpenSpec markdown body for spec-driven mode |
| `specSource` | `string` | -- | Origin of spec: `editor`, `github:<path>`, etc. |
| `specRunID` | `string` | -- | Groups runs from a single spec execution |
| `parentRunID` | `string` | -- | Links junior run to parent senior run |
| `displayName` | `string` | -- | Human-readable name (LLM-generated from prompt) |
| `workspaceName` | `string` | -- | Workspace preset name |
| `pipelineConfig` | `*PipelineConfig` | -- | Per-stage config for spec-driven runs |

## Repository

| Field | Type | Description |
|-------|------|-------------|
| `url` | `string` | Git repository URL (required) |
| `branch` | `string` | Branch to check out |
| `path` | `string` | Directory name under `/workspace/<repo>/` (derived from URL if empty) |

## Orchestration Modes

| Mode | Value | Description |
|------|-------|-------------|
| Single | `single` | One agent handles the entire task |
| Auto | `auto` | Agent autonomously decomposes into subtasks |
| Manual | `manual` | User defines explicit subtask list |
| Spec-Driven | `spec-driven` | OpenSpec Plan/Execute/Verify pipeline |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | `AgentRunPhase` | Current lifecycle phase |
| `message` | `string` | Human-readable status information |
| `podName` | `string` | Name of the provisioned pod |
| `deploymentName` | `string` | Name of the managing Deployment |
| `traceID` | `string` | OpenTelemetry trace ID |
| `worktreePath` | `string` | Git worktree path on the agent |
| `startedAt` | `*metav1.Time` | When the agent started running |
| `completedAt` | `*metav1.Time` | When the agent finished |
| `retainUntil` | `*metav1.Time` | Pod retention expiry |
| `logOutput` | `string` | Persisted agent log (up to 1MB) |
| `debugActive` | `bool` | Whether a debug session is active |
| `stage` | `string` | Pipeline stage: `planning`, `executing`, `verifying` |
| `retryCount` | `int32` | Execute/verify retry attempts completed |
| `verificationResult` | `string` | JSON-encoded verification verdict |
| `conditions` | `[]metav1.Condition` | Standard Kubernetes conditions |

## Phase Enum

| Phase | Description |
|-------|-------------|
| `Pending` | Waiting for pod provisioning |
| `Running` | Agent actively working |
| `WaitingForInput` | Paused for human-in-the-loop input |
| `Succeeded` | Task completed successfully |
| `Failed` | Unrecoverable error |
| `Cancelled` | Cancelled by user |

## Print Columns

`kubectl get agentruns` displays: Backend, Phase, Age.
