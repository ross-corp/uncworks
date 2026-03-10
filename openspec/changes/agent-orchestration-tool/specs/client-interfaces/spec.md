## ADDED Requirements

### Requirement: TUI Fleet Dashboard
The system SHALL provide a Bubbletea-based terminal dashboard to monitor all active agents.

#### Scenario: Real-time log streaming in TUI
- **WHEN** an agent emits a message or trace
- **THEN** the TUI SHALL stream the log output to the terminal with zero-latency

### Requirement: Web-based OTel Visualization
The system SHALL provide a web interface to visualize agent execution traces using OpenTelemetry data.

#### Scenario: Visualizing thought process in Web UI
- **WHEN** a user selects an `AgentRun` in the Web interface
- **THEN** the system SHALL display a Gantt-style chart of its LLM thinking and tool execution phases
