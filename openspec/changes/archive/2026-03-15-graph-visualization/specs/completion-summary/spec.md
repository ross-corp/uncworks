## ADDED Requirements

### Requirement: Completion summary appears when spec run finishes
When all agents in a spec run reach terminal phases, a completion summary panel SHALL replace the orchestration graph view with aggregated results.

#### Scenario: All agents succeed
- **WHEN** all agents in the spec run have phase Succeeded
- **THEN** the completion summary panel appears with a "SPEC COMPLETE — ALL SYSTEMS NOMINAL" banner
- **AND** the banner text types out character-by-character using TerminalBoot animation

#### Scenario: Some agents failed
- **WHEN** at least one agent has phase Failed and all agents are in terminal phases
- **THEN** the completion summary panel appears with a "SPEC COMPLETE — FAILURES DETECTED" banner in amber
- **AND** failed agents are listed first in the results table

#### Scenario: User can return to graph view
- **WHEN** the completion summary is shown
- **THEN** a "VIEW GRAPH" button allows the user to switch back to the orchestration graph view

### Requirement: Agent results table
The completion summary SHALL include a table of all agent runs with their outcomes.

#### Scenario: Results table content
- **WHEN** the completion summary is displayed
- **THEN** the table shows each agent's: run ID, agent type (senior/junior), final phase, duration, and files changed count
- **AND** rows are sorted by agent type (senior first) then by start time
- **AND** failed agents have amber text, succeeded agents have green text

### Requirement: Aggregated diff list
The completion summary SHALL include an expandable list of all files modified across all agents in the spec run.

#### Scenario: Files list display
- **WHEN** the completion summary is displayed
- **THEN** a list of all modified files is shown, grouped by agent
- **AND** each file entry shows the file path, lines added (green), and lines removed (red)

#### Scenario: Click file opens diff viewer
- **WHEN** user clicks a file entry in the aggregated diff list
- **THEN** the diff viewer opens in a modal showing that file's before/after content

### Requirement: Duration breakdown
The completion summary SHALL include a visual breakdown of time spent across agents.

#### Scenario: Duration bar chart
- **WHEN** the completion summary is displayed
- **THEN** a horizontal bar chart shows each agent's duration as a bar
- **AND** overlapping bars indicate parallel execution
- **AND** total wall-clock time and total agent-time are displayed as summary numbers

### Requirement: Completion summary follows Peak-End Rule
The completion summary SHALL be the most visually polished view, serving as the satisfying endpoint of the orchestration experience.

#### Scenario: Visual quality
- **WHEN** the completion summary appears
- **THEN** the TerminalBoot typing animation plays for the status banner
- **AND** the results table fades in after the banner finishes typing
- **AND** all numbers (duration, files changed, lines added/removed) are formatted clearly
- **AND** the overall layout uses generous spacing and MU-TH-UR design tokens
