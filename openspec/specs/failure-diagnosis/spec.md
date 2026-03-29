# failure-diagnosis Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Failure diagnosis panel on failed runs
When a run's phase is failed, RunDetailView SHALL display a collapsible diagnosis panel below the header.

#### Scenario: Panel appears on failed runs
- **WHEN** run.status.phase === "failed"
- **THEN** a red-bordered collapsible panel appears below the status header
- **AND** the panel shows: which stage failed, the error message, elapsed time at failure

#### Scenario: Panel links to failing trace span
- **WHEN** the failure diagnosis panel is visible
- **THEN** a "View in Traces" button jumps to the Traces tab and auto-scrolls to the first failed span

#### Scenario: Panel offers retry and edit actions
- **WHEN** the failure diagnosis panel is visible
- **THEN** action buttons include: "Retry", "Edit & Retry", and "Archive"
- **AND** "Retry" creates a new run with the same spec/prompt
- **AND** "Edit & Retry" navigates to NewRunView pre-filled with the failed run's parameters

