## ADDED Requirements

### Requirement: Regression tests are tagged with a build tag
All regression scenario tests SHALL use `//go:build regression` at the top of their file and live in `test/regression/`.

#### Scenario: Regression tests excluded from normal test run
- **WHEN** `go test ./...` is run without build tags
- **THEN** regression-tagged tests SHALL NOT be compiled or executed

#### Scenario: Regression tests included with tag
- **WHEN** `go test -tags regression ./test/regression/...` is run
- **THEN** all regression tests SHALL be compiled and executed

### Requirement: A Taskfile task runs the regression suite
A `test:regression` task SHALL exist in `Taskfile.yml` that runs all regression-tagged Go tests plus the Playwright smoke spec.

#### Scenario: Regression task runs all scenarios
- **WHEN** `task test:regression` is executed
- **THEN** it SHALL run `go test -tags regression` on regression test files AND `playwright test e2e/smoke.spec.ts`

#### Scenario: Regression task fails if any scenario fails
- **WHEN** any regression scenario fails
- **THEN** the `test:regression` task SHALL exit non-zero

### Requirement: Regression suite covers the critical paths
The regression suite SHALL include tests for: full run lifecycle, webhook delivery, project provisioning, chain execution with dependencies, auth boundary (unauthenticated → 401), and rate limiting threshold.

#### Scenario: Run lifecycle regression
- **WHEN** the regression suite runs
- **THEN** it SHALL execute a complete pending→running→complete lifecycle test with LiteLLM stubbed

#### Scenario: Auth boundary regression
- **WHEN** the regression suite runs
- **THEN** it SHALL verify that requests without a valid auth token receive HTTP 401

#### Scenario: Webhook delivery regression
- **WHEN** the regression suite runs
- **THEN** it SHALL verify that a run completion triggers a webhook POST to the configured endpoint

#### Scenario: Rate limiting regression
- **WHEN** the regression suite runs
- **THEN** it SHALL verify that exceeding the configured rate limit returns HTTP 429

### Requirement: Regression gate runs in CI on PRs and release tags
The CI pipeline SHALL run the regression suite on every PR targeting `main` and on every push matching `v*` tags.

#### Scenario: Regression gate on PR
- **WHEN** a pull request is opened or updated targeting `main`
- **THEN** the CI regression gate SHALL run and block merge if it fails

#### Scenario: Regression gate on release tag
- **WHEN** a tag matching `v*` is pushed
- **THEN** the CI regression gate SHALL run before any release artifacts are published

#### Scenario: Regression gate not blocking normal pushes
- **WHEN** a push is made directly to a non-main branch without a PR
- **THEN** the regression gate SHALL NOT run (to preserve fast feedback on feature branches)
