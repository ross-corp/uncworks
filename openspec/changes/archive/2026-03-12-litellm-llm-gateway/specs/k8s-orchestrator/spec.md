## MODIFIED Requirements

### Requirement: Agent Pod LLM Environment Variables
Agent container spec SHALL include LLM gateway environment variables for transparent proxy routing.

#### Scenario: OPENAI_BASE_URL injection
- **WHEN** the controller builds an agent pod spec
- **THEN** the agent container SHALL include an `OPENAI_BASE_URL` environment variable
- **AND** the value SHALL be derived from the `LITELLM_BASE_URL` configuration with `/v1` appended (e.g., `http://litellm:4000/v1`)

#### Scenario: OPENAI_API_KEY injection
- **WHEN** the controller builds an agent pod spec
- **THEN** the agent container SHALL include an `OPENAI_API_KEY` environment variable
- **AND** the value SHALL contain the provisioned LiteLLM virtual key for that AgentRun

### Requirement: Model Tier Selection
`AgentRunSpec` MAY include a `model_tier` field to control model routing through LiteLLM.

#### Scenario: Default model tier
- **WHEN** an `AgentRun` is created without a `model_tier` field
- **THEN** the system SHALL use `"default"` as the model tier
- **AND** the provisioned virtual key SHALL be restricted to default and default-cloud tier models

#### Scenario: Explicit model tier
- **WHEN** an `AgentRun` is created with `model_tier: "premium"`
- **THEN** the provisioned virtual key SHALL authorize access to premium tier models
