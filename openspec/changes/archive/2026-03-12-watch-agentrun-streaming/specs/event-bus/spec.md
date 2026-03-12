## ADDED Requirements

### Requirement: Event bus publishes AgentRun events to subscribers
The system SHALL provide an in-process event bus that accepts `AgentRunEvent` messages and delivers them to all subscribers registered for that run ID.

#### Scenario: Single subscriber receives events
- **WHEN** a subscriber registers for run ID "run-123" AND the controller publishes a phase-change event for "run-123"
- **THEN** the subscriber's channel receives the event within 10ms

#### Scenario: Multiple subscribers receive same event
- **WHEN** two subscribers register for run ID "run-123" AND the controller publishes an event for "run-123"
- **THEN** both subscribers receive the event

#### Scenario: Subscriber does not receive events for other runs
- **WHEN** a subscriber registers for run ID "run-123" AND the controller publishes an event for "run-456"
- **THEN** the subscriber's channel receives nothing

### Requirement: Event bus drops events for slow subscribers
The system SHALL use buffered channels (capacity 64) and SHALL NOT block the publisher if a subscriber's channel is full.

#### Scenario: Full channel causes event drop
- **WHEN** a subscriber's channel buffer is full (64 pending events) AND a new event is published
- **THEN** the new event is silently dropped for that subscriber AND the publisher is not blocked

### Requirement: Subscriber cleanup on unsubscribe
The system SHALL remove a subscriber's channel when it unsubscribes and SHALL remove the topic entry when no subscribers remain.

#### Scenario: Unsubscribe removes subscriber
- **WHEN** a subscriber unsubscribes from run ID "run-123"
- **THEN** the subscriber no longer receives events for "run-123" AND the subscriber's channel is closed

#### Scenario: Empty topic is cleaned up
- **WHEN** the last subscriber for run ID "run-123" unsubscribes
- **THEN** the topic entry for "run-123" is removed from the bus

### Requirement: Controller emits events on status changes
The `AgentRunReconciler` SHALL call `EventBus.Publish()` after every successful status subresource update with an `AgentRunEvent` containing the run ID, event type, and current phase.

#### Scenario: Phase transition emits event
- **WHEN** the controller updates an AgentRun from Pending to Running
- **THEN** an event with type PHASE_CHANGED and phase Running is published to the bus

#### Scenario: TTL expiry emits event
- **WHEN** the controller fails an AgentRun due to TTL expiry
- **THEN** an event with type COMPLETED and phase Failed is published to the bus

### Requirement: WatchAgentRun streams events from the bus
The `WatchAgentRun` gRPC RPC SHALL subscribe to the event bus for the requested run ID, send the current state as the first message, then stream subsequent events until the run completes or the client disconnects.

#### Scenario: Client receives initial state then updates
- **WHEN** a client calls WatchAgentRun for an existing run in Running phase
- **THEN** the first message contains the current AgentRun state AND subsequent phase changes are streamed as they occur

#### Scenario: Stream closes on run completion
- **WHEN** a watched run transitions to Succeeded, Failed, or Cancelled
- **THEN** the final event is sent AND the stream is closed

#### Scenario: Stream closes on client disconnect
- **WHEN** the gRPC client disconnects (context cancelled)
- **THEN** the subscriber is unsubscribed from the bus AND no goroutine leak occurs

### Requirement: WebSocket hub consumes from the event bus
The WebSocket hub SHALL subscribe to the event bus for each agent run that has active WebSocket subscribers and SHALL broadcast events to connected clients.

#### Scenario: WebSocket client receives real-time updates
- **WHEN** a browser client subscribes to run "run-123" via WebSocket AND the controller updates the run's phase
- **THEN** the WebSocket client receives a JSON event message within 100ms
