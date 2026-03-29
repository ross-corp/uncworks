## Why

The codebase has zero coverage reporting across all languages (Go, TypeScript, Node.js), Playwright e2e tests that exist but never run in CI, no React component unit tests, and no structured regression gate before releases. Without visibility into what's covered and automated enforcement, coverage declines silently and regressions ship.

## What Changes

- **Go coverage**: Add `-coverprofile` + `-covermode=atomic` to all CI `go test` invocations; generate HTML and LCOV artifacts from Dagger; enforce per-package thresholds (soft initially, ratcheted up over time)
- **TypeScript coverage**: Configure Vitest in `web/` (dependency already installed), add `test:unit` and `test:coverage` scripts, wire into CI Dagger module; migrate `packages/shared` and `packages/pi-aot-extension` from `node --test` to Vitest for consistent coverage output
- **Playwright in CI**: Add `PlaywrightTests()` Dagger function to `ci/main.go` using `mcr.microsoft.com/playwright` base image; include in `All()` pipeline (all 10 specs are already fully API-mocked via `page.route()` — no real backend needed)
- **React component unit tests**: Write Vitest + `@testing-library/react` tests for key hooks (`useSettings`, `usePoll`, `apiFetch`, `useThemeNew`) and views (`RunListView`, `ProjectListView`, `SettingsView`) — currently zero
- **Layer 2 pipeline tests**: Go tests for the run lifecycle state machine with LiteLLM stubbed — the largest uncovered surface in the backend (pending→running→complete transitions, HITL pause/resume, tool dispatch, SSE ordering, trace span generation, error retry)
- **Regression suite**: Curate critical-path tests under `//go:build regression` Go build tag; add `test:regression` Taskfile task; gate runs on PR to main and every `v*` release tag in CI

## Capabilities

### New Capabilities

- `coverage-reporting`: Per-language coverage collection, reporting, and threshold enforcement across Go, TypeScript web, and Node packages
- `playwright-ci`: Playwright e2e tests running in every CI pipeline (currently exist but not wired to CI)
- `react-unit-tests`: Vitest component and hook unit test suite for the React frontend
- `layer2-pipeline-tests`: Go tests for agent run lifecycle with LLM responses stubbed — covers state machine transitions, HITL flow, tool dispatch, SSE, traces, error handling
- `regression-gate`: Curated regression test suite with build tag, Taskfile task, and CI gate on PRs and releases

### Modified Capabilities

- `test-coverage`: Extends existing partial spec — adds coverage reporting requirements, browser tests, and regression gate requirements
- `reliable-ci`: Adds Playwright and coverage reporting as CI requirements

## Impact

- `ci/main.go`: New `PlaywrightTests()` and `CoverageReport()` Dagger functions; updated `All()` pipeline
- `Taskfile.yml`: Add `-coverprofile` flags to test tasks; add `test:regression`, `test:coverage:go`, `test:coverage:web`
- `web/`: New `vitest.config.ts`, new `src/__tests__/` directory, updated `package.json` scripts
- `packages/shared/` + `packages/pi-aot-extension/`: Migrate to Vitest, add coverage scripts
- `internal/server/`, `internal/controller/`, `internal/temporal/`: New Layer 2 test files with LiteLLM stub
- New `test/regression/` directory with tagged Go regression scenarios
- Go `go.mod`: No new dependencies (testify + Ginkgo already present); Vitest already in `web/package.json`
