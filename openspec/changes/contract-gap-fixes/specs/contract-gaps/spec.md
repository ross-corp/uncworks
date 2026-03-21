## ADDED Requirements

### Requirement: Backend field mapped from CRD to WorkflowInput
The controller's `BuildWorkflowInput` SHALL copy the CRD `spec.backend` value into `WorkflowInput.Backend` as a string. The workflow SHALL have access to the requested backend type.

#### Scenario: Pod backend mapped
- **WHEN** an AgentRun CRD has `spec.backend` set to "Pod"
- **THEN** `BuildWorkflowInput` produces a WorkflowInput with `Backend` equal to "Pod"

#### Scenario: KubeVirt backend mapped
- **WHEN** an AgentRun CRD has `spec.backend` set to "KubeVirt"
- **THEN** `BuildWorkflowInput` produces a WorkflowInput with `Backend` equal to "KubeVirt"

#### Scenario: External backend mapped
- **WHEN** an AgentRun CRD has `spec.backend` set to "External"
- **THEN** `BuildWorkflowInput` produces a WorkflowInput with `Backend` equal to "External"

### Requirement: SpecSource field mapped from CRD to WorkflowInput
The controller's `BuildWorkflowInput` SHALL copy the CRD `spec.specSource` value into `WorkflowInput.SpecSource`. The workflow SHALL have access to the spec origin for traceability.

#### Scenario: SpecSource present
- **WHEN** an AgentRun CRD has `spec.specSource` set to "github:org/repo/path"
- **THEN** `BuildWorkflowInput` produces a WorkflowInput with `SpecSource` equal to "github:org/repo/path"

#### Scenario: SpecSource absent
- **WHEN** an AgentRun CRD has no `spec.specSource` set
- **THEN** `BuildWorkflowInput` produces a WorkflowInput with `SpecSource` as empty string

### Requirement: TraceSpan time fields use consistent types
The server's `TraceSpan` struct SHALL use `time.Time` for `StartTime` and `EndTime`, matching the sidecar's `TraceSpan` type. JSON serialization SHALL produce RFC 3339 strings.

#### Scenario: Server TraceSpan serializes times as RFC 3339
- **WHEN** a `server.TraceSpan` is marshaled to JSON with `StartTime` and `EndTime` set
- **THEN** the JSON contains `startTime` and `endTime` as RFC 3339 formatted strings (e.g., "2026-03-20T10:00:00Z")

#### Scenario: Server and sidecar TraceSpan types are aligned
- **WHEN** a `sidecar.TraceSpan` is converted to a `server.TraceSpan`
- **THEN** `StartTime` and `EndTime` can be assigned directly without string conversion

### Requirement: AgentLogEntry includes optional spanId
The server's `AgentLogEntry` struct SHALL include a `SpanId` field with JSON tag `spanId,omitempty`. When the agent's JSONL log contains a `spanId` value, it SHALL appear in the REST API response.

#### Scenario: Log entry with spanId
- **WHEN** an AgentLogEntry is marshaled to JSON with `SpanId` set to "span-abc123"
- **THEN** the JSON contains `"spanId":"span-abc123"`

#### Scenario: Log entry without spanId
- **WHEN** an AgentLogEntry is marshaled to JSON with `SpanId` empty
- **THEN** the JSON does NOT contain a `"spanId"` key (omitempty behavior)

## MODIFIED Requirements

### Requirement: Contract tests cover all CRD-to-WorkflowInput field mappings
The existing boundary contract test `TestBoundary_CRDToWorkflowInput_AllFieldsMapped` SHALL verify that Backend and SpecSource are included in the mapping.

#### Scenario: All spec fields accounted for
- **WHEN** the boundary test constructs a fully-populated AgentRun CRD
- **THEN** the resulting WorkflowInput has non-zero values for Backend and SpecSource
- **AND** no CRD spec field is silently dropped during mapping
