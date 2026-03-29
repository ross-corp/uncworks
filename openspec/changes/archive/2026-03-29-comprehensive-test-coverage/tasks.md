## 1. Go Coverage Instrumentation

- [x] 1.1 Add `-coverprofile=coverage.out -covermode=atomic` to `Test()` function in `ci/main.go`
- [x] 1.2 Export `coverage.out` as a Dagger file artifact from `Test()`
- [x] 1.3 Add `go tool cover -html=coverage.out -o coverage.html` step in `ci/main.go` after tests
- [x] 1.4 Add `go tool cover -func coverage.out | grep total` summary print in `ci/main.go`
- [x] 1.5 Add `-coverprofile` flag to `test:go` task in `Taskfile.yml`
- [x] 1.6 Add `-coverprofile` flag to `test:contract` task in `Taskfile.yml`
- [x] 1.7 Add `test:coverage:go` Taskfile task that runs coverage + opens HTML report
- [x] 1.8 Add coverage threshold check script: fail if `internal/server/` < 50%, `internal/controller/` < 40%, `internal/temporal/` < 40%
- [x] 1.9 Wire threshold check into CI via `CheckCoverage()` Dagger function

## 2. Playwright in CI

- [x] 2.1 Add `PlaywrightTests()` function to `ci/main.go` using `mcr.microsoft.com/playwright:v1.50.0-noble` base image
- [x] 2.2 In `PlaywrightTests()`: mount source, run `npm ci` in `web/`, then `npx playwright test`
- [x] 2.3 Add `PlaywrightTests()` call to `All()` function in `ci/main.go`
- [x] 2.4 Set `CI=true` env var in the Playwright Dagger container so `playwright.config.ts` uses 1 worker and retries
- [x] 2.5 Verify `web/playwright.config.ts` `reuseExistingServer: !process.env.CI` is set (already done — confirm)
- [x] 2.6 Add `dagger call playwright-tests --source .` to `Taskfile.yml` as `test:playwright:ci`

## 3. Vitest Configuration for web/

- [x] 3.1 Create `web/vitest.config.ts` with `jsdom` environment, `@testing-library/react` setup, and `coverage: { provider: 'v8', reporter: ['text', 'lcov'] }`
- [x] 3.2 Add `@testing-library/react` and `@testing-library/user-event` to `web/package.json` devDependencies
- [x] 3.3 Add `@testing-library/jest-dom` for matcher extensions, configure in Vitest setup file
- [x] 3.4 Add `test:unit` script to `web/package.json`: `vitest run`
- [x] 3.5 Add `test:coverage` script to `web/package.json`: `vitest run --coverage`
- [x] 3.6 Create `web/src/test-setup.ts` with `@testing-library/jest-dom` import
- [x] 3.7 Add `test:unit` to the Dagger `Check()` function in `ci/main.go` after `tsc --noEmit`

## 4. Migrate Node Packages to Vitest

- [x] 4.1 Add `vitest` to `packages/shared/package.json` devDependencies
- [x] 4.2 Update `packages/shared/package.json` `test` script from `node --test ...` to `vitest run`
- [x] 4.3 Add `test:coverage` script to `packages/shared/package.json`: `vitest run --coverage`
- [x] 4.4 Create `packages/shared/vitest.config.ts` with coverage configuration
- [x] 4.5 Verify existing `packages/shared` tests pass under Vitest (fix any API differences)
- [x] 4.6 Add `vitest` to `packages/pi-aot-extension/package.json` devDependencies
- [x] 4.7 Update `packages/pi-aot-extension/package.json` `test` script to Vitest
- [x] 4.8 Add `test:coverage` script to `packages/pi-aot-extension/package.json`
- [x] 4.9 Create `packages/pi-aot-extension/vitest.config.ts` with coverage configuration
- [x] 4.10 Verify existing `packages/pi-aot-extension` tests pass under Vitest

## 5. React Component Unit Tests

