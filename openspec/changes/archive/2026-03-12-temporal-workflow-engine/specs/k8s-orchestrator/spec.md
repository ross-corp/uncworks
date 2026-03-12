## MODIFIED Requirements

### Requirement: AgentRun CRD Management (MODIFIED)
The controller SHALL act as a bridge between AgentRun CRDs and Temporal workflows, delegating lifecycle execution to Temporal.

#### Scenario: New AgentRun CRD without workflow ID annotation
- **WHEN** an `AgentRun` CRD is created or reconciled and does NOT have a `aot.uncworks.io/workflow-id` annotation
- **THEN** the controller SHALL start a new `AgentRunWorkflow` in Temporal with the CRD's spec as input
- **AND** the controller SHALL annotate the CRD with `aot.uncworks.io/workflow-id` set to the Temporal workflow ID
- **AND** the controller SHALL annotate the CRD with `aot.uncworks.io/workflow-run-id` set to the Temporal run ID

#### Scenario: Existing AgentRun CRD with workflow ID annotation
- **WHEN** an `AgentRun` CRD is reconciled and HAS a `aot.uncworks.io/workflow-id` annotation
- **THEN** the controller SHALL query the Temporal workflow's `get-state` query handler
- **AND** the controller SHALL sync the returned phase, message, and pod name to the CRD's status fields
- **AND** the controller SHALL requeue reconciliation after 30 seconds if the workflow is not in a terminal state

#### Scenario: AgentRun CRD deletion
- **WHEN** an `AgentRun` CRD is deleted
- **THEN** the controller SHALL cancel the associated Temporal workflow using the workflow ID from the annotation
- **AND** the workflow's cleanup logic (CleanupPod) SHALL handle pod deletion

### Requirement: Multi-Backend AgentRun Support (UNCHANGED)
The `AgentRun` CRD SHALL support multiple execution backends via its `spec.backend` field.

#### Scenario: Provisioning a Pod-based AgentRun
- **WHEN** an `AgentRun` is created with `spec.backend: "pod"`
- **THEN** the Orchestrator SHALL start a Temporal workflow for pod-based execution

#### Scenario: Validating KubeVirt and External Stubs
- **WHEN** an `AgentRun` is created with `spec.backend: "kubevirt"` OR `"external"`
- **THEN** the Orchestrator SHALL initially reject the request with a "Not Yet Implemented" error while maintaining the CRD schema for future support

### Requirement: Pod Creation Delegation (MODIFIED)
Pod creation logic moves from the controller reconcile loop to Temporal activities, but still uses a controller-runtime K8s client.

#### Scenario: Pod creation via Temporal activity
- **WHEN** the `AgentRunWorkflow` executes the `CreateAgentPod` activity
- **THEN** the activity SHALL use a controller-runtime K8s client to create the pod
- **AND** the pod spec SHALL match the existing pod structure (agent container, RPC Gateway sidecar, hydration init container)
- **AND** the controller SHALL no longer create pods directly in its reconcile loop

### Requirement: Job Queuing and Worker Limits (MODIFIED)
Job queuing is replaced by Temporal's native task queue and workflow scheduling.

#### Scenario: Queueing when limit is reached
- **WHEN** the `max_parallel_agents` limit is reached
- **THEN** concurrency control SHALL be enforced at the Temporal worker level via `MaxConcurrentWorkflowTaskExecutionSize` and `MaxConcurrentActivityExecutionSize` configuration
- **AND** excess workflows SHALL queue naturally in the Temporal task queue until worker capacity is available

### Requirement: TTL Enforcement Delegation (MODIFIED)
TTL enforcement moves from the controller reconcile loop to the Temporal workflow.

#### Scenario: TTL enforcement via Temporal timer
- **WHEN** an `AgentRun` has a configured `ttl_seconds`
- **THEN** the `AgentRunWorkflow` SHALL enforce the TTL via `workflow.NewTimer`
- **AND** the controller SHALL no longer check TTL in its reconcile loop
