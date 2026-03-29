## ADDED Requirements

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
