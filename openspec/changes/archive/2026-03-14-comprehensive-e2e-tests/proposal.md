## Why

The platform has unit tests, contract tests, and Temporal workflow tests, but no comprehensive end-to-end test suite that verifies the full pipeline — from API call through CRD creation, Temporal orchestration, pod hydration, real LLM-driven agent execution, and back to status updates. The existing Playwright tests are stale (reference old UI elements) and the Go E2E tests use fake repo URLs that can't actually clone. We need tests that prove the entire system works with real git repos, real LLM inference, and real browser interactions — runnable locally against the aot-local cluster.

## What Changes

- **Soft-Serve git server integration** — embed Charmbracelet's Soft-Serve as a local git server for tests, replacing fake `github.com/example/repo.git` URLs with real cloneable repos. Managed as a process via Taskfile, shared across Go and Playwright test suites.
- **Test fixture repository** — a minimal repo at `test/fixtures/e2e-repo/` with `devbox.json`, `README.md`, and `main.go`, pushed to Soft-Serve during test setup. Additional fixtures for multi-repo and spec-driven scenarios.
- **API-driven E2E test suite (Go)** — comprehensive tests covering full agent lifecycle with real LLM (Ollama qwen2.5:0.5b), spec-driven runs, multi-repo workspaces, webhook receiver, GitHub push/pull endpoints, TTL expiry, and concurrent runs. All tests clone from Soft-Serve and execute against the live aot-local cluster.
- **Playwright UI E2E test suite (TS)** — browser tests covering every user journey: run creation (prompt and spec modes), workspace presets, multi-repo form, detail panel with HITL input, status watching, filtering/search, repo registry, and Monaco editor interactions. Tests run against real API with real data.
- **data-testid instrumentation** — add structured `data-testid` attributes to all interactive web components to enable reliable Playwright selectors.
- **Taskfile integration** — new `test:e2e:full` command that starts Soft-Serve, pushes fixtures, runs Go E2E tests, then runs Playwright tests, all against the aot-local cluster.

## Capabilities

### New Capabilities
- `e2e-test-harness`: Soft-Serve git server lifecycle management, fixture repo setup/teardown, test environment configuration, and shared helpers for both Go and Playwright test suites
- `api-e2e-tests`: Go E2E tests covering full agent lifecycle (create → hydrate → run → complete), spec-driven runs, multi-repo workspaces, webhook receiver, GitHub API endpoints, cancellation, HITL, TTL expiry, and error states
- `playwright-e2e-tests`: Browser E2E tests covering run creation, workspace presets, spec editor, HITL interaction, status watching, filtering, search, repo registry, and all modal interactions
- `ui-test-instrumentation`: Structured data-testid attributes across all web components enabling reliable Playwright selectors

### Modified Capabilities
<!-- No existing spec-level requirements change -->

## Impact

- **New dependency**: `github.com/charmbracelet/soft-serve` (or binary install) for local git server
- **Test fixtures**: `test/fixtures/e2e-repo/` — minimal repo content for E2E tests
- **Go E2E tests** (`e2e/`): New and updated test files, new test harness with Soft-Serve lifecycle
- **Playwright tests** (`web/e2e/`): Complete rewrite — 6-8 spec files replacing the stale `app.spec.ts`
- **Web components** (`web/src/components/*.tsx`): All components gain `data-testid` attributes (no behavior change)
- **Playwright config** (`web/playwright.config.ts`): Adjusted timeouts for real API/LLM operations
- **Taskfile** (`Taskfile.yml`): New E2E tasks for Soft-Serve lifecycle and full test orchestration
- **Infrastructure requirements**: Ollama with `qwen2.5:0.5b` model, aot-local cluster running
