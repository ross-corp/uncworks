## ADDED Requirements

### Requirement: On-demand debug pod via Deployment scaling
The system SHALL provide a "Debug Run" capability that scales a completed run's Deployment from 0 to 1 in debug mode.

#### Scenario: Start debug session
- **WHEN** a user clicks "Debug Run" for a completed run
- **THEN** the Deployment is patched to replicas=1 with annotation `aot.uncworks.io/mode: debug`
- **AND** the sidecar starts in debug mode (shell access only, no agent launch)
- **AND** the Shell tab becomes active with an interactive terminal

#### Scenario: Debug pod has same workspace
- **WHEN** a debug pod starts
- **THEN** it mounts the same PVC at `/workspace`
- **AND** the workspace contains the exact files the agent left

#### Scenario: Debug pod auto-expires
- **WHEN** a debug pod has no active WebSocket or exec connections for 30 minutes
- **THEN** the Deployment is scaled back to 0

#### Scenario: Manual debug stop
- **WHEN** the user clicks "Stop Debug" or the API receives `DELETE /api/v1/runs/{id}/debug`
- **THEN** the Deployment is scaled to 0

### Requirement: Debug Run UI integration
The detail panel Shell tab SHALL show a "Debug Run" button when the Pod is not running.

#### Scenario: Shell tab for completed run
- **WHEN** a completed run is selected and the Pod is scaled to 0
- **THEN** the Shell tab shows a "Debug Run" button instead of "Pod expired"

#### Scenario: Shell tab after debug started
- **WHEN** the debug pod is running
- **THEN** the Shell tab shows the interactive terminal
- **AND** a "Stop Debug" button is available
