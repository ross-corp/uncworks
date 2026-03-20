## Purpose

Show the agent's in-progress thinking in the activity feed in real-time.

## ADDED Requirements

### Requirement: Thinking endpoint returns partial agent text
The API SHALL provide a `/logs/thinking` endpoint that returns the agent's current in-progress text from the JSONL stream.

#### Scenario: Agent is thinking
- **WHEN** the agent is generating a response and `message_update` events exist without a closing `message_end`
- **THEN** the endpoint returns `{"thinking": true, "text": "partial text..."}`

#### Scenario: No active thinking
- **WHEN** all messages are complete (every `message_start` has a `message_end`)
- **THEN** the endpoint returns `{"thinking": false}`

### Requirement: Activity feed shows thinking indicator
The activity feed SHALL display a dimmed, italic "thinking" entry with a pulsing indicator while the agent is generating a response.

#### Scenario: Thinking visible during active run
- **WHEN** the run phase is "running" and the thinking endpoint returns text
- **THEN** a thinking entry appears at the bottom of the activity feed

#### Scenario: Thinking replaced by completed message
- **WHEN** a completed message arrives in the structured logs
- **THEN** the thinking entry is replaced by the completed entry
