## Context

The UNCWORKS frontend is a React/TypeScript SPA embedded in a Wails v2 macOS app. It has 17 view components and 3 are tested (~18%). Existing tests mock `useClient`, `apiFetch`, and UI components entirely — so bugs in type mapping (`mapRun`), serialization, and the API data pipeline go undetected. Recent regressions (broken `<select>`, wrong `mapRun` double-call, copilot double-`/v1`, settings not persisting) were all frontend issues discovered through manual use, never by CI.

The test infrastructure already in place: Vitest + React Testing Library + jsdom. The gap is (1) no MSW layer so real network behavior is exercised, and (2) no Playwright harness for true end-to-end coverage.

## Goals / Non-Goals

**Goals:**
- Every view component has at least one integration test that exercises a real user flow through the full frontend data pipeline (fetch → deserialize → render)
- MSW intercepts API calls at the fetch boundary — no mocked hooks, no mocked clients
- A Playwright harness can drive the UNCWORKS desktop app window for critical flows
- `task test:ui` runs all frontend tests in CI under ~60 seconds
- `task test:e2e:app` drives the installed app against the dev cluster (manual / release gate)
- New frontend code that breaks a tested flow fails CI before merge

**Non-Goals:**
- 100% line coverage — we want high-value flow coverage, not exhaustive unit coverage
- Testing Wails-specific native features (clipboard, tray, dock) — those require a real macOS session and belong in manual QA
- Visual regression testing (screenshot diffing) — too brittle for rapid iteration
- Backend API contract tests — those are already covered by Go e2e tests

## Decisions

### 1. MSW over Vitest mocks for API boundaries

**Decision:** Use [MSW v2](https://mswjs.io/) to intercept `fetch` calls at the network layer in jsdom tests.

**Why:** Current tests mock `useClient` and `apiFetch` as module-level imports. This means a bug in `mapRun` (wrong field mapping), a serialization mismatch, or an API shape change passes all tests silently. MSW intercepts the actual `fetch()` call, so the full frontend data pipeline runs: `apiFetch` → JSON parse → `toAgentRun` / `mapRun` → component render. The `mapRun` double-call bug in `ProjectDetailView` would have been caught immediately.

**Alternative considered:** Keep module mocks but add integration tests against a local test server. Rejected: requires running a Go server in CI, adds latency, and couples frontend tests to backend build.

### 2. Playwright for desktop app e2e

**Decision:** Use Playwright with the `--channel=chromium` WebDriver pointing at the Wails webview via CDP (Chrome DevTools Protocol).

**Why:** Wails exposes a CDP endpoint when built with `--devtools`. Playwright can connect to it and drive the real embedded webview — same browser environment the production app uses. This catches WKWebView-specific issues (like the broken native `<select>`) that jsdom/Vitest cannot.

**Alternative considered:** Cypress. Rejected: Cypress doesn't support CDP-attach mode well; Playwright's `connectOverCDP` is purpose-built for this.

**Alternative considered:** Manual QA only for e2e. Rejected: this is the current state and it's what causes the regressions.

### 3. Test fixture factories over inline JSON

**Decision:** Create typed fixture factories in `web/src/mocks/fixtures.ts` that return minimal valid objects for each domain type (`AgentRun`, `Project`, `ChatMessage`, etc.), with overrides via spread.

**Why:** Inline JSON fixtures in tests are brittle — they break when types change and they omit required fields silently. Typed factories enforced by TypeScript catch breaking API shape changes at compile time, not at runtime.

### 4. Coverage target: flows over lines

**Decision:** Require at least one passing test for each of the 12 critical user flows (see specs), not a line-coverage percentage.

**Why:** Line coverage incentivizes testing trivial code and discourages deleting dead code. Flow coverage ensures the most important user paths are verified. The 12 flows were chosen based on the bugs found in the current session — each flow was broken at some point.

### 5. MSW handlers colocated with tests, shared via `src/mocks/handlers.ts`

**Decision:** Default MSW handlers live in `src/mocks/handlers.ts` (happy-path data). Individual tests override handlers inline using `server.use(...)` for error cases or alternate data shapes.

**Why:** Sharing defaults reduces boilerplate. Per-test overrides keep failure-path tests readable and self-contained.

## Risks / Trade-offs

- **Playwright CDP attach requires `--devtools` build flag** → For CI e2e we build the app with devtools enabled; for release builds it stays off. Mitigation: add a `UNCWORKS_DEVTOOLS=1` build tag in Taskfile.
- **jsdom doesn't support CSS transitions or `<details>` toggle events perfectly** → `CustomSelect` tests may need `userEvent.click` + explicit assertion on open state rather than relying on CSS visibility. Mitigation: test the value change behavior, not open/close animation.
- **MSW service worker in jsdom** → MSW v2 uses Node-mode (`setupServer`) for Vitest, not service workers. No browser required. No risk.
- **Playwright test flakiness** → E2e tests against a real cluster can flake if pods restart. Mitigation: retry count of 2; skip e2e in PR CI (runs only on release tags and manually).
- **Fixture maintenance** → As API shapes evolve, fixtures need updating. Mitigation: typed factories fail compilation on shape mismatch, so breakage is caught immediately.

## Migration Plan

1. Add MSW and Playwright as dev dependencies (no production impact)
2. Create `src/mocks/` infrastructure (handlers, fixtures, server setup)
3. Add integration tests for the 12 flows — no existing tests removed yet
4. Replace existing mock-heavy tests with MSW-based equivalents (3 tests)
5. Add Playwright config and smoke e2e test
6. Wire `task test:ui` into CI
7. Rollback: delete `src/mocks/` and test files; remove Taskfile targets — zero production impact

## Open Questions

- Should we run Playwright e2e in CI on every PR (requires building the app, ~5 min) or only on release tags? Leaning toward release-tags-only + manual trigger.
- Should `task test:ui:watch` be a separate target or just `vitest --watch`?
