## ADDED Requirements

### Requirement: ChainRun CRD tracks chain execution state
The system SHALL provide a ChainRun custom resource that represents a single execution of a Chain. The ChainRun SHALL track per-step status, the overall execution phase, and timing information. A ChainRun SHALL be backed by a Temporal workflow.

#### Scenario: Create a ChainRun from a Chain
- **WHEN** a user triggers a Chain (manually or via Schedule)
- **THEN** the system creates a ChainRun with a reference to the Chain
- **AND** the ChainRun status initializes all steps to phase "Pending"
- **AND** the system starts a Temporal workflow with workflow ID matching the ChainRun name

#### Scenario: ChainRun tracks step phases
- **WHEN** a ChainRun is executing and step A completes successfully
- **THEN** the ChainRun status updates step A's phase to "Succeeded"
- **AND** step A's agentRunRef is set to the name of the created AgentRun
- **AND** step A's completedAt timestamp is recorded

#### Scenario: ChainRun overall phase reflects step states
- **WHEN** all steps in a ChainRun have phase "Succeeded"
- **THEN** the ChainRun overall phase transitions to "Succeeded"
- **AND** the ChainRun's completedAt timestamp is recorded

#### Scenario: ChainRun fails when a step fails
- **WHEN** step A fails and downstream steps B and C depend on A
- **THEN** step A's phase transitions to "Failed"
- **AND** steps B and C transition to "Skipped" (their dependencies cannot be satisfied)
- **AND** the ChainRun overall phase transitions to "Failed"

### Requirement: ChainRun DAG executor via Temporal
The chain controller SHALL implement DAG execution using a Temporal workflow. The workflow SHALL perform topological sort of steps, execute ready steps as child workflows (AgentRunWorkflow), and manage fan-out/fan-in synchronization.

#### Scenario: Execute root steps in parallel
- **WHEN** a ChainRun starts and the Chain has root steps A and B (no dependencies)
- **THEN** the Temporal workflow launches AgentRunWorkflow child workflows for A and B simultaneously
- **AND** both steps' phases transition to "Running"

#### Scenario: Execute dependent step after parent succeeds
- **WHEN** step A succeeds and step B dependsOn [A]
- **THEN** the Temporal workflow creates an AgentRun for step B with context from step A
- **AND** step B's phase transitions to "Running"

#### Scenario: Fan-in waits for all parents
- **WHEN** step C dependsOn [A, B] and step A has succeeded but step B is still running
- **THEN** step C remains in phase "Pending"
- **AND** step C transitions to "Running" only after both A and B succeed

#### Scenario: Context injection during step creation
- **WHEN** step B has `contextFrom: ["A"]` and step A's AgentRun has log output
- **THEN** the workflow reads step A's AgentRun log output from the status
- **AND** prepends it to step B's prompt as a context section
- **AND** creates the AgentRun with the enriched prompt

#### Scenario: Branch propagation during step creation
- **WHEN** step B has `branchFrom: "A"` and step A pushed to branch "chainrun-xyz-step-a"
- **THEN** the workflow reads the branch name from step A's AgentRun status
- **AND** creates step B's AgentRun with repos[].branch set to step A's branch

#### Scenario: Cancel a running ChainRun
- **WHEN** a user sends a cancel signal to a ChainRun
- **THEN** the Temporal workflow cancels all currently running child workflows
- **AND** all running steps transition to "Cancelled"
- **AND** all pending steps transition to "Skipped"
- **AND** the ChainRun overall phase transitions to "Cancelled"

### Requirement: ChainRun CRUD and trigger API
The REST API SHALL expose endpoints for listing, getting, and cancelling ChainRuns, and for triggering a Chain to create a new ChainRun.

#### Scenario: Trigger a chain
- **WHEN** a user calls POST /api/v1/chains/{name}/trigger
- **THEN** the system creates a ChainRun referencing the Chain
- **AND** returns the ChainRun name and status

#### Scenario: List ChainRuns
- **WHEN** a user calls GET /api/v1/chain-runs
- **THEN** the system returns all ChainRuns with per-step status summaries

#### Scenario: List ChainRuns filtered by chain
- **WHEN** a user calls GET /api/v1/chain-runs?chain=my-chain
- **THEN** the system returns only ChainRuns whose spec.chainRef matches "my-chain"

#### Scenario: Get ChainRun detail
- **WHEN** a user calls GET /api/v1/chain-runs/{name}
- **THEN** the system returns the full ChainRun including per-step status, agentRunRefs, timing, and phase

#### Scenario: Cancel a ChainRun
- **WHEN** a user calls POST /api/v1/chain-runs/{name}/cancel
- **THEN** the system sends a cancel signal to the Temporal workflow
- **AND** the ChainRun transitions to "Cancelling" then "Cancelled"
