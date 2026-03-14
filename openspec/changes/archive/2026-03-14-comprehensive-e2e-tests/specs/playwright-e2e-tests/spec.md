## ADDED Requirements

### Requirement: Dashboard smoke test
The Playwright suite SHALL verify the dashboard loads and displays core UI elements.

#### Scenario: Dashboard renders
- **WHEN** the user navigates to the root URL
- **THEN** the sidebar, table area, and header are visible
- **AND** the phase filter section shows run counts

### Requirement: Create run via form
The Playwright suite SHALL verify the full run creation flow through the web form.

#### Scenario: Create prompt-based run
- **WHEN** the user clicks "+ New Agent Run", fills name/repo/prompt, and clicks "Create Run"
- **THEN** the form closes, a success toast appears, and the new run appears in the table

#### Scenario: Create spec-based run
- **WHEN** the user switches to the "Spec" tab, types spec content in Monaco, and submits
- **THEN** the run is created with `specContent` set and a "spec" badge appears on the table row

#### Scenario: Form validation prevents empty submission
- **WHEN** the user clicks "Create Run" without filling required fields
- **THEN** the form does not submit and required fields are highlighted

### Requirement: Run lifecycle watching
The Playwright suite SHALL verify that run status updates are reflected in the UI in real-time.

#### Scenario: Status transitions visible in table
- **WHEN** a run is created and the user watches the table
- **THEN** the phase badge transitions from Pending through Running to a terminal state within the configured timeout

#### Scenario: Detail panel reflects current status
- **WHEN** the user selects a run in the table
- **THEN** the detail panel shows the run's current phase, repos, prompt, and metadata

### Requirement: HITL interaction via UI
The Playwright suite SHALL verify the human-in-the-loop input flow through the detail panel.

#### Scenario: Send input to waiting agent
- **WHEN** a run reaches "waiting_for_input" phase and the user types in the HITL textarea and clicks "Send Input"
- **THEN** a success toast appears and the run transitions back to "running"

### Requirement: Workspace preset management
The Playwright suite SHALL verify workspace creation, selection, and filtering.

#### Scenario: Create workspace preset
- **WHEN** the user clicks "+ New workspace" in the sidebar, fills name and repos, and saves
- **THEN** the workspace appears in the sidebar with a run count of 0

#### Scenario: Select workspace filters runs
- **WHEN** the user clicks a workspace in the sidebar
- **THEN** only runs tagged with that workspace name are shown in the table

### Requirement: Filter and search
The Playwright suite SHALL verify all filtering mechanisms work correctly.

#### Scenario: Phase filter
- **WHEN** the user clicks a phase filter in the sidebar (e.g., "Running")
- **THEN** only runs with that phase are shown in the table

#### Scenario: Search by name
- **WHEN** the user types a run name in the search bar
- **THEN** only matching runs are shown in the table

### Requirement: Spec editor interactions
The Playwright suite SHALL verify the Monaco editor and GitHub modal interactions.

#### Scenario: Monaco editor loads on spec tab
- **WHEN** the user clicks the "Spec" tab in the run form
- **THEN** the Monaco editor loads (lazy) and accepts text input

#### Scenario: GitHub Load modal
- **WHEN** the user clicks "Load from GitHub" in spec mode
- **THEN** a modal opens with repo and path inputs
- **AND** submitting the form calls the pull API (mocked via page.route)

### Requirement: Repo registry management
The Playwright suite SHALL verify the interactive repo registry.

#### Scenario: Add repo to registry
- **WHEN** the user navigates to the Repos view, enters a URL, and clicks "Add"
- **THEN** the repo appears in the list and becomes available in the run form dropdown
