# subagent-visibility Specification

## Purpose
TBD - created by archiving change agent-architecture-v2. Update Purpose after archive.
## Requirements
### Requirement: subagent spawn tracking
The system SHALL track when agents spawn subagents and expose this in the activity feed and traces.

#### Scenario: subagent appears in activity feed
- **WHEN** agent-manage or agent-implement spawns a subagent
- **THEN** the activity feed SHALL display a "subagent started" entry with the subagent's task description

#### Scenario: subagent appears in traces
- **WHEN** a subagent completes
- **THEN** the trace timeline SHALL show the subagent as a child span of the parent agent's span

#### Scenario: subagent completion in activity
- **WHEN** a subagent finishes (success or failure)
- **THEN** the activity feed SHALL display its result with status indication (green for success, red for failure)

### Requirement: subagent tree display
The system SHALL display subagent relationships as a tree in the run detail view.

#### Scenario: nested subagent visibility
- **WHEN** viewing a run where agents spawned subagents
- **THEN** the UI SHALL show the agent hierarchy with indentation in the activity feed

