## ADDED Requirements

### Requirement: Navigation shortcuts
The keys `j` and `k` SHALL move the selected run down and up respectively in the run list. Selection SHALL wrap: pressing `j` on the last row selects the first row, and `k` on the first row selects the last row. The selected row SHALL scroll into view if off-screen.

#### Scenario: Navigate down with j
- **WHEN** run 3 of 10 is selected
- **AND** user presses `j`
- **THEN** run 4 is selected

#### Scenario: Navigate up with k
- **WHEN** run 3 of 10 is selected
- **AND** user presses `k`
- **THEN** run 2 is selected

#### Scenario: Wrap at list boundaries
- **WHEN** the last run is selected
- **AND** user presses `j`
- **THEN** the first run is selected

### Requirement: Detail pane shortcuts
`Enter` SHALL open the detail pane for the selected run. `Escape` SHALL close the detail pane (if no overlay like command palette is open). `q` SHALL close the detail pane and deselect the current run.

#### Scenario: Open detail with Enter
- **WHEN** a run is selected and detail is closed
- **AND** user presses Enter
- **THEN** the detail pane opens for the selected run

#### Scenario: Close detail with Escape
- **WHEN** detail pane is open and no overlay is active
- **AND** user presses Escape
- **THEN** the detail pane closes

#### Scenario: Close all with q
- **WHEN** detail pane is open
- **AND** user presses `q`
- **THEN** the detail pane closes and no run is selected

### Requirement: Command palette shortcut
⌘K (macOS) or Ctrl+K (other platforms) SHALL open the command palette. This shortcut SHALL work even when an input is focused.

#### Scenario: Open command palette
- **WHEN** user presses ⌘K
- **THEN** the command palette opens

### Requirement: Action shortcuts
`n` SHALL open the create run dialog. `1` SHALL set filter to All. `2` SHALL set filter to Active. `3` SHALL set filter to Done. `4` SHALL set filter to Failed. `Tab` SHALL cycle to the next detail tab when the detail pane is open.

#### Scenario: New run shortcut
- **WHEN** user presses `n`
- **THEN** the create run dialog opens

#### Scenario: Filter shortcut
- **WHEN** user presses `2`
- **THEN** the run list filters to Active runs only

#### Scenario: Cycle detail tabs
- **WHEN** detail pane is open on the Info tab
- **AND** user presses Tab
- **THEN** the Logs tab is selected

### Requirement: Shortcuts disabled when input is focused
All single-key shortcuts (j, k, n, q, 1-4) SHALL be suppressed when the active element is an `<input>`, `<textarea>`, or has `contenteditable="true"`. ⌘K and Escape SHALL remain active regardless of focus state.

#### Scenario: Typing in input does not trigger shortcuts
- **WHEN** user is typing in the create run form prompt textarea
- **AND** user types the letter "j"
- **THEN** "j" is typed into the textarea (no navigation occurs)

#### Scenario: ⌘K works while typing
- **WHEN** user is typing in a text input
- **AND** user presses ⌘K
- **THEN** the command palette opens

### Requirement: Visual shortcut hint bar
A fixed bar at the bottom of the viewport SHALL display available keyboard shortcuts. The bar SHALL show context-appropriate hints: list shortcuts when detail is closed, detail shortcuts when detail is open. The bar SHALL be dismissible and its visibility stored in localStorage.

#### Scenario: Hint bar shows list shortcuts
- **WHEN** no detail pane is open
- **THEN** the hint bar shows: j/k Navigate, Enter Open, ⌘K Commands, n New Run

#### Scenario: Hint bar shows detail shortcuts
- **WHEN** detail pane is open
- **THEN** the hint bar shows: Esc Close, Tab Switch Tab, q Close All, ⌘K Commands
