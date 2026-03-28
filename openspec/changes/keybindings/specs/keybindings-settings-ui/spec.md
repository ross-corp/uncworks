## ADDED Requirements

### Requirement: Settings has a Keybindings section
The system SHALL show a Keybindings section in the Settings page with preset selection, delay configuration, and per-binding edit capability.

#### Scenario: Preset selection
- **WHEN** the user selects a preset radio button
- **THEN** the binding table updates to show the selected preset's bindings

#### Scenario: Preset switch with existing overrides shows warning
- **WHEN** the user has custom overrides and selects a new preset
- **THEN** an inline warning is shown: "Switching presets will clear your custom overrides" with a confirm button

### Requirement: Individual bindings can be recaptured
The system SHALL allow the user to click an edit icon on any binding row to enter capture mode, press a new key sequence, and save it.

#### Scenario: Key capture
- **WHEN** user clicks the edit icon for a binding and presses a new key sequence
- **THEN** the binding is updated and stored as an override

#### Scenario: Conflict detection
- **WHEN** the captured key sequence is already bound to another action
- **THEN** an inline warning shows which action has the conflict; user can confirm (unbinding the other) or cancel

#### Scenario: Reset single binding
- **WHEN** user clicks the reset icon for a binding that has an override
- **THEN** that override is removed and the preset default is restored
