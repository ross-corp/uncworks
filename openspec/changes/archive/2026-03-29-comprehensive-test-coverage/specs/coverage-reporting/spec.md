## ADDED Requirements

### Requirement: Go test coverage is collected in CI
The CI pipeline SHALL collect Go test coverage using `-coverprofile=coverage.out -covermode=atomic` on every test run and emit the coverage artifact.

#### Scenario: Coverage file generated
- **WHEN** the CI `Test()` Dagger function runs
- **THEN** a `coverage.out` file SHALL be produced alongside test results

#### Scenario: HTML coverage report generated
- **WHEN** `coverage.out` exists after a CI test run
- **THEN** `go tool cover -html=coverage.out -o coverage.html` SHALL produce a browsable HTML report

#### Scenario: Coverage summary printed
- **WHEN** CI runs
- **THEN** `go tool cover -func coverage.out | grep total` SHALL print the total coverage percentage to stdout

### Requirement: TypeScript web coverage is collected via Vitest
The web frontend SHALL have a Vitest configuration with `@vitest/coverage-v8` that produces LCOV output.

#### Scenario: Unit test coverage script exists
- **WHEN** `npm run test:coverage` is executed in `web/`
- **THEN** it SHALL run `vitest run --coverage` and produce `coverage/lcov.info`

#### Scenario: Coverage report includes hooks and views
- **WHEN** coverage is collected
- **THEN** `web/src/hooks/` and `web/src/views/` SHALL be included in the coverage report

### Requirement: Node package coverage is collected via Vitest
`packages/shared` and `packages/pi-aot-extension` SHALL have Vitest configured with coverage output, replacing `node --test`.

#### Scenario: Shared package coverage
- **WHEN** `npm run test:coverage` is run in `packages/shared`
- **THEN** it SHALL produce LCOV coverage output for all `src/` files

#### Scenario: Extension package coverage
- **WHEN** `npm run test:coverage` is run in `packages/pi-aot-extension`
- **THEN** it SHALL produce LCOV coverage output for all `src/` files

### Requirement: Coverage thresholds are enforced in CI
CI SHALL check coverage against defined per-package thresholds and fail if any threshold is breached.

#### Scenario: Threshold check passes
- **WHEN** coverage for `internal/server/` is at or above 70%
- **THEN** the CI coverage check step SHALL pass

#### Scenario: Threshold check fails
- **WHEN** coverage for `internal/server/` drops below 70%
- **THEN** the CI coverage check step SHALL fail with a clear message indicating which package and current percentage

#### Scenario: Initial thresholds are non-breaking
- **WHEN** thresholds are first introduced
- **THEN** they SHALL be set at or below the measured baseline so day-1 CI does not fail
