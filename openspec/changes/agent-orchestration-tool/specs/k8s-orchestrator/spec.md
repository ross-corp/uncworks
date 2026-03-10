## ADDED Requirements

### Requirement: Multi-Backend AgentRun Support
The `AgentRun` CRD SHALL support multiple execution backends via its `spec.backend` field.

#### Scenario: Provisioning a Pod-based AgentRun
- **WHEN** an `AgentRun` is created with `spec.backend: "pod"`
- **THEN** the Orchestrator SHALL schedule a standard Kubernetes Pod

#### Scenario: Validating KubeVirt and External Stubs
- **WHEN** an `AgentRun` is created with `spec.backend: "kubevirt"` OR `"external"`
- **THEN** the Orchestrator SHALL initially reject the request with a "Not Yet Implemented" error while maintaining the CRD schema for future support

### Requirement: AgentRun CRD Management
The system SHALL manage the lifecycle of `AgentRun` Custom Resource Definitions (CRDs) in Kubernetes.

#### Scenario: Successful creation of an AgentRun
- **WHEN** a new `AgentRun` CRD is submitted via the API
- **THEN** the Orchestrator SHALL schedule a Pod with the appropriate sidecars and resource limits

#### Scenario: Termination of an AgentRun
- **WHEN** an `AgentRun` reaches its TTL or is marked as complete
- **THEN** the Orchestrator SHALL safely terminate the Pod and clean up ephemeral resources

### Requirement: Job Queuing and Worker Limits
The system SHALL enforce global and per-user limits on the number of active agent workers.

#### Scenario: Queueing when limit is reached
- **WHEN** the `max_parallel_agents` limit is reached
- **THEN** new `AgentRun` requests SHALL be placed in a `Pending` state in the Postgres database until a slot becomes available
