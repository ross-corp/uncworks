## ADDED Requirements

### Requirement: Which-key popup appears after chord prefix is held
The system SHALL show a which-key popup after a configurable delay when the user has pressed a chord prefix key.

#### Scenario: Popup appears after delay
- **WHEN** user presses a chord prefix key and holds without pressing the second key for longer than whichKeyDelayMs
- **THEN** a popup appears showing all bindings that start with that prefix

#### Scenario: Popup content is filtered to prefix
- **WHEN** the popup is visible for prefix "g"
- **THEN** only bindings starting with "g " are shown, with the "g " stripped from the display

#### Scenario: Popup dismissed on chord completion
- **WHEN** the user completes a chord while the popup is visible
- **THEN** the popup hides immediately

### Requirement: Which-key popup is positioned bottom-right and non-blocking
The system SHALL position the popup at the bottom-right of the window (fixed, 24px from edges). It SHALL NOT block interaction with other UI elements.

#### Scenario: Popup does not block clicks
- **WHEN** the which-key popup is visible
- **THEN** the user can click on any UI element and the popup hides

#### Scenario: Popup positioned bottom-right
- **WHEN** the popup is shown
- **THEN** it appears at fixed bottom: 24px, right: 24px with z-index above all content

### Requirement: Which-key delay is user-configurable
The system SHALL expose a whichKeyDelayMs setting (default 500ms, range 0–2000ms) that controls how long after a chord prefix the popup appears.

#### Scenario: Zero delay shows popup instantly
- **WHEN** whichKeyDelayMs is 0 and user presses a chord prefix
- **THEN** the popup appears immediately
