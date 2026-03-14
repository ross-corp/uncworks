## ADDED Requirements

### Requirement: NotifyEvent transitions agent state
The sidecar SHALL implement the `AgentNotificationService.NotifyEvent` RPC. When it receives `EVENT_TYPE_WAITING_FOR_INPUT`, it SHALL set the agent process state to `AGENT_PROCESS_STATE_WAITING_FOR_INPUT`. When it receives `EVENT_TYPE_STARTED`, it SHALL set the state to `AGENT_PROCESS_STATE_RUNNING`. The handler SHALL return `acknowledged: true` on success.

#### Scenario: Agent signals waiting for input
- **WHEN** NotifyEvent is called with `event_type = EVENT_TYPE_WAITING_FOR_INPUT`
- **THEN** `GetStatus` returns `state = AGENT_PROCESS_STATE_WAITING_FOR_INPUT`
- **AND** `NotifyEvent` returns `acknowledged: true`

#### Scenario: Agent signals resumed after input
- **WHEN** NotifyEvent is called with `event_type = EVENT_TYPE_STARTED` after being in WAITING_FOR_INPUT
- **THEN** `GetStatus` returns `state = AGENT_PROCESS_STATE_RUNNING`

#### Scenario: NotifyEvent with no process running
- **WHEN** NotifyEvent is called before any agent process is started
- **THEN** the handler returns a FailedPrecondition error

#### Scenario: Other event types do not change state
- **WHEN** NotifyEvent is called with `event_type = EVENT_TYPE_LOG`
- **THEN** the agent process state remains unchanged
- **AND** `NotifyEvent` returns `acknowledged: true`
