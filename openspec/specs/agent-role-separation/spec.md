# agent-role-separation Specification

## Purpose
TBD - created by archiving change agent-architecture-v2. Update Purpose after archive.
## Requirements
### Requirement: agent-manage role
The system SHALL provide an agent-manage role that orchestrates the spec-driven pipeline, owns OpenSpec artifacts, and delegates implementation to agent-implement.

#### Scenario: manage agent uses openspec CLI
- **WHEN** agent-manage runs during the plan stage
- **THEN** it SHALL use `openspec instructions`, `openspec validate`, and `openspec status` CLI commands to create and validate spec artifacts

#### Scenario: manage agent cannot write code
- **WHEN** agent-manage attempts to write or edit a file inside a repo worktree
- **THEN** the determinism extension SHALL block the tool call with a policy violation

#### Scenario: manage agent asks user questions
- **WHEN** agent-manage needs clarification
- **THEN** it SHALL use the `ask_user` tool to pause and elicit input from the user

### Requirement: agent-implement role
The system SHALL provide an agent-implement role that reads specs and implements code changes in the repo.

#### Scenario: implement agent reads specs
- **WHEN** agent-implement starts during the execute stage
- **THEN** it SHALL read spec artifacts from `/workspace/.openspec/changes/<name>/` to understand what to implement

#### Scenario: implement agent marks tasks
- **WHEN** agent-implement completes a task
- **THEN** it SHALL mark the corresponding checkbox in `/workspace/.openspec/changes/<name>/tasks.md` as `[x]`

#### Scenario: implement agent cannot modify specs
- **WHEN** agent-implement attempts to run openspec CLI commands or write to `.openspec/`
- **THEN** the determinism extension SHALL block the tool call

#### Scenario: implement agent escalates questions
- **WHEN** agent-implement needs clarification it cannot resolve from the specs
- **THEN** it SHALL include the question in its tool result output for agent-manage to surface to the user

### Requirement: role labels in UI
The system SHALL display "manage" and "impl" labels in the activity feed instead of "unc" and "neph".

#### Scenario: activity feed labels
- **WHEN** viewing the activity feed for a spec-driven run
- **THEN** entries from agent-manage SHALL display with label "manage" in blue, and entries from agent-implement SHALL display with label "impl" in green

