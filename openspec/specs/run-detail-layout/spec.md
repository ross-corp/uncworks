# run-detail-layout Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Sidebar+main+panel layout for RunDetailView
RunDetailView SHALL use a three-zone layout: a left nav sidebar (Logs / Traces / Files / Shell), a main content area, and a right detail panel that slides in contextually.

#### Scenario: Default view shows activity feed
- **WHEN** the user navigates to /run/:id
- **THEN** the main content area shows the ActivityFeed by default
- **AND** the left nav sidebar shows Logs, Traces, Files, Shell as nav items

#### Scenario: Right panel slides in on trace span selection
- **WHEN** the user clicks a trace span in the Traces view
- **THEN** a detail panel slides in from the right without replacing the main content
- **AND** the user can see both the trace waterfall and the span detail simultaneously

#### Scenario: Keyboard shortcuts preserved
- **WHEN** the user presses 1/2/3/4
- **THEN** the main content switches to Logs/Traces/Files/Shell respectively
- **AND** Escape closes the right detail panel if open

### Requirement: Unified status header
RunDetailView SHALL show a two-row status header with run name/phase/status in row 1 and current activity summary in row 2.

#### Scenario: Header shows live elapsed time
- **WHEN** a run is in a running phase
- **THEN** the header displays elapsed time as a live counter (e.g., "5m 23s")

#### Scenario: Header shows last activity summary
- **WHEN** the activity feed has entries
- **THEN** row 2 of the header shows the most recent event type and tool name (e.g., "Executing: bash tool")

### Requirement: Phase/stage step indicator
RunDetailView SHALL show a horizontal step indicator (Planning → Executing → Verifying) replacing the current badge row.

#### Scenario: Current stage highlighted
- **WHEN** a run is in the Executing phase
- **THEN** "Planning" shows as complete (green), "Executing" shows as active (blue/pulsing), "Verifying" shows as pending (muted)

#### Scenario: Failed stage shown with error
- **WHEN** a run fails during Executing
- **THEN** "Executing" shows as failed (red X)
- **AND** clicking the failed stage jumps to its trace spans

