## ADDED Requirements

### Requirement: Proto Lint Enforcement
Proto definitions SHALL be validated for style consistency on every change.

#### Scenario: Running proto lint
- **GIVEN** the proto definitions in `proto/`
- **WHEN** `task proto:lint` is executed
- **THEN** it SHALL run `buf lint` against all proto files
- **AND** it SHALL fail the build on any lint violation

#### Scenario: Lint rule set
- **GIVEN** the buf lint configuration
- **THEN** it SHALL use buf's DEFAULT rule set

### Requirement: Proto Breaking Change Detection
Proto definitions SHALL be checked for backward-incompatible changes against the main branch.

#### Scenario: Running breaking change detection
- **GIVEN** the proto definitions in `proto/`
- **WHEN** `task proto:breaking` is executed
- **THEN** it SHALL run `buf breaking --against '.git#branch=main'`
- **AND** it SHALL fail the build on any breaking change

#### Scenario: Breaking change rule set
- **GIVEN** the buf breaking configuration
- **THEN** it SHALL use buf's FILE rule set (strictest level)

#### Scenario: Comparison baseline
- **GIVEN** a PR branch with proto changes
- **WHEN** `buf breaking` runs
- **THEN** it SHALL compare against the main branch HEAD

### Requirement: Schema Gate Pipeline Position
Schema contract tests SHALL be the first stage of the CI pipeline.

#### Scenario: Pipeline ordering
- **GIVEN** the CI pipeline is triggered
- **WHEN** schema contract tests are executed
- **THEN** they SHALL run before all other test stages (unit, contract, integration, E2E)

#### Scenario: Gate enforcement
- **GIVEN** the schema contract tests are running
- **WHEN** any schema test fails (lint or breaking)
- **THEN** all subsequent pipeline stages SHALL be blocked from execution
