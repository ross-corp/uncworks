## ADDED Requirements

### Requirement: LiteLLM URL is user-configurable
The system SHALL allow the user to configure the LiteLLM base URL. The default SHALL be `http://litellm:4000`. The user MAY set any HTTP/HTTPS URL (in-cluster or external OpenAI-compatible endpoint).

#### Scenario: Default URL used when not configured
- **WHEN** no LiteLLM URL is configured
- **THEN** all LLM calls use `http://litellm:4000` as the base URL

#### Scenario: Custom URL configured
- **WHEN** the user sets a custom LiteLLM URL in wizard step 3 or Settings
- **THEN** all LLM calls use the custom URL

### Requirement: LiteLLM provider check is a wizard gate
Wizard step 3 SHALL call `GET /models` on the configured LiteLLM URL and check for at least one available model. If zero models are found, the wizard SHALL show a warning with a link to LiteLLM provider configuration docs.

#### Scenario: Providers present — step auto-advances
- **WHEN** LiteLLM returns one or more models
- **THEN** the wizard shows a success indicator listing the model count and enables "Finish"

#### Scenario: No providers configured
- **WHEN** LiteLLM returns zero models or an error
- **THEN** the wizard shows a warning: "No models found. Please configure at least one provider in LiteLLM." with a "Open LiteLLM dashboard" link and a "Skip for now" option

#### Scenario: LiteLLM unreachable
- **WHEN** the `GET /models` request fails (connection refused, timeout)
- **THEN** the wizard shows "Could not reach LiteLLM at <url>" and offers URL edit + retry, plus "Skip for now"

### Requirement: LiteLLM shown in Services list
The system SHALL include a LiteLLM entry in the services/health panel. Its health status SHALL be determined by `GET /health` on the configured LiteLLM URL.

#### Scenario: LiteLLM healthy
- **WHEN** `GET /health` returns 200
- **THEN** the Services list shows "LiteLLM" with a green status indicator

#### Scenario: LiteLLM unhealthy
- **WHEN** `GET /health` returns a non-200 or times out
- **THEN** the Services list shows "LiteLLM" with a red status indicator and the error message
