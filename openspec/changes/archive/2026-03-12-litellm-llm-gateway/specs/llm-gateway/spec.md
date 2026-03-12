## ADDED Requirements

### Requirement: LiteLLM Proxy Deployment
LiteLLM proxy SHALL be deployed as an explicit infrastructure dependency alongside PostgreSQL and Temporal.

#### Scenario: LiteLLM is available in-cluster
- **WHEN** the AOT system is deployed
- **THEN** LiteLLM proxy SHALL be accessible via in-cluster DNS at the configured `LITELLM_BASE_URL`

### Requirement: Agent Pod LLM Routing
All agent pods SHALL route LLM API calls through the LiteLLM proxy.

#### Scenario: Agent makes an LLM API call
- **WHEN** an agent pod issues a request to the OpenAI-compatible API
- **THEN** the request SHALL be routed through the LiteLLM proxy's `/v1` endpoint via the `OPENAI_BASE_URL` environment variable

### Requirement: Agent Pod Environment Variables
Agent pods SHALL receive `OPENAI_BASE_URL` and `OPENAI_API_KEY` as environment variables.

#### Scenario: Agent pod is created
- **WHEN** the controller builds an agent pod spec
- **THEN** the agent container SHALL include `OPENAI_BASE_URL` pointing to `$LITELLM_BASE_URL/v1`
- **AND** the agent container SHALL include `OPENAI_API_KEY` containing the provisioned virtual key

### Requirement: LiteLLM OpenAI-Compatible Endpoint
`OPENAI_BASE_URL` SHALL point to the LiteLLM proxy's `/v1` endpoint.

#### Scenario: Agent SDK compatibility
- **WHEN** an agent uses any OpenAI-compatible SDK (OpenAI Python, LangChain, etc.)
- **THEN** the SDK SHALL connect to LiteLLM transparently via the standard `OPENAI_BASE_URL` environment variable
- **AND** no agent code changes SHALL be required

### Requirement: Model Backend Configuration
LiteLLM proxy SHALL be configured with at least one model backend.

#### Scenario: Minimum viable deployment
- **WHEN** LiteLLM is deployed
- **THEN** at least one model backend (Ollama, OpenRouter, or a premium provider) SHALL be configured in the model list

### Requirement: Fallback Routing
LiteLLM proxy SHALL support fallback routing between model backends.

#### Scenario: Primary backend is unavailable
- **WHEN** the primary model backend (e.g., Ollama) is unavailable or returns an error
- **THEN** LiteLLM SHALL automatically route the request to the configured fallback backend (e.g., OpenRouter free tier)

#### Scenario: Fallback chain for default tier
- **WHEN** a request targets the `default` model tier
- **THEN** LiteLLM SHALL try Ollama first, then fall back to OpenRouter free tier (`default-cloud`)

### Requirement: Reference Configuration
LiteLLM configuration SHALL be provided as a reference config in `deploy/litellm/litellm-config.yaml`.

#### Scenario: Deploying LiteLLM with reference config
- **WHEN** an operator deploys LiteLLM for AOT
- **THEN** the reference config SHALL include model_list entries for Ollama, OpenRouter, and premium providers
- **AND** the reference config SHALL include fallback routing rules
- **AND** the reference config SHALL include database connection settings for virtual key storage

### Requirement: Configurable LiteLLM Endpoint
Connection to LiteLLM SHALL be configurable via `LITELLM_BASE_URL` environment variable with a default of `http://litellm:4000`.

#### Scenario: Default configuration
- **WHEN** `LITELLM_BASE_URL` is not set
- **THEN** the system SHALL use `http://litellm:4000` as the LiteLLM proxy address

#### Scenario: Custom configuration
- **WHEN** `LITELLM_BASE_URL` is set to a custom value (e.g., `http://litellm.infra.svc:4000`)
- **THEN** the system SHALL use that value as the LiteLLM proxy address

### Requirement: Health Check Endpoint
LiteLLM proxy SHALL expose a health check endpoint for Kubernetes readiness and liveness probes.

#### Scenario: Kubernetes probes
- **WHEN** LiteLLM is deployed as a Kubernetes service
- **THEN** the deployment SHALL configure readiness and liveness probes against LiteLLM's health endpoint
- **AND** the proxy SHALL be removed from service if it fails health checks
