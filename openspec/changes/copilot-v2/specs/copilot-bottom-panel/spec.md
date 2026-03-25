## ADDED Requirements

### Requirement: Bottom panel layout
The copilot panel MUST be rendered as a fixed bottom panel spanning the full viewport width, not a center modal.

#### Scenario: Panel toggle
- **WHEN** user presses ⌘K (or Ctrl+K) from any view
- **THEN** the bottom panel slides open to its last-used height (default 320px)

#### Scenario: Panel persists during navigation
- **WHEN** user navigates to a different route while panel is open
- **THEN** panel remains open and messages are preserved

#### Scenario: Panel resize
- **WHEN** user drags the resize handle at the top of the panel
- **THEN** panel height adjusts between 200px and 70vh

#### Scenario: Close panel
- **WHEN** user presses Escape or ⌘K again
- **THEN** panel closes but state is preserved (not reset)
