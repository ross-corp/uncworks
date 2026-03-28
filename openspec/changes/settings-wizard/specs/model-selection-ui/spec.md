## MODIFIED Requirements

### Requirement: Model selection is available in run submission UI
The run submission form SHALL include a model picker for each phase (manage, implement). The pickers SHALL be pre-filled with the user's configured defaults but SHALL allow per-run override.

#### Scenario: Run submitted with default models
- **WHEN** the user submits a run without changing the model pickers
- **THEN** the run uses the configured default models from Settings

#### Scenario: Run submitted with per-run model override
- **WHEN** the user changes one or both model pickers before submitting
- **THEN** the run uses the overridden model(s) for the affected phase(s) only; other runs are unaffected

#### Scenario: Model picker collapsed by default
- **WHEN** the run submission form is opened
- **THEN** the model pickers are shown in a collapsible "Advanced" section, collapsed by default, showing the current default model names as a summary
