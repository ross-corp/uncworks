## ADDED Requirements

### Requirement: Chain CRD defines a DAG of RunTemplate steps
The system SHALL provide a Chain custom resource that defines an ordered DAG of steps. Each step SHALL reference a RunTemplate by name and declare zero or more dependencies via a `dependsOn` field. Steps with no dependencies are root steps and SHALL execute first.

#### Scenario: Create a linear chain (A -> B -> C)
- **WHEN** a user creates a Chain with three steps where step B dependsOn [A] and step C dependsOn [B]
- **THEN** the system persists the Chain
- **AND** the Chain status shows phase "Ready" with stepCount 3

#### Scenario: Create a fan-out chain (A -> B, A -> C)
- **WHEN** a user creates a Chain where steps B and C both dependsOn [A]
- **THEN** the system persists the Chain
- **AND** when triggered, steps B and C SHALL execute in parallel after A succeeds

#### Scenario: Create a fan-in chain (A, B -> C)
- **WHEN** a user creates a Chain where step C dependsOn [A, B]
- **THEN** the system persists the Chain
- **AND** when triggered, step C SHALL wait for both A and B to succeed before executing

#### Scenario: Create a diamond DAG (A -> B, A -> C, B+C -> D)
- **WHEN** a user creates a Chain with four steps forming a diamond pattern
- **THEN** the system persists the Chain
- **AND** topological sort yields execution order: A first, then B and C in parallel, then D

### Requirement: Chain DAG validation
The system SHALL validate the Chain DAG on create and update. The system SHALL reject Chains that contain cycles, reference undefined steps, or reference nonexistent RunTemplates.

#### Scenario: Reject a chain with a cycle
- **WHEN** a user creates a Chain where step A dependsOn [B] and step B dependsOn [A]
- **THEN** the API returns a validation error: "chain contains a cycle involving steps: A, B"

#### Scenario: Reject a chain with undefined dependency
- **WHEN** a user creates a Chain where step B dependsOn ["nonexistent"]
- **THEN** the API returns a validation error: "step B depends on undefined step: nonexistent"

#### Scenario: Reject a chain referencing missing RunTemplate
- **WHEN** a user creates a Chain where step A references RunTemplate "deleted-template" which does not exist
- **THEN** the API returns a validation error: "step A references nonexistent RunTemplate: deleted-template"

### Requirement: Context passing between chain steps
Each chain step SHALL optionally declare `contextFrom` and `branchFrom` fields to pass execution context from parent steps.

#### Scenario: contextFrom injects parent output into child prompt
- **WHEN** step B declares `contextFrom: ["A"]` and step A's AgentRun succeeds with log output
- **THEN** the system prepends step A's log output summary to step B's prompt
- **AND** step B's agent can reference the parent's findings in its work

#### Scenario: branchFrom clones from parent's feature branch
- **WHEN** step B declares `branchFrom: "A"` and step A pushed changes to branch "feature/step-a"
- **THEN** the system creates step B's AgentRun with repos configured to clone from step A's branch
- **AND** step B starts with step A's code changes already in the workspace

#### Scenario: contextFrom with multiple parents
- **WHEN** step C declares `contextFrom: ["A", "B"]` (fan-in)
- **THEN** the system concatenates log output summaries from both A and B into step C's prompt
- **AND** each parent's output is labeled with the step name

#### Scenario: branchFrom with fan-in is rejected
- **WHEN** a user creates a chain where step C dependsOn [A, B] and declares `branchFrom: "A"` AND step C also has `contextFrom: ["B"]`
- **THEN** the system allows this configuration because branchFrom references a single step
- **AND** step C clones from A's branch and receives B's output as prompt context

### Requirement: Chain CRUD API
The REST API SHALL expose endpoints for creating, listing, getting, updating, and deleting Chains.

#### Scenario: List Chains
- **WHEN** a user calls GET /api/v1/chains
- **THEN** the system returns all Chains with their step definitions and status

#### Scenario: List Chains filtered by project
- **WHEN** a user calls GET /api/v1/chains?project=my-project
- **THEN** the system returns only Chains whose spec.projectRef matches "my-project"

#### Scenario: Delete a Chain referenced by a Schedule
- **WHEN** a user calls DELETE /api/v1/chains/{name} and the chain is referenced by an active Schedule
- **THEN** the API returns a 409 Conflict error listing the referencing Schedules
