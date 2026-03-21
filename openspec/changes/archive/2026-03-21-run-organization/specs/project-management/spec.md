## ADDED Requirements

### Requirement: Project CRUD via labels
The system SHALL support creating, listing, and deleting projects as label values on AgentRun CRDs without requiring a separate Project CRD.

#### Scenario: Create project by assigning to run
- **WHEN** a user sets `project: "neph-nvim"` on a new or existing run
- **THEN** the project "neph-nvim" becomes available in the project picker and all runs with that label are grouped

#### Scenario: List projects
- **WHEN** the API receives a ListAgentRuns request
- **THEN** the response SHALL include a `projects` field containing the distinct set of project labels across all runs

#### Scenario: Filter runs by project
- **WHEN** the API receives a ListAgentRuns request with `project_filter: "neph-nvim"`
- **THEN** only runs with label `aot.uncworks.io/project=neph-nvim` are returned

### Requirement: Project selector in UI
The system SHALL provide a project selector in the run list header that filters all visible runs.

#### Scenario: Switch project context
- **WHEN** the user presses `p` in the run list and selects "neph-nvim"
- **THEN** the run list shows only runs belonging to project "neph-nvim"
- **AND** the header displays "[p] neph-nvim" as the active filter

#### Scenario: Show all projects
- **WHEN** the user selects "(all projects)" from the project picker
- **THEN** all runs are shown regardless of project assignment
