## ADDED Requirements

### Requirement: OpenTelemetry (OTel) Tracing
The system SHALL emit OpenTelemetry spans for every tool call and LLM interaction.

#### Scenario: Span propagation for tool calls
- **WHEN** the agent harness calls a tool (e.g., `bash`)
- **THEN** it SHALL emit an OTel span linked to the parent `AgentRun` trace ID

### Requirement: Structured Log Export
The system SHALL export JSON-formatted logs for all agent activities to a centralized collector.

#### Scenario: Aggregating logs in OTel Collector
- **WHEN** multiple agent pods are running in the cluster
- **THEN** the Control Plane SHALL aggregate their logs and traces via the OTel Collector for analysis
