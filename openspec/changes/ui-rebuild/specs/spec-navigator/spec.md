## ADDED Requirements

### Requirement: Tree-based spec browser
The navigator SHALL display specs as top-level tree nodes. Each spec node SHALL show the spec name and an inline orchestration summary (e.g., "3 runs: 2 passed, 1 active"). Expanding a spec SHALL reveal its run tree showing parent-child run relationships. Run nodes SHALL display their phase with MU-TH-UR status indicators: phosphor green pulse for Running, steady amber for Succeeded, red glow for Failed, muted for Pending.

#### Scenario: Specs load on application start
- **WHEN** the application loads
- **THEN** the navigator fetches the spec list and displays them as tree nodes
- **AND** each spec shows its name and run count summary

#### Scenario: Expand spec to see runs
- **WHEN** user clicks the expand arrow on a spec node
- **THEN** the spec's runs are fetched (if not cached) and displayed as child nodes
- **AND** runs are nested to show parent-child orchestration relationships

#### Scenario: Run status indicators
- **WHEN** a run node is visible in the tree
- **THEN** it displays a status indicator matching its phase: pulsing green circle for Running, amber check for Succeeded, red X with glow for Failed, gray circle for Pending

#### Scenario: Live status updates
- **WHEN** a run's phase changes while the navigator is open
- **THEN** the run's status indicator updates without user interaction
- **AND** the parent spec's summary count updates

### Requirement: Spec selection drives workspace
Clicking a spec in the navigator SHALL set it as the selected spec, clear any run/span selection, and switch the workspace to Graph view showing that spec's orchestration tree. Clicking a run in the navigator SHALL set it as the selected run and show its details in the right panel and its logs in the bottom log stream.

#### Scenario: Select spec shows its orchestration graph
- **WHEN** user clicks a spec node in the navigator
- **THEN** the workspace switches to Graph view showing that spec's run graph
- **AND** the detail panel shows spec-level metadata

#### Scenario: Select run from navigator
- **WHEN** user clicks a run node in the navigator tree
- **THEN** the right detail panel shows that run's metadata (phase, duration, repos, prompt)
- **AND** the bottom log stream attaches to that run's log output

### Requirement: Navigator search and filter
The navigator SHALL include a search bar at the top that filters specs by name. When a filter is active, only matching specs (and their run subtrees) SHALL be visible. Clearing the search SHALL restore the full tree.

#### Scenario: Filter specs by name
- **WHEN** user types "auth" in the navigator search bar
- **THEN** only specs whose names contain "auth" are shown
- **AND** their run subtrees are fully visible

#### Scenario: Clear filter
- **WHEN** user clears the search bar
- **THEN** all specs are visible again
