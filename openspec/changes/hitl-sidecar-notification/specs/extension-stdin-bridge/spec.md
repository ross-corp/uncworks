## ADDED Requirements

### Requirement: Extension notifies sidecar of HITL state
The pi-aot-extension SHALL call the sidecar's NotifyEvent RPC when entering and exiting the waiting-for-input state. When `waitForHumanInput()` is called, the extension SHALL send `EVENT_TYPE_WAITING_FOR_INPUT`. When input is received and the Promise resolves, the extension SHALL send `EVENT_TYPE_STARTED`.

#### Scenario: waitForHumanInput sends notification
- **WHEN** `waitForHumanInput("question")` is called
- **THEN** the extension calls NotifyEvent with `EVENT_TYPE_WAITING_FOR_INPUT` and the question as payload
- **AND** the extension enters paused/waiting state

#### Scenario: Input received sends resumed notification
- **WHEN** human input is received (via stdin) while waiting
- **THEN** the extension calls NotifyEvent with `EVENT_TYPE_STARTED`
- **AND** the `waitForHumanInput` Promise resolves with the input text

### Requirement: Extension bridges stdin to Promise
The pi-aot-extension SHALL read from `process.stdin` line-by-line. When `waitForHumanInput()` has a pending Promise and a line arrives on stdin, the Promise SHALL resolve with that line. Lines received when not waiting SHALL be buffered so they are available for the next `waitForHumanInput()` call.

#### Scenario: Stdin line resolves waiting Promise
- **WHEN** `waitForHumanInput()` is pending and a line is written to stdin
- **THEN** the Promise resolves with the stdin line content

#### Scenario: Stdin line buffered when not waiting
- **WHEN** a line arrives on stdin before `waitForHumanInput()` is called
- **THEN** the line is buffered
- **AND** the next `waitForHumanInput()` call resolves immediately with the buffered line

### Requirement: Extension connects to sidecar via ConnectRPC
The extension SHALL use `@connectrpc/connect-node` to create a ConnectRPC client for `AgentNotificationService` at `http://localhost:50052`. The sidecar address SHALL be configurable via the extension config.

#### Scenario: Extension sends NotifyEvent to sidecar
- **WHEN** the extension needs to notify the sidecar
- **THEN** it uses the ConnectRPC client to call NotifyEvent on `localhost:50052`
