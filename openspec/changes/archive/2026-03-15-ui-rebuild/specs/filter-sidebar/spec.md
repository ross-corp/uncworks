## ADDED Requirements

### Requirement: Status filter with toggle chips
The sidebar SHALL contain a Status filter group with toggle chips for filtering runs by phase. The chips SHALL include: All, Active, Succeeded, and Failed. Selecting a chip filters the RunFeed to show only matching runs. "All" shows all runs.

#### Scenario: Status filter chips are visible
- **WHEN** the sidebar renders
- **THEN** it SHALL display a "Status" label followed by toggle chips: All, Active, Succeeded, Failed

#### Scenario: All is selected by default
- **WHEN** the sidebar first renders
- **THEN** the "All" chip SHALL be in the active/selected visual state
- **AND** the RunFeed SHALL show all runs regardless of phase

#### Scenario: Selecting a status chip filters the feed
- **WHEN** the user clicks the "Failed" chip
- **THEN** the "Failed" chip SHALL enter the active visual state
- **AND** the "All" chip SHALL lose its active state
- **AND** the RunFeed SHALL show only runs with phase "Failed"

#### Scenario: Multiple status chips can be selected
- **WHEN** the user clicks "Active" and then also clicks "Failed"
- **THEN** both chips SHALL be in the active state
- **AND** the RunFeed SHALL show runs with phase "Running" OR "Failed"
- **AND** the "All" chip SHALL lose its active state

#### Scenario: Selecting All clears other selections
- **WHEN** the user clicks "All" while "Active" and "Failed" are selected
- **THEN** the "All" chip SHALL enter active state
- **AND** the "Active" and "Failed" chips SHALL lose their active state
- **AND** the RunFeed SHALL show all runs

### Requirement: Repo filter with auto-populated removable chips
The sidebar SHALL contain a Repo filter group with chips auto-populated from the repositories present in the current runs. Each chip SHALL be removable (has an X button) to exclude that repo from results.

#### Scenario: Repo chips are auto-populated
- **WHEN** runs are loaded and contain repos "org/frontend" and "org/backend"
- **THEN** the Repo filter group SHALL display chips for "org/frontend" and "org/backend"

#### Scenario: All repos are shown by default
- **WHEN** the sidebar first renders
- **THEN** all repo chips SHALL be in the active/included state
- **AND** the RunFeed SHALL not filter by repo

#### Scenario: Removing a repo chip filters runs
- **WHEN** the user clicks the X on the "org/frontend" chip
- **THEN** that chip SHALL enter an inactive/excluded state or be visually de-emphasized
- **AND** the RunFeed SHALL exclude runs associated with "org/frontend"

### Requirement: Model filter with toggle chips
The sidebar SHALL contain a Model filter group with toggle chips for filtering by model/backend. The available chips SHALL be derived from the models present in the current runs.

#### Scenario: Model chips rendered
- **WHEN** runs use models "claude-sonnet" and "claude-opus"
- **THEN** the Model filter group SHALL show toggle chips for each model

#### Scenario: Model chip toggles filter
- **WHEN** the user deactivates the "claude-sonnet" chip
- **THEN** the RunFeed SHALL exclude runs that used "claude-sonnet"

### Requirement: Workspace filter with chips
The sidebar SHALL contain a Workspace filter group with chips for filtering by workspace. The available chips SHALL be derived from the workspaces present in the current runs.

#### Scenario: Workspace chips rendered
- **WHEN** runs span workspaces "dev" and "staging"
- **THEN** the Workspace filter group SHALL show toggle chips for each workspace

#### Scenario: Workspace chip toggles filter
- **WHEN** the user deactivates the "dev" chip
- **THEN** the RunFeed SHALL exclude runs from the "dev" workspace

### Requirement: No navigation destinations in sidebar
The sidebar SHALL NOT contain navigation links to separate views (no "Repositories" view, no "Events" view). All content is accessed through the single feed view with filters. Repos and events are represented as filter chips, not navigation targets.

#### Scenario: No Repositories navigation link
- **WHEN** the sidebar is rendered
- **THEN** there SHALL be no clickable link or button labeled "Repositories" that navigates to a separate view

#### Scenario: No Events navigation link
- **WHEN** the sidebar is rendered
- **THEN** there SHALL be no clickable link or button labeled "Events" that navigates to a separate view

### Requirement: New Run button in sidebar
The sidebar SHALL contain a "+ New Run" button that opens the run creation form/dialog. This button SHALL be prominently placed (e.g., at the top of the sidebar).

#### Scenario: New Run button is visible
- **WHEN** the sidebar renders
- **THEN** a button labeled "+ New Run" (or equivalent) SHALL be visible near the top of the sidebar

#### Scenario: New Run button opens creation form
- **WHEN** the user clicks the "+ New Run" button
- **THEN** the run creation form/dialog SHALL open
