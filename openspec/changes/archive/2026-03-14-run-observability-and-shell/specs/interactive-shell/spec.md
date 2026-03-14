## ADDED Requirements

### Requirement: Interactive shell access via WebSocket
The API server SHALL expose a WebSocket endpoint that provides interactive terminal access to agent pod containers.

#### Scenario: Shell session opens
- **WHEN** the web UI connects to `GET /api/v1/runs/{id}/exec` (WebSocket upgrade)
- **THEN** an interactive bash session starts in the agent pod's rpc-gateway container
- **AND** the working directory is `/workspace`

#### Scenario: Terminal I/O round-trip
- **WHEN** the user types a command in the Shell tab
- **THEN** the keystrokes are sent via WebSocket to the pod
- **AND** the command output is returned and rendered in xterm.js

#### Scenario: Terminal resize
- **WHEN** the browser window or panel is resized
- **THEN** the terminal dimensions are sent to the pod via a resize message
- **AND** the shell session adjusts its PTY size

#### Scenario: Pod not available
- **WHEN** a shell connection is attempted for a run whose pod no longer exists
- **THEN** the Shell tab shows "Pod is no longer available" with the retention expiry time if applicable

### Requirement: Shell rendered with xterm.js
The Shell tab SHALL provide a full terminal experience using xterm.js.

#### Scenario: Terminal rendering
- **WHEN** the shell session is active
- **THEN** xterm.js renders all terminal output including ANSI colors, cursor movement, and full-screen applications (vim, less)
