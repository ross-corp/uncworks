## ADDED Requirements

### Requirement: Real-time event streaming on detail view
The run detail view SHALL use AOTClient.watchAgentRun to receive events via server-streaming instead of polling. Events SHALL be displayed in a scrollable event log panel. The stream SHALL auto-reconnect on disconnection.

#### Scenario: Events appear in real-time
- **WHEN** user views a run detail page
- **THEN** the system starts a watchAgentRun stream
- **AND** log, tool_call, and phase_changed events appear in the event log as they arrive

#### Scenario: Stream reconnects on disconnect
- **WHEN** the watchAgentRun stream disconnects
- **THEN** the system automatically reconnects after a brief delay
- **AND** the user sees a visual indicator during reconnection

#### Scenario: Stream cleanup on navigation
- **WHEN** user navigates away from the detail view
- **THEN** the watchAgentRun stream is aborted (AbortController)

### Requirement: Event log panel
The detail view SHALL include an event log panel showing events with timestamp, type badge, and payload. The panel SHALL auto-scroll to the latest event. Events SHALL be stored in the agent store (max 1000).

#### Scenario: Event log displays entries
- **WHEN** events are received from the stream
- **THEN** each event shows its timestamp, a colored type badge, and payload text

#### Scenario: Event log auto-scrolls
- **WHEN** new events arrive and the user has not scrolled up
- **THEN** the log auto-scrolls to show the latest event
