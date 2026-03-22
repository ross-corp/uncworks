## ADDED Requirements

### Requirement: Project List and Detail Views
The web UI SHALL provide a project list view showing all projects with their name, status, run count, and total cost. The UI SHALL provide a project detail view displaying the project's configuration, recent runs, and links to the IDE and spec browser.

#### Scenario: Project list displays all projects
- **WHEN** a user navigates to the projects page
- **THEN** the UI SHALL display all Project resources with their `configRepoReady` status, `runCount`, and `totalCost`

#### Scenario: Project detail shows configuration and runs
- **WHEN** a user clicks on a project in the list
- **THEN** the UI SHALL display the project's repos, devboxPackages, defaults, authorizedKeys, and a list of recent AgentRuns for that project

### Requirement: Spec Browser and Editor
The UI SHALL include a spec browser that reads the directory tree from the project's config repo. The UI SHALL provide a Monaco-based editor for viewing and editing spec files. Saves SHALL commit changes directly to the project's soft-serve config repository.

#### Scenario: Browse specs in config repo
- **WHEN** a user opens the spec browser for a project
- **THEN** the UI SHALL display the file tree under the `openspec/` directory from the project's config repo

#### Scenario: Edit and commit a spec file
- **WHEN** a user edits a spec file in the Monaco editor and clicks save
- **THEN** the UI SHALL commit the updated file to the project's soft-serve config repo with a descriptive commit message

### Requirement: Run from Spec
The UI SHALL provide a "Run this spec" button on each spec file view. Clicking the button SHALL create an AgentRun with the current project as `projectRef` and the spec file path as `specRef`.

#### Scenario: Run triggered from spec view
- **WHEN** a user clicks "Run this spec" on a spec file at path `openspec/auth/spec.md` in project `my-project`
- **THEN** the system SHALL create an AgentRun with `projectRef: my-project` and `specRef: openspec/auth/spec.md`

### Requirement: Project Creation and Settings
The UI SHALL provide a project creation form with fields for name, repos, devbox packages, default model, and SSH authorized keys. The project settings page SHALL include a devbox.json editor for modifying the project's devbox configuration.

#### Scenario: Create a project from the UI
- **WHEN** a user fills in the project creation form and submits it
- **THEN** the system SHALL create a Project resource with the specified configuration

#### Scenario: Edit devbox.json in project settings
- **WHEN** a user modifies the devbox.json content in the project settings editor and saves
- **THEN** the updated devbox.json SHALL be committed to the project's config repo
