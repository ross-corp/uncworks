## ADDED Requirements

### Requirement: Mermaid syntax for all diagrams

All architecture and flow diagrams in markdown files SHALL use Mermaid syntax within fenced `mermaid` code blocks.

#### Scenario: new diagram uses Mermaid

- **WHEN** a developer adds an architecture or flow diagram to any `.md` file
- **THEN** the diagram is written as a fenced `mermaid` code block

### Requirement: no ASCII box-drawing diagrams

ASCII box-drawing diagrams (using `+`, `-`, `|`, and similar characters to draw boxes and connectors) SHALL NOT be used in any `.md` file for architectural or flow diagrams.

#### Scenario: ASCII diagram is rejected in review

- **WHEN** a pull request adds or modifies an `.md` file containing an ASCII box-drawing diagram
- **THEN** the reviewer requests conversion to Mermaid syntax

### Requirement: README.md architecture diagram converted to Mermaid

The architecture diagram in `README.md` SHALL be converted from ASCII art to a Mermaid diagram.

#### Scenario: README contains Mermaid architecture diagram

- **WHEN** a developer reads the `README.md` architecture section
- **THEN** the diagram is rendered via a fenced `mermaid` code block, not ASCII art

### Requirement: state machine diagrams use Mermaid stateDiagram-v2

State machine diagrams (e.g., the AgentRun lifecycle) SHALL use Mermaid `stateDiagram-v2` syntax.

#### Scenario: AgentRun lifecycle uses stateDiagram-v2

- **WHEN** a developer views the AgentRun lifecycle state diagram in documentation
- **THEN** it is rendered using a `stateDiagram-v2` Mermaid block showing states and transitions

### Requirement: sequence diagrams use Mermaid sequenceDiagram

Sequence diagrams (e.g., HITL flow, multi-agent flow) SHALL use Mermaid `sequenceDiagram` syntax.

#### Scenario: HITL flow uses sequenceDiagram

- **WHEN** a developer views the human-in-the-loop flow diagram in documentation
- **THEN** it is rendered using a `sequenceDiagram` Mermaid block showing actors and message exchanges

#### Scenario: multi-agent flow uses sequenceDiagram

- **WHEN** a developer views the multi-agent orchestration flow diagram in documentation
- **THEN** it is rendered using a `sequenceDiagram` Mermaid block

### Requirement: terminal UI mocks may use ASCII

Terminal UI mock outputs MAY remain as ASCII text. They represent actual terminal rendering, not architectural diagrams, and are exempt from the Mermaid requirement.

#### Scenario: TUI mock remains ASCII

- **WHEN** documentation includes a mock of terminal output (e.g., TUI screenshot, CLI help text)
- **THEN** ASCII formatting is acceptable and does not require conversion to Mermaid
