## MODIFIED Requirements

### Requirement: Human-in-the-Loop (HITL) Input Routing (MODIFIED)
HITL input routing SHALL change from direct sidecar calls to Temporal signal-driven delivery. The RPC Gateway sidecar proto contract does NOT change.

#### Scenario: Delivering human input via Temporal signal
- **WHEN** a `human-input` signal is received by the `AgentRunWorkflow`
- **THEN** the workflow SHALL execute the `ForwardHumanInput` activity
- **AND** the activity SHALL make the same gRPC call to the RPC Gateway sidecar as the previous direct routing
- **AND** the sidecar SHALL process the input identically regardless of whether it was routed via Temporal or directly

#### Scenario: Sidecar proto contract unchanged
- **WHEN** the `ForwardHumanInput` activity calls the RPC Gateway sidecar
- **THEN** the gRPC request message format SHALL be identical to the existing `SendHumanInput` sidecar RPC
- **AND** no changes to the sidecar proto definition SHALL be required

### Requirement: RPC Gateway Sidecar (UNCHANGED)
The system SHALL provide an RPC Gateway sidecar in every Agent Pod that translates gRPC streams into standard I/O for the harness.

#### Scenario: Forwarding LLM prompt to harness
- **WHEN** the API Server sends a message via gRPC to the RPC Gateway
- **THEN** the RPC Gateway SHALL write the message to the harness's stdin

### Requirement: Human-in-the-Loop (HITL) Interruption (UNCHANGED)
The Agent Harness Extension SHALL support pausing the execution loop for human input.

#### Scenario: Requesting human input
- **WHEN** the agent uses the `/ask_human` tool
- **THEN** the execution loop SHALL pause and emit a `WaitingForInput` event to the Control Plane

### Requirement: spawn_junior Routing (MODIFIED)
`spawn_junior` routing SHALL change from direct CRD creation to Temporal child workflow invocation.

#### Scenario: Junior agent spawned via child workflow
- **WHEN** the agent harness invokes `spawn_junior`
- **THEN** the request SHALL be routed to the parent `AgentRunWorkflow` (via signal or activity)
- **AND** the parent workflow SHALL start a child `AgentRunWorkflow` for the junior agent
- **AND** the sidecar's `spawn_junior` handling logic SHALL send the request upstream to the Temporal workflow rather than creating a CRD directly
