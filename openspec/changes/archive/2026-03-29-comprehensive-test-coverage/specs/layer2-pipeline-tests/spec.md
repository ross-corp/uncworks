## ADDED Requirements

### Requirement: A LiteLLM stub is available for tests
A reusable `httptest.Server` stub SHALL be available in `test/stubs/litellm.go` that returns configurable canned completion responses, enabling tests to exercise the full agent pipeline without a real LLM.

#### Scenario: Stub returns configured completion
- **WHEN** the stub is configured with a canned response and the code under test calls the LiteLLM `/chat/completions` endpoint
- **THEN** the stub SHALL return the configured JSON response with HTTP 200

#### Scenario: Stub records requests
- **WHEN** the stub receives a request
- **THEN** it SHALL record the request body so tests can assert on what was sent to the LLM

### Requirement: Run lifecycle state transitions are tested
Tests SHALL verify that an AgentRun progresses through `pending → running → complete` with LiteLLM stubbed.

#### Scenario: Run reaches running state
- **WHEN** an AgentRun is created with valid spec and the LiteLLM stub is running
- **THEN** the run controller SHALL transition the run to `running` phase within the test timeout

#### Scenario: Run reaches complete state
- **WHEN** the LiteLLM stub returns a completion response with no tool calls
- **THEN** the run controller SHALL transition the run to `complete` phase

#### Scenario: Run phase stored in status
- **WHEN** a phase transition occurs
- **THEN** the AgentRun's `.status.phase` SHALL reflect the new phase in the fake K8s client

### Requirement: HITL pause and resume flow is tested
Tests SHALL verify that a run pauses when waiting for human input and resumes when input is provided.

#### Scenario: Run pauses at HITL checkpoint
- **WHEN** the LiteLLM stub returns a response requesting human input
- **THEN** the run phase SHALL transition to `waiting_for_input`

#### Scenario: Run resumes after HITL input
- **WHEN** a HITL response is submitted via the API and the run is in `waiting_for_input`
- **THEN** the run phase SHALL transition back to `running`

### Requirement: Activity feed SSE events are ordered correctly
Tests SHALL verify that activity events are emitted in causal order (tool-start before tool-result, log before completion).

#### Scenario: Tool events ordered
- **WHEN** a tool is called and returns a result during a run
- **THEN** the SSE stream SHALL emit `tool_start` before `tool_result` for that tool call

#### Scenario: Completion event is last
- **WHEN** a run completes
- **THEN** the SSE stream SHALL emit the completion event after all tool and log events

### Requirement: Trace spans are generated for agent activity
Tests SHALL verify that trace spans are created for plan, execute, and verify stages.

#### Scenario: Root span created
- **WHEN** a run starts
- **THEN** a root trace span with `stage=run` SHALL be created in the trace store

#### Scenario: Stage child spans created
- **WHEN** a run with plan and execute stages completes
- **THEN** child spans for each stage SHALL be present with correct parent-child relationships

### Requirement: Transient errors trigger retry, not failure
Tests SHALL verify that a transient LiteLLM error causes retry behavior, not immediate run failure.

#### Scenario: 503 from LiteLLM triggers retry
- **WHEN** the LiteLLM stub returns HTTP 503 on the first call then 200 on the second
- **THEN** the run SHALL eventually complete rather than fail

#### Scenario: Permanent error fails the run
- **WHEN** the LiteLLM stub returns HTTP 500 on all calls beyond the retry limit
- **THEN** the run phase SHALL transition to `failed`
