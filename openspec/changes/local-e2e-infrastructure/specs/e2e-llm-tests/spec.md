## ADDED Requirements

### Requirement: E2E lifecycle test with real LLM
The system SHALL have an E2E test that creates an AgentRun, waits for the agent pod to complete with a real LLM, and verifies the workflow reaches Succeeded phase.

#### Scenario: Agent completes simple task
- **WHEN** an AgentRun is created with a deterministic prompt targeting a local LLM
- **THEN** the agent pod starts, the LLM processes the prompt, and the workflow completes within 5 minutes

### Requirement: Deterministic E2E prompts
E2E tests SHALL use prompts that produce verifiable output regardless of LLM model quality.

#### Scenario: File creation prompt
- **WHEN** the prompt instructs the agent to create a specific file with specific content
- **THEN** the test can verify completion by checking the workflow reached Succeeded phase

### Requirement: E2E HITL test with real LLM
The system SHALL have an E2E test that exercises the human-in-the-loop flow with a real agent pod.

#### Scenario: Agent waits for input then resumes
- **WHEN** an AgentRun's workflow reaches WaitingForInput state
- **THEN** the test sends human input via SendHumanInput and the workflow resumes and completes

### Requirement: E2E multi-agent test
The system SHALL have an E2E test that verifies the spawn_junior workflow creates a child agent run.

#### Scenario: Parent spawns junior agent
- **WHEN** a parent AgentRun's workflow spawns a junior agent
- **THEN** a child workflow is created and the parent waits for it to complete

### Requirement: LLM infrastructure deployment
The k0s cluster SHALL have Ollama and LiteLLM deployed for E2E testing.

#### Scenario: Local LLM available
- **WHEN** `task k0s:ollama` and `task k0s:litellm` are run
- **THEN** the LiteLLM proxy is accessible from within the cluster and can serve inference requests via the `ci` model alias
