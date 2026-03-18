## Purpose

Define per-stage configuration for the spec-driven pipeline, allowing users to control model, timeout, retries, and failure behavior for each stage independently.

## ADDED Requirements

### Requirement: Each pipeline stage has independent configuration
The spec-driven pipeline SHALL accept per-stage configuration (model, timeout, retries, onFailure) via the `pipelineConfig` field on AgentRunSpec.

#### Scenario: Custom model per stage
- **WHEN** a run is created with `pipelineConfig.plan.model: "qwen3-coder"` and `pipelineConfig.execute.model: "mistral-small"`
- **THEN** the plan stage uses qwen3-coder and the execute stage uses mistral-small

#### Scenario: Custom timeout per stage
- **WHEN** a run is created with `pipelineConfig.execute.timeoutSeconds: 900`
- **THEN** the execute stage times out after 900 seconds, independent of other stages

#### Scenario: Custom retries per stage
- **WHEN** a run is created with `pipelineConfig.execute.maxRetries: 5`
- **THEN** the execute→verify loop retries up to 5 times on verification failure

### Requirement: Sensible defaults when config is omitted
The pipeline SHALL use production-ready defaults when `pipelineConfig` is not specified or fields are zero/empty.

#### Scenario: No config specified
- **WHEN** a spec-driven run is created without `pipelineConfig`
- **THEN** plan uses 5min timeout, execute uses 15min timeout with 3 retries, verify uses 3min timeout

#### Scenario: Partial config
- **WHEN** only `pipelineConfig.execute.model` is specified
- **THEN** all other fields use their defaults

### Requirement: OnFailure controls stage failure behavior
Each stage SHALL support an `onFailure` field that controls what happens when the stage exhausts its retries.

#### Scenario: OnFailure retry (default for execute)
- **WHEN** the execute stage fails verification and `onFailure` is "retry"
- **THEN** the stage is retried with failure context up to maxRetries

#### Scenario: OnFailure fail
- **WHEN** the plan stage fails and `onFailure` is "fail"
- **THEN** the run is immediately marked as Failed

#### Scenario: OnFailure skip
- **WHEN** the verify stage fails and `onFailure` is "skip"
- **THEN** the run proceeds as if verification passed (marks Succeeded)

### Requirement: Stage model passed to sidecar via environment variable
The pipeline SHALL pass the stage's configured model to the sidecar agent via the `PI_MODEL` environment variable in the `StartAgentRequest.env_vars` field.

#### Scenario: Model override reaches agent
- **WHEN** the plan stage has `model: "mistral-small"`
- **THEN** the sidecar's StartAgent request includes `env_vars: {"PI_MODEL": "mistral-small"}`
- **AND** the pi-coding-agent uses that model for LLM calls
