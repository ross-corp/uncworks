## ADDED Requirements

### Requirement: SpecRun grouping via labels and annotations
AgentRun CRDs participating in orchestrated execution SHALL carry a `aot.uncworks.io/spec-run-id` label set to the senior AgentRun's name. AgentRuns SHALL carry a `aot.uncworks.io/run-role` label with value `senior` or `junior`. Junior AgentRuns SHALL carry a `aot.uncworks.io/parent-run` annotation with the parent AgentRun's name.

#### Scenario: Senior run is created with orchestration
- **WHEN** an AgentRun is created with `orchestrationMode` set to `auto` or `manual`
- **THEN** the controller sets label `aot.uncworks.io/spec-run-id` to the AgentRun's own name
- **AND** the controller sets label `aot.uncworks.io/run-role` to `senior`

#### Scenario: Junior run is spawned by a senior
- **WHEN** the senior workflow spawns a junior AgentRun
- **THEN** the junior's `parentRunID` field is set to the senior's name
- **AND** the controller sets label `aot.uncworks.io/spec-run-id` to the senior's name
- **AND** the controller sets label `aot.uncworks.io/run-role` to `junior`
- **AND** the controller sets annotation `aot.uncworks.io/parent-run` to the senior's name

#### Scenario: Single-mode run has no orchestration labels
- **WHEN** an AgentRun is created with `orchestrationMode` unset or `single`
- **THEN** no `spec-run-id` or `run-role` labels are set
- **AND** the run behaves identically to current behavior

### Requirement: Orchestration mode field on AgentRunSpec
AgentRunSpec SHALL include an `orchestrationMode` field with values `single` (default), `auto`, or `manual`. The field SHALL default to `single` when unspecified, preserving backward compatibility.

#### Scenario: Omitted orchestration mode
- **WHEN** an AgentRun is created without `orchestrationMode`
- **THEN** it defaults to `single`
- **AND** the workflow executes as a single agent run with no decomposition

#### Scenario: Invalid orchestration mode
- **WHEN** an AgentRun is created with an unrecognized `orchestrationMode` value
- **THEN** validation rejects the CRD with a descriptive error

### Requirement: ListAgentRuns supports spec-run-id filter
The `ListAgentRuns` RPC SHALL accept an optional `spec_run_id` filter. When provided, it SHALL return only AgentRuns with the matching `aot.uncworks.io/spec-run-id` label.

#### Scenario: Filter runs by spec-run-id
- **WHEN** `ListAgentRuns` is called with `spec_run_id` = "my-senior-run"
- **THEN** only AgentRuns with label `aot.uncworks.io/spec-run-id=my-senior-run` are returned
- **AND** the result includes both the senior and all its juniors
