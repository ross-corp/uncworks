## ADDED Requirements

### Requirement: Modal for human-in-the-loop input
When a run enters the waiting_for_input phase, the system SHALL display a modal dialog rather than a footer bar.

#### Scenario: Modal appears automatically on waiting_for_input
- **WHEN** a run's phase transitions to waiting_for_input
- **THEN** a modal dialog opens automatically with the agent's prompt text
- **AND** the input field is auto-focused

#### Scenario: Header badge persists if modal is dismissed
- **WHEN** the user dismisses the modal without submitting
- **THEN** a persistent amber badge appears in the run header reading "Waiting for Input"
- **AND** clicking the badge re-opens the modal

#### Scenario: Confirmation on input submission
- **WHEN** the user submits input via the modal
- **THEN** a toast confirms "Input sent · Resuming run"
- **AND** the modal closes and the amber badge disappears

#### Scenario: Distinct visual state in RunListView
- **WHEN** a run is in waiting_for_input phase
- **THEN** the run row shows an amber pulsing badge distinct from the blue "running" badge
- **AND** the status filter includes "waiting" as a filterable state
