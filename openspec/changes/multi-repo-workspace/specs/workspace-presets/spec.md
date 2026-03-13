## ADDED Requirements

### Requirement: Workspace preset persistence
Workspace presets SHALL be stored in localStorage under the key `uncworks:workspaces` as a JSON array. Each workspace MUST have an `id` (uuid), `name`, `description`, `repos` array (each with `url` and `branch`), `createdAt`, and `updatedAt` timestamps.

#### Scenario: Persist a new workspace
- **WHEN** a user creates a workspace named "payments-platform" with repos `[{url: "https://github.com/org/api", branch: "main"}, {url: "https://github.com/org/web", branch: "main"}]`
- **THEN** a new entry is added to `uncworks:workspaces` in localStorage with a generated uuid, the given name, repos, and current timestamps

#### Scenario: Load workspaces on app start
- **WHEN** the app initializes
- **THEN** workspace presets are read from `uncworks:workspaces` in localStorage and made available to the sidebar and form components

#### Scenario: Empty localStorage
- **WHEN** `uncworks:workspaces` does not exist in localStorage
- **THEN** the app treats the workspace list as empty with no errors

### Requirement: Workspace CRUD operations
The UI SHALL support creating, editing, and deleting workspace presets through a workspace editor modal.

#### Scenario: Create workspace
- **WHEN** a user clicks "+ New workspace" in the sidebar
- **THEN** a workspace editor modal opens with empty fields for name, description, and an empty repo list
- **AND** the user can add repos (url + branch), set a name and description, and save

#### Scenario: Edit workspace
- **WHEN** a user opens the editor for an existing workspace
- **THEN** the modal pre-fills with the workspace's current name, description, and repos
- **AND** the user can add/remove repos, change branches, update name/description, and save

#### Scenario: Delete workspace
- **WHEN** a user deletes a workspace from the editor
- **THEN** the workspace is removed from localStorage
- **AND** existing runs tagged with that workspace name are NOT affected

### Requirement: Workspace selector in agent run form
The agent run form SHALL present workspace presets as selectable options, pre-filling the repo list from the selected workspace.

#### Scenario: Select a workspace
- **WHEN** a user selects workspace "payments-platform" in the form
- **THEN** the repos section pre-fills with the workspace's repos and branches
- **AND** the `workspaceName` field is set to "payments-platform"

#### Scenario: Modify repos after workspace selection
- **WHEN** a user selects a workspace and then adds or removes a repo
- **THEN** only this run is affected — the workspace preset is not modified
- **AND** the `workspaceName` still reflects the original workspace

#### Scenario: Custom repos mode
- **WHEN** a user selects "Custom repos" instead of a workspace
- **THEN** the repos section is empty and the user builds the list manually
- **AND** `workspaceName` is empty

### Requirement: Sidebar workspace section
The sidebar SHALL include a "Workspaces" collapsible section showing all saved workspace presets with run counts.

#### Scenario: Display workspaces with counts
- **WHEN** workspace "payments-platform" exists and 5 runs have `workspaceName: "payments-platform"`
- **THEN** the sidebar shows "payments-platform (5)"

#### Scenario: Filter by workspace
- **WHEN** a user clicks workspace "payments-platform" in the sidebar
- **THEN** the runs view filters to only runs with `workspaceName === "payments-platform"`

#### Scenario: No workspaces exist
- **WHEN** no workspace presets are saved
- **THEN** the Workspaces section shows only "+ New workspace..."

### Requirement: workspace_name on AgentRunSpec
The proto `AgentRunSpec` message SHALL include a `string workspace_name` field. The CRD `AgentRunSpec` type SHALL include a corresponding `WorkspaceName string` field. The field is optional and defaults to empty string.

#### Scenario: Run created from workspace
- **WHEN** an agent run is created from workspace "payments-platform"
- **THEN** the run's spec contains `workspace_name: "payments-platform"`

#### Scenario: Run created without workspace
- **WHEN** an agent run is created with custom repos (no workspace)
- **THEN** the run's spec contains `workspace_name: ""`

### Requirement: Repo registry persistence
A repo registry SHALL be stored in localStorage under `uncworks:repos` as a JSON array of URL strings. The registry is merged with repos derived from existing runs to form the complete list available in the form and sidebar.

#### Scenario: Add repo to registry
- **WHEN** a user adds "https://github.com/org/new-service" via the ReposView
- **THEN** the URL is appended to `uncworks:repos` in localStorage
- **AND** the repo appears in the form's repo options and sidebar's repo filter

#### Scenario: Remove repo from registry
- **WHEN** a user removes a repo from the registry
- **THEN** the URL is removed from `uncworks:repos`
- **AND** if no existing runs reference it, the repo disappears from the form and sidebar

#### Scenario: Merge registry with run-derived repos
- **WHEN** the registry contains `[A, B]` and existing runs reference repos `[B, C]`
- **THEN** the combined repo list is `[A, B, C]` (deduplicated)
