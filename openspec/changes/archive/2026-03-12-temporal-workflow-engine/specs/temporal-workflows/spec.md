## ADDED Requirements

### Requirement: AgentRun Workflow Lifecycle
The `AgentRunWorkflow` SHALL accept an `AgentRunSpec` as input and execute the full agent lifecycle as a durable Temporal workflow.

#### Scenario: Successful agent run lifecycle
- **WHEN** an `AgentRunWorkflow` is started with a valid `AgentRunSpec`
- **THEN** the workflow SHALL execute `CreateAgentPod`, `WaitForHydration`, `StartAgent`, poll via `GetAgentStatus` until completion, and `CleanupPod` in sequence
- **AND** the workflow SHALL complete with a success result

#### Scenario: Agent run with failure during execution
- **WHEN** the agent fails during execution (detected via `GetAgentStatus`)
- **THEN** the workflow SHALL execute `CleanupPod` to remove the agent pod
- **AND** the workflow SHALL complete with a failure result containing the error details

### Requirement: Pod Creation Activity
The `AgentRunWorkflow` SHALL create an agent pod via the `CreateAgentPod` activity.

#### Scenario: Creating the agent pod
- **WHEN** the workflow executes the `CreateAgentPod` activity
- **THEN** the activity SHALL create a Kubernetes Pod with the agent container, RPC Gateway sidecar, and hydration init container
- **AND** the activity SHALL return the pod name upon successful creation

### Requirement: Hydration Wait Activity
The `AgentRunWorkflow` SHALL wait for hydration completion via the `WaitForHydration` activity.

#### Scenario: Waiting for init container completion
- **WHEN** the workflow executes the `WaitForHydration` activity
- **THEN** the activity SHALL poll the init container status until it reaches a terminal state
- **AND** the activity SHALL return success when the init container completes successfully
- **AND** the activity SHALL return an error if the init container fails

### Requirement: Agent Start Activity
The `AgentRunWorkflow` SHALL start the agent via the `StartAgent` activity, which makes a gRPC call to the RPC Gateway sidecar.

#### Scenario: Starting agent execution
- **WHEN** the workflow executes the `StartAgent` activity
- **THEN** the activity SHALL send a gRPC request to the RPC Gateway sidecar to begin agent execution
- **AND** the activity SHALL return success when the sidecar acknowledges the start command

### Requirement: Human-in-the-Loop Signal Handling
The `AgentRunWorkflow` SHALL handle `human-input` signals by forwarding input to the RPC Gateway sidecar via the `ForwardHumanInput` activity.

#### Scenario: Receiving and forwarding human input
- **WHEN** the workflow receives a `human-input` signal with input text
- **THEN** the workflow SHALL execute the `ForwardHumanInput` activity to deliver the input to the sidecar via gRPC
- **AND** the workflow SHALL resume monitoring agent status

#### Scenario: Human input received while agent is not waiting
- **WHEN** the workflow receives a `human-input` signal but the agent is not in a `WaitingForInput` state
- **THEN** the workflow SHALL buffer the input and deliver it when the agent next requests input

### Requirement: TTL Enforcement
The `AgentRunWorkflow` SHALL enforce TTL via `workflow.NewTimer` and stop the agent when the TTL expires.

#### Scenario: Agent exceeds TTL
- **WHEN** the workflow's TTL timer fires before the agent completes
- **THEN** the workflow SHALL execute `StopAgent` to gracefully terminate the agent
- **AND** the workflow SHALL execute `CleanupPod` to remove the agent pod
- **AND** the workflow SHALL complete with a failure result indicating TTL expiration

### Requirement: Cancel Signal Handling
The `AgentRunWorkflow` SHALL handle `cancel` signals by stopping the agent and cleaning up resources.

#### Scenario: Cancelling a running agent
- **WHEN** the workflow receives a `cancel` signal
- **THEN** the workflow SHALL execute `StopAgent` to gracefully terminate the agent
- **AND** the workflow SHALL execute `CleanupPod` to remove the agent pod
- **AND** the workflow SHALL complete with a cancelled result

### Requirement: Guaranteed Cleanup
The `AgentRunWorkflow` SHALL clean up the agent pod on any terminal outcome (success, failure, or cancellation) via the `CleanupPod` activity.

#### Scenario: Cleanup on workflow completion
- **WHEN** the workflow reaches a terminal state for any reason
- **THEN** the workflow SHALL execute `CleanupPod` to delete the Kubernetes Pod
- **AND** the `CleanupPod` activity SHALL be idempotent (no error if pod already deleted)

### Requirement: Workflow State Query
The `AgentRunWorkflow` SHALL support a `get-state` query that returns the current phase, message, and pod name.

#### Scenario: Querying workflow state
- **WHEN** a `get-state` query is received by the workflow
- **THEN** the workflow SHALL return the current `AgentRunPhase`, a human-readable message, and the pod name
- **AND** the response SHALL reflect the most recent state transition

### Requirement: Child Workflow for spawn_junior
`spawn_junior` SHALL be implemented as a child workflow of `AgentRunWorkflow`.

#### Scenario: Spawning a junior agent with blocking wait
- **WHEN** the parent workflow spawns a junior agent with `await: true`
- **THEN** the parent workflow SHALL start a child `AgentRunWorkflow` and block until the child completes
- **AND** the child's result SHALL be available to the parent workflow

#### Scenario: Spawning a junior agent with fire-and-forget
- **WHEN** the parent workflow spawns a junior agent with `await: false`
- **THEN** the parent workflow SHALL start a child `AgentRunWorkflow` and continue execution immediately
- **AND** the child workflow SHALL continue executing independently of the parent

### Requirement: Parent Workflow Awareness of Child Workflows
The parent workflow SHALL be able to await child workflow completion or proceed independently.

#### Scenario: Parent cancelled while child is running
- **WHEN** the parent workflow receives a `cancel` signal while a child workflow is running
- **THEN** the parent workflow SHALL cancel all child workflows
- **AND** child workflows SHALL execute their own cleanup (CleanupPod)

### Requirement: Activity Idempotency
All activities SHALL be idempotent to support safe retries by the Temporal framework.

#### Scenario: Retrying CreateAgentPod after transient failure
- **WHEN** the `CreateAgentPod` activity is retried after a transient API server error
- **THEN** the activity SHALL check if the pod already exists before creating it
- **AND** the activity SHALL return success if the pod already exists with the correct spec

#### Scenario: Retrying CleanupPod after transient failure
- **WHEN** the `CleanupPod` activity is retried after a transient error
- **THEN** the activity SHALL return success if the pod is already deleted (NotFound)

### Requirement: Activity Retry Policies
All activities SHALL have configurable retry policies with exponential backoff.

#### Scenario: Transient Kubernetes API failure
- **WHEN** a pod management activity fails due to a transient Kubernetes API error
- **THEN** the Temporal framework SHALL retry the activity with exponential backoff
- **AND** the retry policy SHALL be configurable per activity type

#### Scenario: Non-retryable failure
- **WHEN** an activity fails with a non-retryable error (e.g., invalid spec, permission denied)
- **THEN** the activity SHALL wrap the error as a non-retryable `ApplicationError`
- **AND** the Temporal framework SHALL NOT retry the activity
