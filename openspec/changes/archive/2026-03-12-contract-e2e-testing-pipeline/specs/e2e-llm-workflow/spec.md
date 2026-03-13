## ADDED Requirements

### Requirement: Full E2E Infrastructure
E2E tests SHALL run against a fully deployed system with all real dependencies.

#### Scenario: E2E deployment
- **GIVEN** the E2E test environment
- **THEN** it SHALL deploy to a k0s cluster with PostgreSQL, Temporal, LiteLLM, and Ollama

#### Scenario: LLM backend
- **GIVEN** the E2E test environment
- **THEN** it SHALL use Ollama with a minimal model (e.g., `qwen2.5:0.5b`) via LiteLLM proxy

#### Scenario: E2E setup task
- **GIVEN** the Taskfile
- **WHEN** `task test:e2e:setup` is executed
- **THEN** it SHALL deploy all E2E dependencies to the k0s cluster

#### Scenario: E2E run prerequisite
- **GIVEN** the Taskfile
- **WHEN** `task test:e2e` is executed
- **THEN** it SHALL require a running k0s cluster with all dependencies deployed

### Requirement: Agent Lifecycle E2E Test
The complete agent lifecycle SHALL be verified end-to-end.

#### Scenario: Lifecycle test
- **GIVEN** a fully deployed E2E environment
- **WHEN** an E2E lifecycle test executes
- **THEN** it SHALL: CreateAgentRun -> wait for Running -> wait for Succeeded -> verify agent output

### Requirement: HITL E2E Test
The human-in-the-loop flow SHALL be verified end-to-end.

#### Scenario: HITL flow
- **GIVEN** a fully deployed E2E environment
- **WHEN** an E2E HITL test executes
- **THEN** it SHALL: CreateAgentRun with prompt triggering ask_human -> wait for WaitingForInput -> SendHumanInput -> wait for Succeeded

### Requirement: Multi-Agent E2E Test
The multi-agent orchestration flow SHALL be verified end-to-end.

#### Scenario: Multi-agent flow
- **GIVEN** a fully deployed E2E environment
- **WHEN** an E2E multi-agent test executes
- **THEN** it SHALL: CreateAgentRun with prompt triggering spawn_junior -> verify child AgentRun created -> wait for child Succeeded -> wait for parent Succeeded

### Requirement: Cancel E2E Test
The agent cancellation flow SHALL be verified end-to-end.

#### Scenario: Cancel flow
- **GIVEN** a fully deployed E2E environment
- **WHEN** an E2E cancel test executes
- **THEN** it SHALL: CreateAgentRun -> wait for Running -> CancelAgentRun -> wait for Cancelled

### Requirement: Pod Cleanup Verification
E2E tests SHALL verify that no orphaned pods remain after agent completion.

#### Scenario: Pod cleanup
- **GIVEN** an E2E test where an agent run has completed (any terminal state)
- **WHEN** the test verifies cluster state
- **THEN** there SHALL be no orphaned agent pods remaining

### Requirement: Deterministic Test Prompts
E2E test prompts SHALL produce predictable, verifiable outcomes.

#### Scenario: Prompt design
- **GIVEN** an E2E test prompt
- **THEN** it SHALL be designed to produce deterministic, verifiable outcomes (e.g., "create a file called hello.txt with 'hello world'")
- **AND** tests SHALL verify side effects (file creation, state changes) rather than LLM output text

### Requirement: Playwright E2E via Connect
Web dashboard E2E tests SHALL use ConnectRPC streaming.

#### Scenario: Playwright transport
- **GIVEN** the Playwright E2E test suite for the web dashboard
- **THEN** it SHALL test via ConnectRPC streaming (not WebSocket)

### Requirement: E2E Test Timeout
E2E tests SHALL have bounded execution time.

#### Scenario: Per-test timeout
- **GIVEN** an E2E test case
- **THEN** it SHALL have a timeout of 5 minutes
