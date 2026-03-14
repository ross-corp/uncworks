## ADDED Requirements

### Requirement: Monaco editor component
The web UI SHALL include a `SpecEditor` component that wraps Monaco editor configured for markdown editing with a VS Code dark theme matching the application's design system.

#### Scenario: Editor renders with markdown highlighting
- **WHEN** the SpecEditor component mounts with initial content `"# MyConverter\n\nConverts CSV files to JSON."`
- **THEN** Monaco editor renders with markdown syntax highlighting, line numbers, and word wrap enabled

#### Scenario: Editor reports content changes
- **WHEN** the user edits text in the Monaco editor
- **THEN** the `onChange` callback fires with the updated content string

#### Scenario: Editor supports read-only mode
- **WHEN** SpecEditor is rendered with `readOnly: true`
- **THEN** the editor displays content but does not allow modifications

### Requirement: Monaco lazy loading
Monaco editor SHALL be lazy-loaded via dynamic import to avoid adding its bundle size to the initial page load.

#### Scenario: Initial page load
- **WHEN** the app loads and the user has not opened the spec editor
- **THEN** Monaco's JavaScript is NOT included in the initial bundle

#### Scenario: First spec editor open
- **WHEN** the user clicks the "Spec" tab for the first time
- **THEN** Monaco loads asynchronously and the editor renders after loading completes
- **AND** a loading indicator is shown during the load

### Requirement: Agent run form spec tab
The agent run form SHALL have a tab selector with "Prompt" and "Spec" modes. The Prompt tab shows the existing textarea. The Spec tab shows the Monaco-based SpecEditor.

#### Scenario: Switch to spec mode
- **WHEN** the user clicks the "Spec" tab in the agent run form
- **THEN** the prompt textarea is hidden and the SpecEditor is shown
- **AND** the form tracks that this is a spec-driven run

#### Scenario: Switch back to prompt mode
- **WHEN** the user clicks the "Prompt" tab after being in spec mode
- **THEN** the SpecEditor is hidden and the prompt textarea is shown
- **AND** any spec content entered is preserved (not lost on tab switch)

#### Scenario: Default mode is prompt
- **WHEN** the agent run form opens
- **THEN** the "Prompt" tab is active by default

### Requirement: Spec viewing in detail panel
The detail panel SHALL show spec content for spec-driven runs using the SpecEditor in read-only mode.

#### Scenario: View spec of a completed run
- **WHEN** a user selects a spec-driven run in the table
- **THEN** the detail panel shows a "Spec" section with the full spec content rendered in a read-only Monaco editor

#### Scenario: Prompt-only run
- **WHEN** a user selects a prompt-driven run (no spec content)
- **THEN** the detail panel shows the prompt as before with no spec section

### Requirement: Spec run indicator in table
The agent run table SHALL visually distinguish spec-driven runs from prompt-driven runs.

#### Scenario: Spec run in table
- **WHEN** a run has `specContent` set
- **THEN** the table row shows a small indicator (badge or icon) in the name or message column identifying it as a spec run

#### Scenario: Prompt run in table
- **WHEN** a run has no `specContent`
- **THEN** no spec indicator is shown
