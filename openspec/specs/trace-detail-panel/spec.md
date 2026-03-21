# trace-detail-panel Specification

## Purpose
TBD - created by archiving change observability-ux-overhaul. Update Purpose after archive.
## Requirements
### Requirement: Right split detail panel replaces inline expansion
The system SHALL display span details in a right split panel when a span is clicked, replacing the inline expansion that causes layout overlap.

#### Scenario: Click span opens detail panel
- **WHEN** the user clicks a span row in the waterfall
- **THEN** a detail panel SHALL appear on the right side of the waterfall (40% width)
- **AND** the waterfall SHALL resize to 60% width
- **AND** the detail panel SHALL show the span's metadata, content, and diff

#### Scenario: Empty state when no span selected
- **WHEN** no span is selected in the waterfall
- **THEN** the detail panel area SHALL show "Click a span to view details"

### Requirement: Detail panel shows thinking text for thought spans
The system SHALL display the LLM thinking/response text in the detail panel for `*.thought` spans.

#### Scenario: Thought span shows content
- **WHEN** the user clicks a `manage.thought` span
- **THEN** the detail panel SHALL show a "Content" section with the LLM response text
- **AND** the text SHALL be rendered with markdown formatting

### Requirement: Detail panel shows diff for spans with hasDiff
The system SHALL fetch and display the git diff in the detail panel when a span has `hasDiff=true`.

#### Scenario: Diff loaded on span click
- **WHEN** the user clicks a span with `hasDiff=true`
- **THEN** the detail panel SHALL fetch the diff from `/api/v1/runs/{id}/traces/{spanId}/diff`
- **AND** display it with green/red line coloring for additions/deletions

#### Scenario: Diff shows file paths
- **WHEN** a diff with multiple files is displayed
- **THEN** each file SHALL be shown with its path as a collapsible header

### Requirement: Stage separator lines in waterfall
The system SHALL render horizontal separator lines between spans of different pipeline stages.

#### Scenario: Stage transition separator
- **WHEN** consecutive spans have different `metadata.stage` values (e.g., "plan" then "execute")
- **THEN** a horizontal separator line SHALL be rendered between them with the new stage name

