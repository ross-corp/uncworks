## ADDED Requirements

### Requirement: TUI fetches initial agent run list via gRPC
The TUI SHALL call `ListAgentRuns` on startup to populate the dashboard with current runs.

#### Scenario: Dashboard populated on startup
- **WHEN** the TUI starts and connects to the API server
- **THEN** all existing agent runs are displayed in the list view

#### Scenario: Connection failure shows error
- **WHEN** the TUI cannot connect to the API server
- **THEN** an error message is displayed with the connection target and a retry prompt

### Requirement: TUI subscribes to live updates via WatchAgentRun
The TUI SHALL call `WatchAgentRun` for the currently selected agent run and update the detail view in real time.

#### Scenario: Live phase updates in detail view
- **WHEN** the user selects an agent run AND the run transitions from Running to Succeeded
- **THEN** the detail panel updates to show the new phase without user action

#### Scenario: Selection change resubscribes
- **WHEN** the user selects a different agent run
- **THEN** the previous watch stream is cancelled AND a new watch stream is opened for the newly selected run

### Requirement: TUI sends human input via gRPC
The TUI SHALL call `SendHumanInput` when the user submits text in HITL input mode.

#### Scenario: Successful input delivery
- **WHEN** the user submits "Use Stripe for payments" for agent "run-123"
- **THEN** `SendHumanInput` is called with agent_run_id="run-123" and input="Use Stripe for payments"

#### Scenario: Input delivery failure
- **WHEN** the `SendHumanInput` call fails (e.g., agent already resumed)
- **THEN** an error message is shown at the bottom of the TUI for 3 seconds
