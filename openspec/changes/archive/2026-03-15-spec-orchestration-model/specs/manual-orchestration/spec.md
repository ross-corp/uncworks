## ADDED Requirements

### Requirement: User-defined orchestration tree in spec
When `orchestrationMode` is `manual`, the AgentRunSpec SHALL include an `orchestration` field containing a list of tasks. Each task SHALL have a `name` and `prompt`. Optionally, each task MAY specify `repoUrls` to restrict which repos are cloned into the junior's workspace.

#### Scenario: Manual orchestration with three tasks
- **GIVEN** an AgentRun with `orchestrationMode=manual` and `orchestration.tasks` containing three entries
- **WHEN** the workflow starts
- **THEN** it spawns three junior AgentRuns, one per task
- **AND** each junior receives the task's `prompt` as its prompt
- **AND** no senior agent is started for decomposition

#### Scenario: Manual orchestration with repo scoping
- **GIVEN** a task in `orchestration.tasks` with `repoUrls: ["https://github.com/org/frontend"]`
- **WHEN** the junior AgentRun is created
- **THEN** it clones only the specified repo, not all repos from the parent spec

#### Scenario: Manual orchestration with empty tasks
- **WHEN** an AgentRun has `orchestrationMode=manual` but `orchestration.tasks` is empty
- **THEN** validation rejects the CRD with error "manual orchestration requires at least one task"

### Requirement: Manual mode skips senior agent
In `manual` mode, the workflow SHALL NOT start a senior agent for decomposition or integration. It SHALL spawn juniors directly from the user-defined task list and wait for all to complete.

#### Scenario: All manual juniors complete
- **WHEN** all junior AgentRuns reach a terminal state
- **THEN** the senior AgentRun transitions to `Succeeded` if all juniors succeeded
- **AND** transitions to `Failed` if any junior failed, with a message listing failed tasks

#### Scenario: Manual mode has no integration step
- **WHEN** manual orchestration completes
- **THEN** no integration agent is started
- **AND** junior workspaces are preserved for user review

### Requirement: Orchestration field validation
The `orchestration` field SHALL be validated: task names must be unique, non-empty, and match `^[a-z0-9-]+$`. Task prompts must be non-empty. The maximum number of tasks is 7.

#### Scenario: Duplicate task names
- **WHEN** an AgentRun has `orchestration.tasks` with two tasks named "fix-auth"
- **THEN** validation rejects the CRD with error "duplicate task name: fix-auth"

#### Scenario: Task name with invalid characters
- **WHEN** a task name contains spaces or uppercase letters
- **THEN** validation rejects the CRD with a descriptive error

#### Scenario: More than 7 tasks
- **WHEN** `orchestration.tasks` contains 8 entries
- **THEN** validation rejects the CRD with error "maximum 7 orchestration tasks allowed"
