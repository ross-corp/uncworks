## ADDED Requirements

### Requirement: Per-AgentRun Virtual Key Provisioning
A LiteLLM virtual key SHALL be provisioned for each AgentRun at workflow start.

#### Scenario: Workflow starts a new AgentRun
- **WHEN** the Temporal workflow begins execution for an AgentRun
- **THEN** a LiteLLM virtual key SHALL be provisioned via the LiteLLM Admin API (`POST /key/generate`)
- **AND** the virtual key SHALL be unique to that AgentRun

### Requirement: Temporal Activity for Key Provisioning
Virtual key provisioning SHALL be implemented as a Temporal activity (`ProvisionLLMKey`).

#### Scenario: ProvisionLLMKey activity executes
- **WHEN** the `ProvisionLLMKey` activity is invoked
- **THEN** it SHALL call the LiteLLM Admin API to generate a new virtual key
- **AND** it SHALL return the generated key for injection into the agent pod

### Requirement: Per-Agent Budget Cap
Each virtual key SHALL have a configurable `max_budget` (USD) per agent run.

#### Scenario: Agent exceeds budget
- **WHEN** an agent's LLM spend reaches the `max_budget` on its virtual key
- **THEN** LiteLLM SHALL reject further requests from that key with a budget exceeded error

#### Scenario: Budget is configurable
- **WHEN** a virtual key is provisioned
- **THEN** the `max_budget` SHALL be set based on system configuration (e.g., per-tier defaults or per-agent overrides)

### Requirement: Model Tier Restrictions
Each virtual key SHALL have model restrictions based on the agent spec's model tier.

#### Scenario: Default tier agent
- **WHEN** an agent has `modelTier: "default"`
- **THEN** the virtual key SHALL only authorize access to `default` and `default-cloud` tier models

#### Scenario: Premium tier agent
- **WHEN** an agent has `modelTier: "premium"`
- **THEN** the virtual key SHALL authorize access to premium provider models (Anthropic, OpenAI) in addition to default tiers

### Requirement: Virtual Key Injection
Virtual key SHALL be injected as `OPENAI_API_KEY` environment variable in the agent pod.

#### Scenario: Agent pod receives key
- **WHEN** the agent pod is created after key provisioning
- **THEN** the `OPENAI_API_KEY` environment variable SHALL contain the provisioned LiteLLM virtual key
- **AND** the agent SHALL use this key for all LLM API calls through the proxy

### Requirement: Virtual Key Revocation on Completion
Virtual key SHALL be revoked via Temporal activity (`RevokeLLMKey`) on workflow completion.

#### Scenario: RevokeLLMKey activity executes
- **WHEN** the `RevokeLLMKey` activity is invoked
- **THEN** it SHALL call the LiteLLM Admin API (`POST /key/delete`) to revoke the virtual key

### Requirement: Revocation on All Terminal States
Virtual key revocation SHALL occur on success, failure, cancellation, and TTL expiry.

#### Scenario: Successful completion
- **WHEN** an AgentRun workflow completes successfully
- **THEN** the `RevokeLLMKey` activity SHALL be executed

#### Scenario: Failed completion
- **WHEN** an AgentRun workflow fails
- **THEN** the `RevokeLLMKey` activity SHALL be executed

#### Scenario: Cancellation
- **WHEN** an AgentRun workflow is cancelled
- **THEN** the `RevokeLLMKey` activity SHALL be executed

#### Scenario: TTL expiry
- **WHEN** an AgentRun exceeds its TTL
- **THEN** the `RevokeLLMKey` activity SHALL be executed

### Requirement: Master Key Storage
LiteLLM master key for admin API access SHALL be stored as a Kubernetes Secret.

#### Scenario: Controller accesses LiteLLM Admin API
- **WHEN** the controller or Temporal worker needs to provision or revoke virtual keys
- **THEN** it SHALL read the LiteLLM master key from a Kubernetes Secret
- **AND** the master key Secret SHALL NOT be mounted into agent pods

### Requirement: Retry on LiteLLM Unavailability
If LiteLLM is unavailable, virtual key provisioning activity SHALL retry with exponential backoff.

#### Scenario: LiteLLM temporarily unavailable
- **WHEN** the `ProvisionLLMKey` activity fails to reach the LiteLLM Admin API
- **THEN** Temporal SHALL retry the activity with exponential backoff per the activity's retry policy
- **AND** the AgentRun SHALL not proceed until a virtual key is successfully provisioned

### Requirement: Spend Tracking
Spend per agent run SHALL be queryable via LiteLLM's spend tracking API.

#### Scenario: Querying spend for an AgentRun
- **WHEN** an operator or system queries spend for a specific AgentRun
- **THEN** the spend SHALL be retrievable via LiteLLM's `/spend/logs` API filtered by the virtual key associated with that AgentRun
