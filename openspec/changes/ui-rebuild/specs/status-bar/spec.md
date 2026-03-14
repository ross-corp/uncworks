## ADDED Requirements

### Requirement: Persistent status bar
The application SHALL display a status bar at the very bottom of the viewport, below the log stream. The status bar SHALL show: system health indicator (green/amber/red dot), count of active runs, WebSocket/gRPC connection status, and keyboard shortcut hints for the current context.

#### Scenario: Status bar shows active run count
- **WHEN** there are 3 runs in Running phase
- **THEN** the status bar displays "3 active runs" with a phosphor green indicator

#### Scenario: Connection status indicator
- **WHEN** the WebSocket connection to the backend is healthy
- **THEN** the status bar shows a green "Connected" indicator
- **AND** when the connection drops, it shows a red "Disconnected" indicator with a reconnecting animation

#### Scenario: Keyboard shortcut hints
- **WHEN** the workspace is in Graph view
- **THEN** the status bar shows contextual hints: "Ctrl+L Logs | Ctrl+F Files | Ctrl+T Terminal | Ctrl+K Command"

#### Scenario: System health aggregation
- **WHEN** all backend services (API, Temporal, K8s) are responsive
- **THEN** the status bar shows a green health dot
- **AND** when any service is unresponsive, the dot changes to amber or red with a tooltip showing which service is down
