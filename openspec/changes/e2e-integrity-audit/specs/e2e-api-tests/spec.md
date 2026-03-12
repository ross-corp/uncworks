## ADDED Requirements

### Requirement: E2E test creates run via API and verifies CRD
An E2E test SHALL create an AgentRun via the gRPC API and verify that a corresponding K8s CRD is created with the correct spec fields.

#### Scenario: API creation produces CRD
- **WHEN** the E2E test calls `CreateAgentRun` via ConnectRPC
- **THEN** an AgentRun CRD exists in the cluster with matching repoURL, prompt, and backend
- **AND** the CRD name matches the ID returned by the API

### Requirement: E2E test verifies full lifecycle via API
An E2E test SHALL create a run via the API and verify it progresses through Pending → Running → Succeeded, with the agent pod actually executing.

#### Scenario: Full lifecycle via API
- **WHEN** the E2E test creates a run via `CreateAgentRun` with a simple prompt
- **THEN** `GetAgentRun` eventually returns phase Running
- **AND** `GetAgentRun` eventually returns phase Succeeded
- **AND** an agent pod was created and completed in the cluster

### Requirement: E2E test verifies cancellation via API
An E2E test SHALL create a run via the API, cancel it, and verify the run reaches Cancelled phase.

#### Scenario: Cancel via API
- **WHEN** the E2E test creates a run and waits for Running phase
- **AND** calls `CancelAgentRun`
- **THEN** `GetAgentRun` eventually returns phase Cancelled

### Requirement: E2E test verifies human input via API
An E2E test SHALL create a run that requires human input, send input via the API, and verify the run completes.

#### Scenario: HITL via API
- **WHEN** the E2E test creates a run with a HITL prompt
- **AND** waits for WaitingForInput phase
- **AND** calls `SendHumanInput`
- **THEN** `GetAgentRun` eventually returns phase Succeeded

### Requirement: Temporal workflow executes activities without errors
The Temporal workflow SHALL execute all activities (ProvisionLLMKey, CreateAgentPod, WaitForHydration, StartAgent, GetAgentStatus, CleanupPod, RevokeLLMKey) without nil pointer or invocation errors.

#### Scenario: Activity execution succeeds
- **WHEN** a workflow is started by the controller
- **THEN** all activities execute via the registered worker
- **AND** no nil pointer or "activity not found" errors occur

### Requirement: Sidecar streams both stdout and stderr
The sidecar SHALL stream both stdout and stderr from the agent process to connected clients.

#### Scenario: Stderr is captured
- **WHEN** the agent process writes to stderr
- **THEN** the output appears in the stream with type STDERR

### Requirement: Proto types match CRD types for all user-facing fields
The proto `AgentRunSpec` and `AgentRunStatus` messages SHALL include all fields present in the K8s CRD types that are relevant to API consumers.

#### Scenario: Proto has image field
- **WHEN** a client creates a run with a custom `image` field
- **THEN** the CRD is created with that image override

#### Scenario: Proto status includes timestamps
- **WHEN** a client calls `GetAgentRun` for a completed run
- **THEN** the response includes `started_at` and `completed_at` timestamps
