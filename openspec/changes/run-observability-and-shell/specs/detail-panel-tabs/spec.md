## ADDED Requirements

### Requirement: Tabbed detail panel interface
The AgentRunDetailPanel SHALL use a tabbed layout with Info, Logs, Files, and Shell tabs.

#### Scenario: Default tab is Info
- **WHEN** the user selects a run in the table
- **THEN** the detail panel opens with the Info tab active
- **AND** the Info tab displays all current metadata (phase, repos, prompt, env vars, status, HITL input)

#### Scenario: Switch to Logs tab
- **WHEN** the user clicks the Logs tab
- **THEN** the log viewer renders with xterm.js
- **AND** if the run is active, logs stream in real-time
- **AND** if the run is completed with persisted logs, the stored output is displayed

#### Scenario: Switch to Files tab
- **WHEN** the user clicks the Files tab and the pod exists
- **THEN** the file tree loads the workspace directory structure
- **AND** clicking a file shows its content in a read-only Monaco editor

#### Scenario: Switch to Shell tab
- **WHEN** the user clicks the Shell tab and the pod exists
- **THEN** an interactive terminal connects to the pod via WebSocket
- **AND** the terminal is focused and ready for input

#### Scenario: Tab availability indicators
- **WHEN** a run's pod no longer exists
- **THEN** the Logs tab remains available (falls back to persisted logs)
- **AND** the Files and Shell tabs show a disabled state with "Pod expired" message

### Requirement: Lazy-loaded tab content
Each tab's heavy dependencies (xterm.js, Monaco) SHALL be lazy-loaded to avoid impacting initial page load.

#### Scenario: First Logs tab open
- **WHEN** the user opens the Logs tab for the first time
- **THEN** xterm.js loads asynchronously
- **AND** a loading indicator is shown until the terminal is ready

#### Scenario: First Files tab open
- **WHEN** the user opens the Files tab for the first time
- **THEN** the tree component and Monaco editor load asynchronously
