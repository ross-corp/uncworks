# deterministic-policy Specification

## Purpose
TBD - created by archiving change agent-architecture-v2. Update Purpose after archive.
## Requirements
### Requirement: role-based tool policies
The system SHALL enforce deterministic tool access policies based on the agent's role (manage vs implement).

#### Scenario: manage agent blocked from code writes
- **WHEN** agent-manage calls the `write` tool targeting a repo file
- **THEN** the determinism extension SHALL return `{block: true}` with reason "manage agents cannot write code"

#### Scenario: implement agent blocked from openspec
- **WHEN** agent-implement calls `bash` with an `openspec` command
- **THEN** the determinism extension SHALL return `{block: true}` with reason "implement agents cannot modify specs"

#### Scenario: implement agent blocked from ask_user
- **WHEN** agent-implement calls `ask_user`
- **THEN** the determinism extension SHALL return `{block: true}` with reason "implement agents must escalate questions through tool results"

### Requirement: loop detection
The system SHALL detect and break infinite tool call loops.

#### Scenario: repeated identical tool calls
- **WHEN** an agent makes 3 identical consecutive tool calls (same tool name and input hash)
- **THEN** the extension SHALL block the call and return a reason explaining the loop

### Requirement: turn limit
The system SHALL enforce a maximum turn count per agent invocation.

#### Scenario: turn limit exceeded
- **WHEN** an agent exceeds 50 turns
- **THEN** the extension SHALL block further turns with reason "turn limit exceeded"

### Requirement: spec format enforcement
The system SHALL validate OpenSpec artifacts written by agent-manage during the plan stage.

#### Scenario: missing SHALL/MUST keywords
- **WHEN** agent-manage writes a spec requirement without SHALL or MUST
- **THEN** the extension SHALL block the write with reason explaining the requirement

#### Scenario: excessive tasks
- **WHEN** agent-manage writes tasks.md with more than 30 checkboxes
- **THEN** the extension SHALL block the write with reason to keep tasks proportional

