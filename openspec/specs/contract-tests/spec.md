# contract-tests Specification

## Purpose
TBD - created by archiving change comprehensive-test-strategy. Update Purpose after archive.
## Requirements
### Requirement: Proto-CRD field mapping contract
The system SHALL have a contract test that verifies every field in the proto AgentRunSpec maps to a corresponding field in the CRD AgentRunSpec, and vice versa.

#### Scenario: All proto fields preserved in CRD
- **WHEN** a proto AgentRunSpec is created with all fields populated (including autoPush, autoPR, pipelineConfig, prBaseBranch)
- **THEN** `specProtoToCRD()` produces a CRD spec where every proto field has a corresponding non-zero CRD field

#### Scenario: All CRD fields preserved in proto response
- **WHEN** a CRD AgentRun with all status fields populated (including prUrl, stage, retryCount) is converted
- **THEN** `crdToProto()` produces a proto response where every CRD field has a corresponding non-zero proto field

### Requirement: CRD-Workflow field mapping contract
The system SHALL have a contract test that verifies every CRD spec field is mapped into the Temporal WorkflowInput struct by the controller.

#### Scenario: Pipeline config reaches workflow
- **WHEN** a CRD with PipelineConfig (plan.timeoutSeconds=900, execute.maxRetries=5) is reconciled
- **THEN** the WorkflowInput.PipelineConfig contains matching values

#### Scenario: AutoPush/AutoPR reach workflow
- **WHEN** a CRD with AutoPush=true, AutoPR=true, PRBaseBranch="develop" is reconciled
- **THEN** the WorkflowInput has AutoPush=true, AutoPR=true, PRBaseBranch="develop"

### Requirement: Sidecar span types match frontend enum
The system SHALL have a contract test that verifies every span type emitted by the sidecar is a valid value in the frontend TraceSpan.type union.

#### Scenario: All span types are valid
- **WHEN** the sidecar creates spans for agent_started, tool_execution, llm_response, and thinking events
- **THEN** every span.Type value is one of: "llm", "tool", "thought", "input", "delegate", "lifecycle"

### Requirement: Workflow-Activity field mapping contract
The system SHALL have a contract test that verifies PlanRunInput and VerifyRunInput contain all fields needed by their respective activities.

#### Scenario: PlanRunInput completeness
- **WHEN** the spec-driven workflow constructs PlanRunInput
- **THEN** it includes AgentRunName, PodIP, Prompt, SpecContent, RepoPath, and Model

#### Scenario: VerifyRunInput completeness
- **WHEN** the spec-driven workflow constructs VerifyRunInput
- **THEN** it includes AgentRunName, PodIP, ChangeName, and RepoPath

### Requirement: Nginx routes cover all backend endpoints
The system SHALL have a test that verifies the nginx proxy configuration covers every REST and gRPC endpoint registered by the API server.

#### Scenario: All API routes proxied
- **WHEN** the nginx config is parsed for proxy_pass rules
- **THEN** every route registered via `mux.HandleFunc` in the API server is covered by either the `/api/` or `/aot.api.v1.` proxy rules

### Requirement: REST response types match frontend expectations
The system SHALL have a contract test that verifies JSON field names from server REST responses match the TypeScript interface field names.

#### Scenario: AgentLogEntry matches frontend LogEntry
- **WHEN** an AgentLogEntry is serialized to JSON
- **THEN** it contains fields: timestamp, type, content, toolName, toolInput, model

#### Scenario: TraceSpan JSON matches frontend TraceSpan
- **WHEN** a server TraceSpan is serialized to JSON
- **THEN** it contains fields: id, parentId, name, type, startTime, endTime, metadata, hasDiff

