## ADDED Requirements

### Requirement: SolidJS Web Dashboard
The system SHALL provide a Web-based dashboard built with SolidJS and Tailwind for monitoring agents.

#### Scenario: Visualizing thought process in Web UI
- **WHEN** a user selects an `AgentRun` in the Web interface
- **THEN** the system SHALL display a Gantt-style chart of its LLM thinking and tool execution phases
- **AND** verify its correctness via **Playwright** E2E tests

### Requirement: SolidJS TUI Dashboard (OpenTUI)
The system SHALL provide a high-performance terminal dashboard built with SolidJS and OpenTUI.

#### Scenario: Real-time log streaming in TUI
- **WHEN** an agent emits a message or trace via gRPC
- **THEN** the SolidJS TUI SHALL reactively update the log view with zero-latency
- **AND** use **Yoga Flexbox** for consistent terminal layout

### Requirement: Shared Logic Layer
The TUI and WebUI SHALL share a common TypeScript library for state management (Solid Store) and gRPC client interactions.

#### Scenario: Unified state across interfaces
- **WHEN** a user "pauses" an agent in the TUI
- **THEN** the WebUI SHALL reactively show the "Paused" state without requiring a refresh
