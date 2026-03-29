# project-management Specification

## Purpose
TBD - created by archiving change run-organization. Update Purpose after archive.
## Requirements
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

### Requirement: Runs tab in ProjectDetailView
ProjectDetailView SHALL include a Runs tab showing all runs associated with the project.

#### Scenario: Runs tab shows project-filtered runs
- **WHEN** the user clicks the Runs tab in ProjectDetailView
- **THEN** all runs where spec.project === projectName are shown in a list
- **AND** each run shows status, name, model tier, age, and a link to the run detail

#### Scenario: Empty runs tab shows call to action
- **WHEN** the project has no runs
- **THEN** the Runs tab shows "No runs yet — [+ New Run]" with a link to /new?project=:name

### Requirement: Real tabs in ProjectDetailView
ProjectDetailView SHALL use proper tab components (not Badge-based onClick toggles) for Specs / Runs / Settings.

#### Scenario: Tabs use shadcn Tabs component
- **WHEN** the project detail view renders
- **THEN** the Specs / Runs / Settings tabs use the shadcn ui Tabs component
- **AND** the active tab is highlighted with the standard active tab style

