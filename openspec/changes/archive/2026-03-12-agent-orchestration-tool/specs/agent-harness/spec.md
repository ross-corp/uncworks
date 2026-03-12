## ADDED Requirements

### Requirement: RPC Gateway Sidecar
The system SHALL provide an RPC Gateway sidecar in every Agent Pod that translates gRPC streams into standard I/O for the harness.

#### Scenario: Forwarding LLM prompt to harness
- **WHEN** the API Server sends a message via gRPC to the RPC Gateway
- **THEN** the RPC Gateway SHALL write the message to the harness's stdin

### Requirement: Human-in-the-Loop (HITL) Interruption
The Agent Harness Extension SHALL support pausing the execution loop for human input.

#### Scenario: Requesting human input
- **WHEN** the agent uses the `/ask_human` tool
- **THEN** the execution loop SHALL pause and emit a `WaitingForInput` event to the Control Plane
