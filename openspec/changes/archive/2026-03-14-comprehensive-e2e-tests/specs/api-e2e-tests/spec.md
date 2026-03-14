## ADDED Requirements

### Requirement: Full agent lifecycle test
The API E2E test suite SHALL verify the complete agent lifecycle from creation through LLM-driven execution to successful completion, using a real Soft-Serve repo and Ollama.

#### Scenario: Agent run succeeds end-to-end
- **WHEN** an AgentRun is created with a Soft-Serve repo URL and a simple prompt
- **THEN** the workflow transitions through Pending → Running → Succeeded
- **AND** the pod was created, hydration completed (repo cloned), and agent executed

#### Scenario: Agent run with TTL expiry
- **WHEN** an AgentRun is created with a short TTL (e.g., 10 seconds) and a prompt that takes longer
- **THEN** the workflow transitions to Failed with message containing "TTL"

### Requirement: Spec-driven run test
The API E2E test suite SHALL verify that spec-driven runs write the spec file to the workspace and execute correctly.

#### Scenario: Spec content written to workspace
- **WHEN** an AgentRun is created with `spec_content` set and no explicit prompt
- **THEN** the workflow auto-generates a codespeak build prompt
- **AND** the run proceeds through the normal lifecycle

### Requirement: Multi-repo workspace test
The API E2E test suite SHALL verify that multi-repo runs clone all repos and generate the uncspace.yaml manifest.

#### Scenario: Two repos cloned and manifested
- **WHEN** an AgentRun is created with two Soft-Serve repo URLs
- **THEN** both repos are cloned into the workspace
- **AND** the agent starts at `/workspace` (not a single repo directory)

### Requirement: Cancel and HITL tests with real agent
The API E2E test suite SHALL verify cancellation and human-in-the-loop flows against running agent pods.

#### Scenario: Cancel running agent
- **WHEN** an AgentRun reaches Running phase and CancelAgentRun is called
- **THEN** the workflow transitions to Cancelled and the pod is cleaned up

#### Scenario: HITL input forwarded to agent
- **WHEN** an AgentRun reaches WaitingForInput phase and SendHumanInput is called
- **THEN** the agent receives the input and the workflow transitions back to Running

### Requirement: Webhook receiver test
The API E2E test suite SHALL verify the GitHub webhook endpoint creates agent runs for .cs.md file changes.

#### Scenario: Webhook creates spec run
- **WHEN** a POST to `/api/v1/webhooks/github` contains a push payload with a modified `.cs.md` file
- **THEN** an AgentRun CRD is created with `spec_content` populated and `spec_source` set to `webhook:github:...`

#### Scenario: Webhook rejects invalid signature
- **WHEN** a POST to `/api/v1/webhooks/github` has an invalid `X-Hub-Signature-256` header
- **THEN** the request is rejected with HTTP 401

### Requirement: Concurrent runs test
The API E2E test suite SHALL verify that multiple agent runs can execute simultaneously without interference.

#### Scenario: Three runs complete independently
- **WHEN** three AgentRuns are created in quick succession
- **THEN** all three reach a terminal phase (Succeeded or Failed) independently
- **AND** each has a unique pod name
