## MODIFIED Requirements

### Requirement: SendHumanInput RPC Routing (MODIFIED)
The `SendHumanInput` RPC handler SHALL send a Temporal signal instead of directly calling the RPC Gateway sidecar.

#### Scenario: Client sends human input
- **WHEN** a client calls the `SendHumanInput` RPC with an `agent_run_id` and `input` text
- **THEN** the gRPC server handler SHALL look up the Temporal workflow ID associated with the agent run
- **AND** the handler SHALL send a `human-input` signal to the Temporal workflow with the input payload
- **AND** the handler SHALL return `accepted: true` once the signal is successfully dispatched

#### Scenario: Signal delivery to non-existent workflow
- **WHEN** a client calls `SendHumanInput` for an agent run whose Temporal workflow has already completed or does not exist
- **THEN** the handler SHALL return an appropriate error indicating the agent run is not active

### Requirement: CancelAgentRun RPC Routing (MODIFIED)
The `CancelAgentRun` RPC handler SHALL cancel the Temporal workflow instead of directly deleting the pod.

#### Scenario: Client cancels an agent run
- **WHEN** a client calls the `CancelAgentRun` RPC with an agent run `id`
- **THEN** the gRPC server handler SHALL look up the Temporal workflow ID associated with the agent run
- **AND** the handler SHALL cancel the Temporal workflow
- **AND** the workflow's cancellation logic SHALL handle agent stopping and pod cleanup
- **AND** the handler SHALL return the updated `AgentRun` with phase set to `CANCELLED`

#### Scenario: Cancelling an already-completed agent run
- **WHEN** a client calls `CancelAgentRun` for an agent run whose Temporal workflow has already completed
- **THEN** the handler SHALL return the current `AgentRun` state without error

### Requirement: SolidJS Web Dashboard (UNCHANGED)
The system SHALL provide a Web-based dashboard built with SolidJS and Tailwind for monitoring agents.

#### Scenario: Visualizing thought process in Web UI
- **WHEN** a user selects an `AgentRun` in the Web interface
- **THEN** the system SHALL display a Gantt-style chart of its LLM thinking and tool execution phases
- **AND** verify its correctness via **Playwright** E2E tests

### Requirement: SolidJS TUI Dashboard (OpenTUI) (UNCHANGED)
The system SHALL provide a high-performance terminal dashboard built with SolidJS and OpenTUI.

#### Scenario: Real-time log streaming in TUI
- **WHEN** an agent emits a message or trace via gRPC
- **THEN** the SolidJS TUI SHALL reactively update the log view with zero-latency
- **AND** use **Yoga Flexbox** for consistent terminal layout

### Requirement: Shared Logic Layer (UNCHANGED)
The TUI and WebUI SHALL share a common TypeScript library for state management (Solid Store) and gRPC client interactions.

#### Scenario: Unified state across interfaces
- **WHEN** a user "pauses" an agent in the TUI
- **THEN** the WebUI SHALL reactively show the "Paused" state without requiring a refresh