- [x] 5.1 Create `web/src/hooks/__tests__/apiFetch.test.ts` — test base URL prepending and relative URL fallback
- [x] 5.2 Create `web/src/hooks/__tests__/usePoll.test.ts` — test interval callback with fake timers, cleanup on unmount
- [x] 5.3 Create `web/src/hooks/__tests__/useSettings.test.ts` — test load, cache, save round-trip with mocked Wails binding
- [x] 5.4 Create `web/src/hooks/__tests__/useThemeNew.test.ts` — test mode cycling and localStorage persistence
- [x] 5.5 Create `web/src/views/__tests__/RunListView.test.tsx` — test run rows render, empty state, navigation on click
- [x] 5.6 Create `web/src/views/__tests__/ProjectListView.test.tsx` — test project rows render, empty state, navigation on click
- [x] 5.7 Create `web/src/views/__tests__/SettingsView.test.tsx` — test field change, save button calls SaveSettings, GitHub test button visible when token configured
- [x] 5.8 Create `web/src/test-utils.tsx` with shared render wrapper (router, theme provider) for view tests

## 6. LiteLLM Stub + Layer 2 Pipeline Tests

- [x] 6.1 Create `test/stubs/litellm.go` — `httptest.Server` that returns configurable OpenAI-compatible completion responses and records requests
- [x] 6.2 Create `test/stubs/litellm_test.go` — unit test the stub itself (returns configured response, records request body)
- [x] 6.3 Create `test/layer2/agentrun_lifecycle_test.go` — test pending→running→complete state transitions with LiteLLM stub
- [x] 6.4 Create `test/layer2/hitl_flow_test.go` — test waiting_for_input pause and resume via API
- [x] 6.5 Create `test/layer2/sse_ordering_test.go` — test activity feed SSE event ordering (tool_start before tool_result, completion last)
- [x] 6.6 Create `test/layer2/trace_generation_test.go` — test root span creation and stage child spans with correct parent-child relationships
- [x] 6.7 Create `test/layer2/error_retry_test.go` — test 503 triggers retry, permanent 500 transitions to failed
- [x] 6.8 Add `test:layer2` task to `Taskfile.yml`: `go test -v ./test/layer2/... -count=1`
- [x] 6.9 Add `test:layer2` to the `test` aggregate task in `Taskfile.yml`
- [x] 6.10 Add Layer2Tests() Dagger function to `ci/main.go` and include in `All()`

## 7. Regression Suite

- [x] 7.1 Create `test/regression/` directory with `doc.go` declaring `//go:build regression` package docs
- [x] 7.2 Create `test/regression/run_lifecycle_test.go` — full lifecycle regression scenario (tags: regression)
- [x] 7.3 Create `test/regression/webhook_delivery_test.go` — run completion triggers webhook POST (tags: regression)
- [x] 7.4 Create `test/regression/project_provisioning_test.go` — project created → configRepoReady transitions (tags: regression)
- [x] 7.5 Create `test/regression/auth_boundary_test.go` — unauthenticated requests return 401 (tags: regression)
- [x] 7.6 Create `test/regression/rate_limiting_test.go` — exceeding rate limit returns 429 (tags: regression)
- [x] 7.7 Create `test/regression/chain_execution_test.go` — chain with dependencies executes in correct order (tags: regression)
- [x] 7.8 Add `test:regression` task to `Taskfile.yml`
- [x] 7.9 Add `RegressionTests()` Dagger function to `ci/main.go`
- [x] 7.10 Add `regression` CI job to `.github/workflows/ci.yml` triggered on PRs to main and `v*` tags
- [ ] 7.11 Mark regression job as required check in branch protection rules (document in CONTRIBUTING.md or README)

## 8. Baseline Measurement + Threshold Rollout

- [ ] 8.1 Run full Go test suite with coverage and record baseline per-package percentages in `openspec/specs/coverage-reporting/baseline.md`
- [ ] 8.2 Set initial CI thresholds at baseline − 5% (floor at 0%) to avoid day-1 breakage
- [ ] 8.3 Run Playwright suite and confirm all 10 specs pass in the Dagger container
- [ ] 8.4 Run React unit tests and confirm coverage output is generated
- [ ] 8.5 Run `packages/shared` and `packages/pi-aot-extension` tests under Vitest and confirm coverage output
- [ ] 8.6 Publish coverage HTML artifacts via GitHub Actions upload-artifact step
- [ ] 8.7 Add coverage badge links to README (optional, based on artifact URLs)
