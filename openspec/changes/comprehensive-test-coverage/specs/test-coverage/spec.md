## ADDED Requirements

### Requirement: Coverage instrumentation is added to all test commands
Every `go test` invocation in `ci/main.go` and `Taskfile.yml` SHALL include `-coverprofile` and `-covermode=atomic` flags.

#### Scenario: Taskfile test:go emits coverage
- **WHEN** `task test:go` is run
- **THEN** a `coverage.out` file SHALL be produced in the working directory

#### Scenario: CI Dagger Test function emits coverage
- **WHEN** the `Test()` Dagger function runs
- **THEN** `coverage.out` SHALL be exported as a file artifact

### Requirement: Coverage thresholds are defined per package group
Minimum coverage thresholds SHALL be defined for package groups and enforced in CI.

#### Scenario: Server package at threshold
- **WHEN** `internal/server/` coverage is at or above the configured threshold (initially 70%)
- **THEN** the coverage check step SHALL pass

#### Scenario: Controller package at threshold
- **WHEN** `internal/controller/` coverage is at or above the configured threshold (initially 60%)
- **THEN** the coverage check step SHALL pass

#### Scenario: Temporal package at threshold
- **WHEN** `internal/temporal/` coverage is at or above the configured threshold (initially 55%)
- **THEN** the coverage check step SHALL pass
