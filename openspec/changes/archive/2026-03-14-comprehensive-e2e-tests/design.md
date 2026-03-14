## Context

The platform has a solid test pyramid (unit, contract, Temporal workflow tests) but the E2E layer is incomplete. The existing Playwright tests reference a stale UI (list-based layout, `data-testid` attributes that don't exist). The Go E2E tests use fake repo URLs (`github.com/example/repo.git`) so hydration always fails — they can only test API plumbing, not the full agent lifecycle. No test currently proves: user action → API → CRD → Temporal → Pod → hydration → LLM agent → completion → UI update.

The aot-local cluster (k0s + Temporal + controller + worker) already runs locally for development. Ollama with `qwen2.5:0.5b` (397MB) is available on the host. What's missing is a real git source for hydration and test suites that exercise the full stack.

## Goals / Non-Goals

**Goals:**
- Full lifecycle E2E: create run → pod starts → clones real repo → agent runs with real LLM → completes → status visible in UI
- Local-first: all tests run against the aot-local cluster with `task test:e2e:full`
- Soft-Serve as the git server: tests clone from a local Soft-Serve instance, no GitHub dependency
- Ollama with qwen2.5:0.5b for real LLM inference in agent pods
- Playwright tests against the live web UI + live API (no mocking the backend)
- Every user-facing UI flow covered by Playwright tests
- Structured `data-testid` attributes across all components

**Non-Goals:**
- CI pipeline integration (future — focus on local execution first)
- Multi-browser testing (Chromium only)
- Performance/load testing
- Testing KubeVirt or External backends (not implemented)
- GitHub API integration tests against real GitHub (mock in Playwright, skip in Go E2E)

## Decisions

### 1. Soft-Serve as standalone process via Taskfile

**Decision**: Run Soft-Serve as a standalone binary process, managed by Taskfile tasks. Start it before tests, push fixture repos via `git push`, stop it after tests. Both Go and Playwright suites connect to the same instance.

**Rationale**: Soft-Serve supports git:// protocol out of the box, which is unauthenticated and fast. Running it as a process (vs. in-process Go embedding or Docker) is simplest — no coupling to Soft-Serve internals, no Docker dependency beyond what the cluster already needs. The binary is a single `go install`. A shared process means both Go E2E and Playwright tests use the same git server.

**Alternative considered**: Embedding via `pkg/daemon` in Go TestMain — rejected because Playwright tests also need the git server and can't share a Go process. Docker container via testcontainers — rejected as unnecessary overhead when a binary works.

### 2. Fixture repo pushed to Soft-Serve per test session

**Decision**: Store fixture repo content at `test/fixtures/e2e-repo/` in the project. During test setup, `git init` a temp repo, copy fixtures in, commit, and `git push` to Soft-Serve. Additional fixture variants (multi-repo, spec-driven) follow the same pattern.

```
test/fixtures/
├── e2e-repo/           # Standard single-repo fixture
│   ├── devbox.json     # {"packages": []}
│   ├── main.go         # Simple Go file agent can modify
│   └── README.md
├── e2e-repo-frontend/  # Second repo for multi-repo tests
│   ├── package.json    # {"name": "e2e-frontend"}
│   └── index.ts
└── push-fixtures.sh    # Script: init + push all fixtures to soft-serve
```

**Rationale**: Fixtures in the project repo are version-controlled and reproducible. Pushing to Soft-Serve on each test session ensures a clean slate. The push script is reusable across Go and Playwright.

### 3. Ollama qwen2.5:0.5b for LLM — assumes running on host

**Decision**: Tests assume Ollama is running on the host with the `qwen2.5:0.5b` model pulled. The existing LiteLLM proxy routes agent requests to Ollama. No test-specific LLM setup — we use whatever the aot-local cluster is configured with.

**Rationale**: The aot-local dev environment already has Ollama + LiteLLM configured. Tests should verify the real pipeline, not a mock. qwen2.5:0.5b is small (397MB) and fast enough for simple prompts like "create a file called DONE.txt". Test prompts are designed to be deterministic and simple to minimize LLM variance.

**Trade-off**: Tests are slower (30-120s per agent run) but prove the real system works. We accept this for E2E — unit/contract tests remain fast.

### 4. Soft-Serve git:// protocol with configurable address

**Decision**: Tests use `git://{SOFT_SERVE_ADDR}/repo-name` URLs for agent runs. The address defaults to `localhost:9418` (Soft-Serve's default git daemon port) and is configurable via `SOFT_SERVE_ADDR` env var. The hydrator already supports any git URL — no code changes needed for cloning.

**Rationale**: git:// is unauthenticated, fast, and Soft-Serve supports it natively. No SSH key setup, no HTTP auth. The hydrator's `git clone --bare` works identically with git:// URLs.

### 5. Playwright tests against real API, GitHub endpoints mocked

**Decision**: Playwright tests hit the real web UI (localhost:3000) proxying to the real API (localhost:50055). Agent runs created in Playwright tests go through the full pipeline (real pod, real LLM). GitHub push/pull endpoints are mocked via `page.route()` since we don't want to depend on GitHub credentials in tests.

**Rationale**: The whole point of E2E is testing real integration. Mocking the API would defeat the purpose. GitHub endpoints are the one exception — they require external credentials and aren't part of the core pipeline.

### 6. data-testid naming convention

**Decision**: All interactive web components get `data-testid` attributes following the pattern `[component]-[element]-[qualifier]`:

```
sidebar-phase-{phase}           # Phase filter buttons
sidebar-workspace-{name}        # Workspace filter buttons
sidebar-repo-{encoded-url}      # Repo filter buttons
table-row-{id}                  # Table rows
table-row-{id}-phase            # Phase badge in row
detail-panel                    # Detail panel container
detail-name                     # Run name in detail
detail-phase                    # Phase in detail
detail-hitl-input               # HITL textarea
detail-hitl-send                # HITL send button
form-modal                      # Form modal container
form-name-input                 # Name field
form-repo-row-{index}-url       # Repo URL input
form-repo-row-{index}-branch    # Repo branch input
form-add-repo                   # Add repo button
form-tab-prompt                 # Prompt tab
form-tab-spec                   # Spec tab
form-submit                     # Submit button
spec-editor                     # Monaco editor container
github-modal                    # GitHub modal
workspace-editor                # Workspace editor modal
toast                           # Toast notification
```

**Rationale**: Structured naming makes selectors predictable and maintainable. Using `data-testid` over CSS selectors decouples tests from styling. The pattern is grep-able and IDE-searchable.

### 7. Playwright timeouts for real LLM operations

**Decision**: Playwright config gets a global action timeout of 5s (for UI interactions) but individual tests that wait for agent completion use explicit `expect(...).toHaveText(..., { timeout: 180_000 })` (3 minutes). A helper `waitForRunPhase(page, runId, phase, timeout)` encapsulates polling.

**Rationale**: Real agent runs with qwen2.5:0.5b take 30-120 seconds. Global timeout stays short for UI responsiveness, but lifecycle tests explicitly wait longer. The helper keeps test code clean.

### 8. Test execution via Taskfile

**Decision**: New Taskfile tasks:

```yaml
test:e2e:full        # Start soft-serve, push fixtures, run Go E2E, run Playwright, stop soft-serve
test:e2e:setup       # Start soft-serve + push fixtures only
test:e2e:teardown    # Stop soft-serve
test:e2e:go          # Run Go E2E tests only (assumes setup done)
test:e2e:playwright  # Run Playwright tests only (assumes setup done)
```

**Rationale**: Separating setup from execution lets developers iterate. Run `test:e2e:setup` once, then `test:e2e:go` or `test:e2e:playwright` repeatedly while developing tests. The `full` task does everything for a clean-room run.

## Risks / Trade-offs

**Soft-Serve availability in agent pods** — The agent pod runs inside k0s. It needs to reach Soft-Serve running on the host. → Mitigation: k0s pods can reach the host via the node IP or `host.k0s.internal`. The Soft-Serve listen address can be configured to bind `0.0.0.0` so it's reachable from pods.

**LLM non-determinism** — qwen2.5:0.5b may not always follow the prompt precisely. → Mitigation: Use extremely simple prompts ("create file X with content Y"). Accept that the test verifies the pipeline completes, not that the LLM output is perfect. Check for `Succeeded` phase, not file contents.

**Test execution time** — Full E2E suite with real LLM will take 5-15 minutes. → Mitigation: Acceptable for E2E — these aren't meant to run on every save. Go unit/contract tests stay fast. Tests can be run selectively via `-run TestE2E_Lifecycle`.

**Soft-Serve binary installation** — Developers need `soft` binary installed. → Mitigation: Add to devbox.json packages or provide `go install` command in setup task.

**Stale Playwright tests referencing old UI** — Current `web/e2e/app.spec.ts` is dead code. → Mitigation: Delete it entirely and replace with new spec files that target the current component structure.
