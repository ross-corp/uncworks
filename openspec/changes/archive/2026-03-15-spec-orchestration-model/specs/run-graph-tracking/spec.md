## ADDED Requirements

### Requirement: Parent-child relationship on AgentRunSpec
AgentRunSpec SHALL include a `parentRunID` field (proto: `parent_run_id`, CRD: `parentRunID`). When set, the AgentRun is a junior run whose parent is the referenced AgentRun. The field is empty for root/senior runs and for single-mode runs.

#### Scenario: Junior run references its parent
- **WHEN** a junior AgentRun is created by the orchestration workflow
- **THEN** its `parentRunID` field contains the senior AgentRun's name
- **AND** the parent AgentRun can be queried by name

#### Scenario: Single-mode run has no parent
- **WHEN** an AgentRun is created with `orchestrationMode=single`
- **THEN** its `parentRunID` is empty

### Requirement: Run graph queryable via API
The `GetAgentRun` response SHALL include a `children` field listing the names of all junior AgentRuns (queried by `parent_run_id` match). The `ListAgentRuns` response SHALL support `spec_run_id` and `parent_run_id` filters.

#### Scenario: Get senior run with children
- **WHEN** `GetAgentRun` is called for a senior run that spawned 3 juniors
- **THEN** the response includes `children: ["junior-1", "junior-2", "junior-3"]`

#### Scenario: List runs filtered by parent
- **WHEN** `ListAgentRuns` is called with `parent_run_id` = "my-senior-run"
- **THEN** only direct children of "my-senior-run" are returned

### Requirement: Run graph visualization data
The API SHALL provide a `GetRunGraph` RPC that returns the full tree structure for a spec-run. The response SHALL include nodes (AgentRun name, phase, role, duration) and edges (parent-child relationships).

#### Scenario: Fetch run graph for a completed orchestration
- **WHEN** `GetRunGraph` is called with the senior run's ID
- **THEN** the response contains one node per AgentRun in the spec execution
- **AND** each node includes `name`, `phase`, `role` (senior/junior), `startedAt`, `completedAt`
- **AND** edges connect each junior to its parent

#### Scenario: Fetch run graph for a single-mode run
- **WHEN** `GetRunGraph` is called for a run with no children and no parent
- **THEN** the response contains a single node with no edges

### Requirement: UI displays run graph
The run detail page SHALL display a tree visualization when the run has children or a parent. The tree SHALL show each node with a phase badge (color-coded), name, and duration. Clicking a node SHALL navigate to that run's detail page.

#### Scenario: Viewing a senior run in the UI
- **WHEN** a user opens the detail page for a senior run with 5 juniors
- **THEN** the page shows a tree with the senior as root and 5 junior nodes
- **AND** each node shows its current phase as a colored badge
- **AND** completed juniors show their duration

#### Scenario: Viewing a junior run in the UI
- **WHEN** a user opens the detail page for a junior run
- **THEN** a breadcrumb shows the path: senior run name > junior run name
- **AND** a link navigates back to the senior run

#### Scenario: Progress indication
- **WHEN** an orchestrated run is in progress with 3 of 5 juniors completed
- **THEN** the tree shows 3 nodes with success badges and 2 with running badges
- **AND** a summary line shows "3/5 tasks complete"
