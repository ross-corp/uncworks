## ADDED Requirements

### Requirement: Command palette opens via keyboard shortcut
The command palette SHALL open when the user presses ⌘K (macOS) or Ctrl+K (other platforms). The shortcut SHALL work regardless of current focus state. When opened, the input field SHALL be auto-focused.

#### Scenario: Open palette with keyboard
- **WHEN** user presses ⌘K
- **THEN** the command palette dialog appears as a centered overlay
- **AND** the search input is focused

#### Scenario: Palette opens even when input is focused
- **WHEN** user is typing in another input field and presses ⌘K
- **THEN** the command palette opens (⌘K is never suppressed)

### Requirement: Fuzzy search over runs, commands, and filters
The palette SHALL search across three result types: runs (matched against name, ID, prompt, and repo), built-in commands, and filter presets. Search SHALL use case-insensitive `String.includes()` on each searchable field. Results SHALL be grouped by type with a type label header.

#### Scenario: Search matches run by prompt substring
- **WHEN** user types "refactor" in the palette
- **THEN** runs whose prompt contains "refactor" appear in results under a "Runs" group

#### Scenario: Search matches built-in command
- **WHEN** user types "theme" in the palette
- **THEN** "Toggle Theme" appears in results under a "Commands" group

#### Scenario: No results
- **WHEN** user types a string that matches nothing
- **THEN** the palette shows "No results" text

### Requirement: Built-in commands
The palette SHALL include these built-in commands: New Run, Toggle Theme, Filter Active, Filter Done, Filter Failed, Show All. Each command SHALL have an icon and a label. Executing a command SHALL close the palette and perform the action.

#### Scenario: Execute New Run command
- **WHEN** user selects "New Run" from the palette
- **THEN** the palette closes
- **AND** the create run dialog opens

#### Scenario: Execute Toggle Theme command
- **WHEN** user selects "Toggle Theme" from the palette
- **THEN** the palette closes
- **AND** the theme switches between light and dark

### Requirement: Keyboard navigation within palette
Arrow keys SHALL navigate the result list. Enter SHALL execute the highlighted result. Escape SHALL close the palette. The first result SHALL be highlighted by default.

#### Scenario: Navigate and select with keyboard
- **WHEN** palette is open with results
- **THEN** the first result is highlighted
- **WHEN** user presses Down arrow twice then Enter
- **THEN** the third result is executed

#### Scenario: Escape closes palette
- **WHEN** palette is open
- **AND** user presses Escape
- **THEN** the palette closes and focus returns to the previous element

### Requirement: Most recently used shown when input is empty
When the palette opens and the search input is empty, the palette SHALL display the most recently executed commands and recently viewed runs (up to 5 items). Items are ordered by recency.

#### Scenario: Empty input shows recent items
- **WHEN** palette opens with empty input
- **AND** user previously opened run "abc123" and executed "Toggle Theme"
- **THEN** both items appear in the results list ordered by recency
