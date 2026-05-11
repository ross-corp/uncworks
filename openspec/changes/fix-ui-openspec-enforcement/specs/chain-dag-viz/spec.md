## REMOVED Requirements

### Requirement: Timeline view tab in ChainRunDetailView
**Reason**: The Timeline tab displays the same step data (name, phase, duration, start time) as the Runs tab with no meaningful differentiation. Maintaining two tabs with identical data confuses users and creates UI surface area with no additional value.
**Migration**: Users should use the "Runs" tab for the same step list information. No data is lost.

## MODIFIED Requirements

### Requirement: Visual DAG in ChainRunDetailView
ChainRunDetailView SHALL render a visual DAG using react-flow with nodes for each step and edges for dependencies. The tab containing this DAG SHALL be labeled "Overview" (not "DAG").

#### Scenario: Nodes colored by phase
- **WHEN** a chain run detail is viewed
- **THEN** each step node is colored by its phase: blue=running, green=succeeded, red=failed, gray=pending, yellow=skipped

#### Scenario: Edges show dependency direction
- **WHEN** step B depends on step A
- **THEN** a directed edge is drawn from A to B with an arrow

#### Scenario: Node shows duration
- **WHEN** a step has completed (succeeded or failed)
- **THEN** the node shows the elapsed duration inside (e.g., "1m 23s")

#### Scenario: Clicking a node links to the run
- **WHEN** the user clicks a step node that has a run ID
- **THEN** the user is navigated to /run/:id for that step's run

#### Scenario: Fallback to text representation
- **WHEN** react-flow fails to load
- **THEN** the existing text-based step list is shown as fallback

#### Scenario: Tab label is "Overview"
- **WHEN** the user views a chain run detail page
- **THEN** the first tab is labeled "Overview" not "DAG"

## ADDED Requirements

### Requirement: Runs tab fills panel width
The Runs sub-tab in ChainRunDetailView SHALL render the step table at full panel width without centering constraints.

#### Scenario: No max-width centering on runs tab
- **WHEN** the user views the Runs tab in ChainRunDetailView
- **THEN** the step table occupies the full available width of the content panel
- **AND** no `max-w-2xl` or `mx-auto` centering is applied to the table container
