## ADDED Requirements

### Requirement: CreateAgentRun creates a K8s CRD
The API server SHALL create an AgentRun CRD in Kubernetes when `CreateAgentRun` is called. The CRD SHALL contain all spec fields from the request. The API SHALL return immediately after CRD creation with phase Pending.

#### Scenario: Successful creation via API
- **WHEN** a client calls `CreateAgentRun` with a valid spec (backend, repoURL, prompt)
- **THEN** an AgentRun CRD is created in the configured namespace with a generated name
- **AND** the response contains the CRD name as the run ID
- **AND** the status phase is Pending

#### Scenario: Invalid spec rejected
- **WHEN** a client calls `CreateAgentRun` with missing required fields
- **THEN** the API returns an InvalidArgument error
- **AND** no CRD is created

### Requirement: ListAgentRuns reads from K8s
The API server SHALL list AgentRun CRDs from Kubernetes when `ListAgentRuns` is called. Results SHALL be sorted by creation timestamp descending.

#### Scenario: List returns all runs
- **WHEN** a client calls `ListAgentRuns` with no filters
- **THEN** all AgentRun CRDs in the namespace are returned
- **AND** results are ordered by creation time (newest first)

#### Scenario: List with phase filter
- **WHEN** a client calls `ListAgentRuns` with a phase filter
- **THEN** only AgentRun CRDs matching that phase are returned

### Requirement: GetAgentRun reads from K8s with Temporal enrichment
The API server SHALL read AgentRun CRDs from Kubernetes and enrich the status with live Temporal workflow state when available.

#### Scenario: Get existing run
- **WHEN** a client calls `GetAgentRun` with a valid ID
- **THEN** the AgentRun CRD is returned with status enriched from Temporal workflow state
- **AND** all status fields are populated (phase, message, podName, startedAt, completedAt)

#### Scenario: Get non-existent run
- **WHEN** a client calls `GetAgentRun` with an ID that does not exist
- **THEN** the API returns a NotFound error

### Requirement: CancelAgentRun cancels via Temporal
The API server SHALL cancel an AgentRun by signaling the Temporal workflow. The CRD status update SHALL be handled by the controller's reconciliation loop, not the API server directly.

#### Scenario: Cancel a running agent
- **WHEN** a client calls `CancelAgentRun` for a running agent
- **THEN** the Temporal workflow receives a cancel signal
- **AND** the API returns success
- **AND** the controller eventually updates the CRD phase to Cancelled

### Requirement: WatchAgentRun streams real-time events
The API server SHALL stream AgentRun state changes to clients via the EventBus. Events SHALL include phase transitions, output, and errors.

#### Scenario: Watch receives phase transitions
- **WHEN** a client calls `WatchAgentRun` for an existing run
- **AND** the run transitions from Pending to Running to Succeeded
- **THEN** the client receives events for each phase transition

### Requirement: API server has K8s RBAC for AgentRun CRDs
The API server deployment SHALL use a ServiceAccount with permissions to create, get, list, and watch AgentRun CRDs.

#### Scenario: API server can create CRDs
- **WHEN** the API server is deployed with the Helm chart
- **THEN** it has RBAC permissions to create, get, list, and watch agentruns.aot.uncworks.io resources
