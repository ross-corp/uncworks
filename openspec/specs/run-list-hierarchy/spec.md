# run-list-hierarchy Specification

## Purpose
TBD - created by archiving change run-organization. Update Purpose after archive.
## Requirements
### Requirement: Feature-grouped run list view
The system SHALL support a run list mode that groups runs by feature, showing each feature as a row with aggregate status.

#### Scenario: Features view
- **WHEN** the user selects the "features" view mode (press 1)
- **THEN** runs are grouped by feature label
- **AND** each feature row shows: name, status (DONE/FAILED/RUNNING), attempt count, PR link

#### Scenario: Expand feature to see runs
- **WHEN** the user selects a feature row and presses enter
- **THEN** the feature detail view shows all runs for that feature with individual status

#### Scenario: Unassigned runs shown separately
- **WHEN** runs exist without a feature label
- **THEN** they are shown in a separate "Unassigned" section below the features

### Requirement: Tab order update
The system SHALL order the run detail tabs as: 1) Logs, 2) Traces, 3) Files, 4) Shell.

#### Scenario: Tab ordering
- **WHEN** the user views a run detail
- **THEN** pressing 1 shows Logs, 2 shows Traces, 3 shows Files, 4 shows Shell
- **AND** the Verify tab SHALL be removed (verification data shown inline in Logs)

### Requirement: Feature group header expand/collapse affordance
Feature group headers in RunListView SHALL have a clearly visible chevron and full-row hover state.

#### Scenario: Chevron is visually prominent
- **WHEN** a feature group header is rendered
- **THEN** the chevron is text-sm or larger, foreground color (not muted)
- **AND** rotates 90° when expanded

#### Scenario: Full row is hoverable
- **WHEN** the user hovers over a feature group header
- **THEN** the entire row shows a background highlight indicating interactivity

