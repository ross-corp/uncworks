# trace-search Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Span search and filter in TraceTimeline
The TraceTimeline SHALL provide a filter bar to search and filter spans by type, status, and text.

#### Scenario: Text search filters spans
- **WHEN** the user types in the trace search box
- **THEN** only spans whose name contains the search text are shown
- **AND** non-matching parent spans are shown collapsed if they have matching children

#### Scenario: Filter by status shows only failed spans
- **WHEN** the user selects the "Failed" filter chip
- **THEN** only spans with error status are shown in the waterfall
- **AND** a count badge shows "N failed spans"

#### Scenario: Filter by tool type
- **WHEN** the user selects a tool type chip (e.g., "bash", "write", "llm")
- **THEN** only spans of that tool type are shown

#### Scenario: Clear filters restores full view
- **WHEN** the user clicks "Clear" or presses Escape in the search box
- **THEN** all spans are shown again

### Requirement: Expand all / Collapse all for trace spans
The TraceTimeline SHALL provide buttons to expand or collapse all span groups at once.

#### Scenario: Collapse all collapses every expandable group
- **WHEN** the user clicks "Collapse All"
- **THEN** all stage-level span groups are collapsed to show only their headers
- **AND** the button changes to "Expand All"

#### Scenario: Expand all expands every group
- **WHEN** the user clicks "Expand All"
- **THEN** all span groups are expanded showing all child spans
- **AND** the collapse state persists until the user navigates away

#### Scenario: Collapsed groups show child count badge
- **WHEN** a span group is collapsed
- **THEN** a badge shows "[N hidden]" next to the group header

### Requirement: Span hover and click affordance
Spans SHALL have clear hover and click affordances indicating they are interactive.

#### Scenario: Hover highlights span row
- **WHEN** the user hovers over a span row
- **THEN** the entire row gets a visible background highlight and the span name gets an underline
- **AND** a cursor:pointer is shown

#### Scenario: Selected span persists highlight
- **WHEN** the user clicks a span
- **THEN** the row stays highlighted (accent background) until another span is clicked or Escape is pressed

