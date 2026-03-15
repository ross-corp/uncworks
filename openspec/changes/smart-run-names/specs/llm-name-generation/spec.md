## ADDED Requirements

### Requirement: Generate descriptive name from prompt via LLM
The API server SHALL call the in-cluster LLM (qwen2.5:0.5b via LiteLLM proxy) at run creation time to generate a short kebab-case name from the run's prompt. The generated name SHALL be stored as `display_name` on the AgentRunSpec.

#### Scenario: Successful name generation
- **WHEN** a client calls `CreateAgentRun` with a prompt like "Fix the auth middleware to handle expired tokens"
- **THEN** the API server calls the LiteLLM proxy with the system prompt and truncated user prompt
- **AND** the LLM returns a kebab-case name like `fix-auth-token-expiry`
- **AND** the returned run's `display_name` field contains the generated name

#### Scenario: Long prompt is truncated
- **WHEN** a client calls `CreateAgentRun` with a prompt longer than 200 characters
- **THEN** the API server truncates the prompt to 200 characters before sending to the LLM
- **AND** a valid name is still generated

### Requirement: Validate generated names against regex
The API server SHALL validate generated names against the pattern `^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$`. Names that fail validation SHALL be discarded.

#### Scenario: Valid name passes validation
- **WHEN** the LLM returns `fix-auth-token-expiry`
- **THEN** the name passes regex validation
- **AND** it is stored as `display_name`

#### Scenario: Invalid name fails validation
- **WHEN** the LLM returns a name with uppercase letters, spaces, or special characters
- **THEN** the name fails regex validation
- **AND** `display_name` is left empty
- **AND** a warning is logged

#### Scenario: Name too short or too long
- **WHEN** the LLM returns a name shorter than 4 or longer than 50 characters
- **THEN** the name fails regex validation
- **AND** `display_name` is left empty

### Requirement: Fallback when LLM is unavailable
The API server SHALL fall back gracefully when the LLM call fails. Run creation SHALL NOT be blocked by name generation failures.

#### Scenario: LLM call times out
- **WHEN** the LLM call exceeds the 3-second timeout
- **THEN** `display_name` is left empty
- **AND** a warning is logged
- **AND** the run is created successfully with the K8s name as the only identifier

#### Scenario: LLM service is down
- **WHEN** the LiteLLM proxy is unreachable (connection refused, DNS failure)
- **THEN** `display_name` is left empty
- **AND** a warning is logged
- **AND** the run is created successfully

#### Scenario: LLM returns empty or garbage
- **WHEN** the LLM returns an empty string, whitespace, or non-name content (e.g., a sentence)
- **THEN** `display_name` is left empty
- **AND** the run is created successfully

### Requirement: LLM call configuration
The API server SHALL use the existing LiteLLM proxy URL configuration for name generation. The system prompt SHALL instruct the model to output only a kebab-case name.

#### Scenario: System prompt produces correct format
- **WHEN** the LLM is called with system prompt "Generate a short kebab-case name (3-5 words) for this task. Output ONLY the name, nothing else."
- **THEN** the model returns a single kebab-case string without explanation or formatting
