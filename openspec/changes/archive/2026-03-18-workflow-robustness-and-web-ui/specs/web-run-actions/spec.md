## ADDED Requirements

### Requirement: Cancel agent run from detail view
The web UI SHALL display a Cancel button on the run detail view when the run is in a non-terminal phase (Pending, Running, WaitingForInput). Clicking SHALL show a confirmation prompt. On confirm, the system SHALL call AOTClient.cancelAgentRun.

#### Scenario: Cancel a running agent
- **WHEN** user clicks Cancel on a Running run and confirms
- **THEN** the system calls cancelAgentRun with the run ID
- **AND** the run phase updates to Cancelled via the event stream

#### Scenario: Cancel button hidden for terminal phases
- **WHEN** the run phase is Succeeded, Failed, or Cancelled
- **THEN** the Cancel button SHALL NOT be displayed

### Requirement: Send human input from detail view
The web UI SHALL display a text input form when the run phase is WaitingForInput. On submit, the system SHALL call AOTClient.sendHumanInput with the run ID and input text.

#### Scenario: Send input to waiting agent
- **WHEN** run phase is WaitingForInput and user types input and submits
- **THEN** the system calls sendHumanInput with the run ID and input text
- **AND** the input field clears on success

#### Scenario: Input form hidden when not waiting
- **WHEN** run phase is not WaitingForInput
- **THEN** the human input form SHALL NOT be displayed
