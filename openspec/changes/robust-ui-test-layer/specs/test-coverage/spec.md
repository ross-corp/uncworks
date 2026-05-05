## ADDED Requirements

### Requirement: All view components have at least one integration test
Every view component in `web/src/views/` SHALL have at least one test in `web/src/views/__tests__/` that exercises a real user flow using MSW (not module-level mocks of `useClient` or `apiFetch`).

#### Scenario: New view component added without test
- **WHEN** a PR adds a new file in `web/src/views/` with no corresponding `__tests__/` file
- **THEN** CI fails with a lint or coverage gate error

#### Scenario: Existing views covered
- **WHEN** `task test:ui` runs
- **THEN** all 17 view components have at least one passing test

### Requirement: Frontend test task runs in CI
A `task test:ui` Taskfile target SHALL run all frontend tests (Vitest) and exit non-zero on any failure. This target SHALL be added to the PR CI workflow.

#### Scenario: test:ui runs clean
- **WHEN** `task test:ui` runs on main branch
- **THEN** exit code is 0 and no test is skipped

#### Scenario: Failing test blocks PR
- **WHEN** a PR introduces a test failure
- **THEN** CI fails and the PR cannot be merged

### Requirement: MSW is used instead of module-level mocks for API calls
Tests that exercise view components SHALL use MSW handlers to intercept `fetch` rather than mocking `useClient`, `apiFetch`, or the client module directly. The existing 3 view tests SHALL be migrated to MSW.

#### Scenario: apiFetch is not mocked at module level in view tests
- **WHEN** a view test file is linted
- **THEN** there are no `vi.mock('../hooks/apiFetch')` or `vi.mock('../hooks/useClient')` calls in the file

### Requirement: e2e task exists for release verification
A `task test:e2e:app` Taskfile target SHALL run Playwright e2e tests against the installed `/Applications/UNCWORKS.app`. The task SHALL print a clear error and exit 1 if the app is not running.

#### Scenario: e2e task runs Playwright
- **WHEN** `task test:e2e:app` is called with UNCWORKS.app running
- **THEN** Playwright connects and runs all e2e specs
