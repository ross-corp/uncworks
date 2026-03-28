## ADDED Requirements

### Requirement: Keybindings have three preset modes
The system SHALL support three built-in preset modes: "default" (g-leader), "vim" (space-leader), and "emacs" (ctrl+x-leader). Switching preset SHALL take effect immediately without restart.

#### Scenario: Default preset navigation
- **WHEN** preset is "default" and user presses "g" then "w"
- **THEN** the app navigates to the Workflows view

#### Scenario: Vim preset navigation
- **WHEN** preset is "vim" and user presses Space then "w"
- **THEN** the app navigates to the Workflows view

#### Scenario: Emacs preset navigation
- **WHEN** preset is "emacs" and user presses ctrl+x then "w"
- **THEN** the app navigates to the Workflows view

#### Scenario: Preset switch is immediate
- **WHEN** the user changes the preset in Settings
- **THEN** the new keybindings take effect immediately without page reload

### Requirement: Custom keybinding overrides are stored as a sparse delta
The system SHALL store only user-modified bindings (overrides) rather than full keymaps. Effective bindings SHALL be resolved as merge(preset, overrides) at runtime.

#### Scenario: Override a single binding
- **WHEN** the user remaps "nav.workflows" to "ctrl+1"
- **THEN** only that override is stored; all other preset bindings remain unchanged

#### Scenario: Reset override
- **WHEN** the user resets a custom override for an action
- **THEN** that action returns to the current preset's default

### Requirement: Keybindings do not intercept text input fields
The system SHALL NOT intercept keyboard events when the event target is an INPUT, TEXTAREA, SELECT, or contenteditable element.

#### Scenario: Typing in a form field
- **WHEN** the user is typing in a text input and presses a bound key (e.g. "g")
- **THEN** the character is typed normally and no navigation action fires

### Requirement: Chord sequences use a two-step state machine
The system SHALL support two-key chord sequences (e.g. "g w"). The first key SHALL be buffered; the second key within a timeout SHALL complete the chord.

#### Scenario: Chord completes
- **WHEN** user presses "g" followed by "w" within 2 seconds
- **THEN** the "nav.workflows" action fires

#### Scenario: Chord prefix then unrecognized key
- **WHEN** user presses "g" followed by an unbound key
- **THEN** no action fires and the chord state resets to idle

#### Scenario: Chord cancelled by Escape
- **WHEN** user presses a chord prefix then presses Escape
- **THEN** chord state resets to idle and which-key popup (if visible) hides
