## ADDED Requirements

### Requirement: WebSocket client reconnects on disconnection
The `@aot/shared` WebSocket client SHALL automatically reconnect when the connection is lost, using exponential backoff with jitter.

#### Scenario: Reconnect after network drop
- **WHEN** the WebSocket connection is unexpectedly closed
- **THEN** the client attempts reconnection after base delay (1 second) with exponential backoff

#### Scenario: Backoff increases exponentially
- **WHEN** reconnection attempts fail consecutively
- **THEN** the delay doubles each attempt (1s, 2s, 4s, 8s, 16s) up to a maximum of 30 seconds

#### Scenario: Jitter prevents thundering herd
- **WHEN** multiple clients reconnect simultaneously
- **THEN** each client adds random jitter (0-1 second) to its backoff delay

### Requirement: Reconnection resets on success
The client SHALL reset its backoff counter to zero when a reconnection succeeds and a message is received.

#### Scenario: Successful reconnect resets backoff
- **WHEN** a reconnection attempt succeeds AND the client receives a message
- **THEN** the backoff counter resets to 0 AND the next disconnection starts at base delay (1s)

### Requirement: Maximum retry limit
The client SHALL stop reconnecting after a configurable maximum number of attempts (default 10) and SHALL emit a connection-failed event.

#### Scenario: Max retries exceeded
- **WHEN** 10 consecutive reconnection attempts fail
- **THEN** the client stops attempting AND emits a "connection_failed" event to the application

### Requirement: Resubscribe on reconnect
The client SHALL re-send subscription messages for all previously subscribed run IDs when a reconnection succeeds.

#### Scenario: Subscriptions restored after reconnect
- **WHEN** a client was subscribed to runs "run-1" and "run-2" before disconnection AND reconnection succeeds
- **THEN** the client sends subscribe messages for both "run-1" and "run-2"
