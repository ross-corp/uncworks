## MODIFIED Requirements

### Modified Requirement: Panel surface
Panel MUST be a fixed bottom drawer instead of a center Dialog.

#### Scenario: Panel does not block content
- **WHEN** copilot panel is open
- **THEN** main content area above the panel remains interactive

### Modified Requirement: System prompt with guidance capability
The backend system prompt MUST include instructions for using NAV and HIGHLIGHT tokens.

#### Scenario: Guided response
- **WHEN** user asks "where do I see run traces?"
- **THEN** assistant can respond with both text and `[NAV:/run/ar-123]` or `[HIGHLIGHT:selector]`
