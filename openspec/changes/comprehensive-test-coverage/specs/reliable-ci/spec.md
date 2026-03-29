## ADDED Requirements

### Requirement: Playwright e2e runs in CI on every push
The CI pipeline SHALL run Playwright specs as part of every push, not just locally.

#### Scenario: Playwright in All() pipeline
- **WHEN** `dagger call all --source .` runs
- **THEN** Playwright tests SHALL execute and a failure SHALL fail the pipeline

### Requirement: Coverage reporting runs in CI on every push
The CI pipeline SHALL collect and publish coverage reports on every push.

#### Scenario: Go coverage artifact published
- **WHEN** CI runs
- **THEN** `coverage.html` SHALL be available as a downloadable artifact

#### Scenario: TypeScript coverage artifact published
- **WHEN** CI runs
- **THEN** `web/coverage/lcov.info` SHALL be available as a downloadable artifact

### Requirement: Regression gate runs in CI on PRs to main
A dedicated regression check SHALL block merges to `main` if critical-path tests fail.

#### Scenario: Regression check on PR
- **WHEN** a pull request targets `main`
- **THEN** the regression check job SHALL run and be required to pass before merge
