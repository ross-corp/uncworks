# chain-dag-viz Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Visual DAG in ChainRunDetailView
ChainRunDetailView SHALL render a visual DAG using react-flow with nodes for each step and edges for dependencies.

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

### Requirement: Timeline view tab in ChainRunDetailView
ChainRunDetailView SHALL offer a timeline tab showing step execution order with start times and durations.

#### Scenario: Timeline shows steps in execution order
- **WHEN** the user switches to the Timeline tab
- **THEN** steps are shown as horizontal bars on a time axis
- **AND** dependent steps appear after their dependencies on the timeline

