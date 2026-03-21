## ADDED Requirements

### Requirement: LLM thought spans include token usage
The system SHALL extract token usage from pi's message_end events and include it in thought span metadata following OTel GenAI conventions.

#### Scenario: Token counts in thought span
- **WHEN** the sidecar processes a message_end event with usage `{input_tokens: 1200, output_tokens: 340}`
- **THEN** the corresponding thought span's metadata SHALL include `gen_ai.usage.input_tokens: 1200` and `gen_ai.usage.output_tokens: 340`

#### Scenario: Cache token tracking
- **WHEN** the message_end event includes `cache_read_input_tokens: 800`
- **THEN** the span metadata SHALL include `gen_ai.usage.cache_read_tokens: 800`

#### Scenario: Context window utilization
- **WHEN** a thought span uses 13000 input tokens with model deepseek-v3.1 (32K context window)
- **THEN** the span metadata SHALL include `gen_ai.context.utilization_pct: 40`

### Requirement: Model name in span metadata
The system SHALL include the model name in thought span metadata.

#### Scenario: Model from environment
- **WHEN** the sidecar knows the model via PI_MODEL environment variable
- **THEN** thought span metadata SHALL include `gen_ai.request.model: "deepseek-v3.1"`

### Requirement: Stage parent spans aggregate token usage
The system SHALL roll up token counts from all child thought spans into the stage parent span.

#### Scenario: Stage aggregates child tokens
- **WHEN** a PLAN stage has 3 thought spans with input tokens [1200, 800, 1600]
- **THEN** the PLAN stage span's metadata SHALL include `gen_ai.usage.input_tokens: 3600`

### Requirement: Cost estimation on stage and root spans
The system SHALL estimate USD cost from token counts using per-model pricing tables.

#### Scenario: Cost on stage span
- **WHEN** a stage uses deepseek-v3.1 with 4000 input tokens ($0.15/M) and 1090 output tokens ($0.75/M)
- **THEN** the stage span's metadata SHALL include `gen_ai.cost.stage_usd` approximately $0.0014
