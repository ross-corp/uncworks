## Context

The codebase has 93 Go test files and 17 TypeScript test files — the bones are there — but no coverage instrumentation anywhere and Playwright never runs in CI. Today there is no way to know if coverage is improving or declining. The Dagger CI module (`ci/main.go`) runs `go test ... -count=1` with no `-cover` flags. Vitest and `@vitest/coverage-v8` are already installed in `web/package.json` but unconfigured. All 10 Playwright specs use `page.route()` mocking and need no real backend.

## Goals / Non-Goals

**Goals:**
- Coverage reporting (HTML + LCOV) for Go, TypeScript/React, and Node packages, generated in CI and available as artifacts
- Playwright running in every CI pipeline pass
- React component + hook unit test suite (currently zero)
- Layer 2 Go tests: run lifecycle state machine with LiteLLM stubbed
- Regression gate: tagged subset of critical tests, runs on PR to main and `v*` tags

**Non-Goals:**
- 100% coverage — targets start soft and ratchet up over time
- Visual regression testing (screenshot diffing)
- Load/performance testing
- Testing the Wails desktop app binary
- Coverage for `brain/` and `embeddings/` packages (excluded from CI for now; decision deferred)

## Decisions

### Decision 1: Go coverage via `-coverprofile` + `go tool cover`, NOT a third-party service

Add `-coverprofile=coverage.out -covermode=atomic` to all `go test` invocations in `ci/main.go`. Emit `coverage.out` as a Dagger artifact. Generate `coverage.html` from it with `go tool cover -html`. Run `go tool cover -func` to get per-function stats.

**Alternatives considered:**
- Codecov/Coveralls: Adds external dependency, requires token management. Overkill for now; can integrate later.
- `gotestsum`: Nice output, but adds a dependency we don't need yet.

### Decision 2: Vitest for ALL TypeScript (web + packages), not node:test

Migrate `packages/shared` and `packages/pi-aot-extension` from `node --test` to Vitest. This gives uniform coverage output (`coverage/lcov.info`) across all three JS packages, consistent config, and better DX (watch mode, UI mode).

`web/vitest.config.ts` — configure with `jsdom` environment, `@testing-library/react`, coverage provider `v8` (already installed).

**Alternatives considered:**
- Keep `node --test` + `c8`: works but produces inconsistent reports vs Vitest; coverage merging is harder.
- Jest: heavyweight, slower startup; Vitest is already in the dependency tree.

### Decision 3: Playwright in Dagger using `mcr.microsoft.com/playwright` image

Add `PlaywrightTests()` function to `ci/main.go`. Use Microsoft's official Playwright Docker image which pre-installs Chromium, Firefox, and WebKit with correct system deps. Start the Vite dev server inside the container before running tests.

```
dagger call playwright-tests --source .
```

Include in `All()` so it gates every CI run.

**Alternatives considered:**
- `npx playwright install` in the Node base: works but adds 300MB+ to install time on every run.
- GitHub Actions `playwright-action`: loses Dagger portability and local reproducibility.

### Decision 4: Layer 2 tests live in `internal/server/` + `test/layer2/`

New test file per subsystem: `agentrun_lifecycle_test.go`, `hitl_flow_test.go`, `sse_ordering_test.go`, `trace_generation_test.go`. Use a local `httptest.Server` stub for LiteLLM that returns canned completion responses. Leverage the fake K8s client already used throughout `internal/server/`.

**Alternatives considered:**
- WireMock / testcontainers for LiteLLM: overkill; a `net/http/httptest` stub is sufficient since we only need to control response JSON.
- Real LLM in CI: excluded (cost, latency, non-determinism).

### Decision 5: Regression build tag `//go:build regression`

Critical-path Go tests are tagged with `//go:build regression`. They are NOT tagged `integration` (runs with testcontainers) or `e2e` (runs against real cluster). They run with fake clients and LLM stubs — fast, isolated, but covering the highest-value paths.

```
task test:regression   # runs: go test -tags regression ./test/regression/... ./internal/...
```

CI runs `test:regression` on every PR to `main` and every `v*` tag push via a dedicated Dagger function `RegressionTests()`.

Playwright smoke spec is included in regression by running the existing `smoke.spec.ts` file via `playwright test e2e/smoke.spec.ts`.

### Decision 6: Coverage thresholds — measure first, gate later

Week 1: Add coverage instrumentation, measure baseline, publish HTML artifacts.
Week 2+: Set initial gates at observed baseline − 5% to avoid day-1 failures.
Quarter goal: ratchet to documented targets (server: 70%, controller: 60%, temporal: 55%, web hooks: 50%).

Enforcement via `go tool cover -func coverage.out | grep total` + a small shell check in CI. Not a Go linter plugin — keeps the gate transparent.

## Risks / Trade-offs

- **Playwright flakiness in CI** → `retries: 2` already configured in `playwright.config.ts` for CI; add `--reporter=list` to surface failures clearly.
- **Dagger container size** → `mcr.microsoft.com/playwright` is ~1.5GB; first run is slow. Subsequent runs are cached by Dagger's layer cache.
- **Layer 2 test maintenance** → LiteLLM stub responses must evolve with the API. Keep stubs in `test/stubs/litellm.go` as a single source of truth.
- **Coverage noise from generated code** → Add `-coverpkg` filter to exclude `frontend/wailsjs/` and generated proto files from Go coverage counts.
- **Node package Vitest migration** → `node:test` and Vitest have slightly different assertion APIs. Test count: 7 files total. Low-risk migration.

## Migration Plan

1. Add coverage flags to Taskfile tasks and `ci/main.go` (no behavior change, just adds output)
2. Add Playwright Dagger function (additive; existing CI unaffected until `All()` is updated)
3. Configure Vitest in `web/` (additive; existing Playwright e2e unaffected)
4. Migrate `packages/shared` + `packages/pi-aot-extension` to Vitest (behavioral change in test runner; tests still pass)
5. Write Layer 2 tests (additive; no changes to production code)
6. Add regression build tag to selected existing tests + new regression scenarios
7. Wire regression gate into CI on PRs + release tags
8. Measure baseline coverage; set initial thresholds; publish coverage HTML artifacts

No rollback complexity — all changes are additive. The only behavioral change to CI is adding new steps (Playwright, regression); if they fail, existing test steps are unaffected.

## Open Questions

- Should `brain/` and `embeddings/` be included in coverage collection even if test execution remains excluded? (Currently excluded from `go test` entirely — could collect coverage from existing tests without running the expensive ones.)
- What is the right artifact retention policy for coverage HTML reports? (GitHub Actions artifacts default to 90 days — sufficient?)
- Regression suite ownership: who decides what goes in? Should there be a `REGRESSION.md` doc listing covered scenarios and owners?
