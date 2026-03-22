## ADDED Requirements

### Requirement: Progressive runs support dual model selection
When creating a Progressive (spec-driven) run, the user SHALL be able to select separate models for the manage and implement agents.

#### Scenario: Dual model selectors visible in Progressive mode
- **WHEN** a user selects "Progressive" orchestration mode in the New Run view
- **THEN** two model selectors SHALL appear: one labeled "Manage model" and one labeled "Implement model"

#### Scenario: Implement model defaults to manage model
- **WHEN** a user selects a manage model but leaves the implement model unset
- **THEN** the implement agent SHALL use the same model as the manage agent

#### Scenario: Different models for each role
- **WHEN** a user selects "qwen3:8b" for manage and "deepseek-v3.1" for implement
- **THEN** the plan/verify stages SHALL use qwen3:8b AND the execute stage SHALL use deepseek-v3.1

### Requirement: Dual model config propagates to workflow
The system SHALL pass the manage and implement model selections to the spec-driven workflow.

#### Scenario: Model config in workflow input
- **WHEN** a Progressive run is created with dual models
- **THEN** the workflow input SHALL include `manageModel` and `implementModel` fields

#### Scenario: Sidecar receives correct model per stage
- **WHEN** the sidecar starts an agent for the PLAN stage
- **THEN** it SHALL use the manage model AND when starting EXECUTE, it SHALL use the implement model
