## ADDED Requirements

### Requirement: Configurable default model per agent phase
The system SHALL allow the user to configure a separate default model for the manage phase and the implement phase. These defaults SHALL be stored in `config.json` as `defaultManageModel` and `defaultImplementModel`.

#### Scenario: Default models shown in Settings
- **WHEN** the user opens Settings
- **THEN** two model picker fields are shown: "Default manage model" and "Default implement model", populated from available LiteLLM models

#### Scenario: Defaults applied to new runs
- **WHEN** a new run is submitted without explicit model overrides
- **THEN** the run uses `defaultManageModel` for manage-phase agents and `defaultImplementModel` for implement-phase agents

#### Scenario: No default configured
- **WHEN** no default model is set for a phase
- **THEN** LiteLLM selects the model (no model parameter sent in the request)

### Requirement: Model picker shows available models from LiteLLM
The system SHALL populate model pickers (in Settings and run submission) by calling `GET /models` on the configured LiteLLM URL. If LiteLLM is unreachable, the picker SHALL show a text input with the last-known value.

#### Scenario: Models loaded from LiteLLM
- **WHEN** the model picker is rendered and LiteLLM is reachable
- **THEN** a dropdown shows all available model IDs from LiteLLM's `/models` response

#### Scenario: LiteLLM unreachable during model picker
- **WHEN** LiteLLM is not reachable when the model picker renders
- **THEN** a free-text input is shown with a "Could not load models" hint
