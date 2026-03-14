## ADDED Requirements

### Requirement: Global keyboard shortcut system
The application SHALL register global keyboard shortcuts for view switching and panel management. Shortcuts SHALL be disabled when focus is inside an input, textarea, xterm.js terminal, or Monaco editor to avoid conflicts. A `useKeyboardShortcuts` hook SHALL manage registration and cleanup.

#### Scenario: Switch to Graph view
- **WHEN** user presses Ctrl+G and focus is not in an input element
- **THEN** the workspace switches to Graph view

#### Scenario: Toggle log stream
- **WHEN** user presses Ctrl+L
- **THEN** the log stream toggles between minimized and its previous height

#### Scenario: Switch to Files view
- **WHEN** user presses Ctrl+F and focus is not in an input element
- **THEN** the workspace switches to Files view
- **AND** the default browser "Find" action is prevented

#### Scenario: Switch to Shell view
- **WHEN** user presses Ctrl+T
- **THEN** the workspace switches to Shell view
- **AND** the default browser "New Tab" action is prevented

#### Scenario: Shortcuts disabled in input context
- **WHEN** user presses Ctrl+G while typing in the navigator search bar
- **THEN** the shortcut is not triggered and normal text input behavior occurs

### Requirement: Command palette
The application SHALL provide a command palette accessible via Ctrl+K. The palette SHALL offer fuzzy search over: view switching actions, specs (by name), runs (by ID or spec), and navigation commands. Selecting an item SHALL execute the corresponding action. Pressing Escape SHALL close the palette.

#### Scenario: Open command palette
- **WHEN** user presses Ctrl+K
- **THEN** a centered overlay appears with a search input and a list of available commands

#### Scenario: Search for a spec
- **WHEN** user types a spec name in the command palette
- **THEN** matching specs appear in the results list
- **AND** selecting a spec navigates to it (sets selection and shows its graph)

#### Scenario: Execute a view command
- **WHEN** user types "files" in the command palette
- **THEN** "Switch to Files view" appears as a result
- **AND** selecting it switches the workspace to Files view and closes the palette

#### Scenario: Close command palette
- **WHEN** user presses Escape while the command palette is open
- **THEN** the palette closes and focus returns to the previously focused element
