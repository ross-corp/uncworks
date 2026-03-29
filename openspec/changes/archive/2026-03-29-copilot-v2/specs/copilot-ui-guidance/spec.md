## ADDED Requirements

### Requirement: Navigation action tokens
The copilot MUST support `[NAV:/path]` tokens in responses that trigger client-side navigation.

#### Scenario: Navigate action
- **WHEN** assistant response contains `[NAV:/run/ar-123]`
- **THEN** the app navigates to `/run/ar-123` and the token is removed from displayed text

### Requirement: Element highlight tokens
The copilot MUST support `[HIGHLIGHT:css-selector]` tokens that visually highlight UI elements.

#### Scenario: Highlight element
- **WHEN** assistant response contains `[HIGHLIGHT:.run-status-badge]`
- **THEN** matching elements get a visible ring highlight for 3 seconds
- **AND** the token is removed from displayed text

#### Scenario: No match — silent fail
- **WHEN** no element matches the selector
- **THEN** nothing visible happens (no error)
