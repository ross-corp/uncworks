## ADDED Requirements

### Requirement: Playwright tests run in every CI pipeline
The CI pipeline SHALL execute all Playwright e2e specs on every push, using the `mcr.microsoft.com/playwright` Docker image.

#### Scenario: Playwright function exists in Dagger module
- **WHEN** `dagger call playwright-tests --source .` is invoked
- **THEN** it SHALL install dependencies, start the Vite dev server, run all specs, and return test output

#### Scenario: Playwright included in All() pipeline
- **WHEN** `dagger call all --source .` runs
- **THEN** Playwright tests SHALL run alongside Go tests, lint, and type checking

#### Scenario: Playwright failure blocks CI
- **WHEN** any Playwright spec fails
- **THEN** the CI pipeline SHALL fail and report which spec(s) failed

### Requirement: Playwright tests require no real backend
All Playwright specs SHALL use `page.route()` to mock API responses so they can run without any cluster or API server.

#### Scenario: API routes are intercepted
- **WHEN** a Playwright test starts
- **THEN** all `**/api/v1/**` requests SHALL be intercepted and returned from test fixtures

#### Scenario: Dev server starts automatically
- **WHEN** Playwright tests run in CI
- **THEN** the Vite dev server SHALL start automatically via `webServer` config and be ready before specs execute

### Requirement: Playwright retries on flakiness
CI SHALL retry each failing spec up to 2 times before marking it failed.

#### Scenario: Transient failure retried
- **WHEN** a spec fails on the first attempt due to a timing issue
- **THEN** Playwright SHALL retry up to 2 times before reporting failure
