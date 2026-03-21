# smoke-tests Specification

## Purpose
TBD - created by archiving change comprehensive-test-strategy. Update Purpose after archive.
## Requirements
### Requirement: Full spec-driven pipeline succeeds end-to-end
The system SHALL have a smoke test that creates a spec-driven run with a simple prompt and verifies it reaches SUCCEEDED phase.

#### Scenario: Plan-Execute-Verify pipeline completes
- **WHEN** a spec-driven run is created with prompt "Create a file called TEST.md with test content"
- **THEN** the run reaches AGENT_RUN_PHASE_SUCCEEDED within 15 minutes

### Requirement: File explorer works during hydration and execution
The system SHALL have a smoke test that verifies the files endpoint returns data during workspace hydration and after agent execution.

#### Scenario: Files available during run
- **WHEN** a run reaches RUNNING phase
- **THEN** `GET /api/v1/runs/{id}/files?path=/workspace` returns a non-empty entries array

#### Scenario: Hidden dirs filtered
- **WHEN** files are listed for a workspace
- **THEN** the response does not contain entries named ".aot" or ".bare"

### Requirement: Shell WebSocket upgrade succeeds for running pods
The system SHALL have a smoke test that verifies the exec WebSocket endpoint returns 101 Switching Protocols when a pod is running.

#### Scenario: WebSocket 101 during execution
- **WHEN** a run has a running pod and a WebSocket upgrade request is sent to `GET /api/v1/runs/{id}/exec`
- **THEN** the server responds with HTTP 101 Switching Protocols

### Requirement: Trace spans match activity feed entries
The system SHALL have a smoke test that verifies trace span counts match structured log entry counts for tool calls and LLM responses.

#### Scenario: Tool span count matches activity
- **WHEN** a completed run has N tool_call entries in structured logs
- **THEN** the traces endpoint returns at least N spans of type "tool"

#### Scenario: LLM span count matches activity
- **WHEN** a completed run has M assistant entries in structured logs
- **THEN** the traces endpoint returns at least M spans of type "llm"

### Requirement: Git push creates feature branch
The system SHALL have a smoke test that verifies autoPush creates a git branch on the remote repository.

#### Scenario: Feature branch pushed
- **WHEN** a spec-driven run with autoPush=true succeeds
- **THEN** a branch named "aot/{run-id}" exists on the remote repository

### Requirement: PR creation returns valid URL
The system SHALL have a smoke test that verifies autoPR creates a GitHub PR and returns the URL.

#### Scenario: PR created on success
- **WHEN** a spec-driven run with autoPush=true and autoPR=true succeeds
- **THEN** the run status contains a non-empty prUrl field matching "https://github.com/*/pull/*"

### Requirement: OpenSpec validation catches invalid specs
The system SHALL have a smoke test that verifies the plan stage rejects specs that don't contain SHALL or MUST requirements.

#### Scenario: Invalid spec rejected
- **WHEN** the plan agent produces a spec without SHALL/MUST requirement text
- **THEN** `openspec validate` returns validation errors and the pipeline reports them in the failure message

