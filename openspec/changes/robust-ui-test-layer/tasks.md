## 1. MSW Infrastructure

- [x] 1.1 Add `msw` dev dependency to `web/package.json`
- [x] 1.2 Create `web/src/mocks/server.ts` — Vitest `setupServer` with lifecycle hooks (beforeAll/afterEach/afterAll)
- [x] 1.3 Create `web/src/mocks/fixtures.ts` — typed factory functions for `AgentRun`, `Project`, `ChatMessage`, `ServiceInfo`, `AppSettings`
- [x] 1.4 Create `web/src/mocks/handlers.ts` — happy-path MSW handlers for all API endpoints used by view components
- [x] 1.5 Wire `server.ts` into Vitest setup via `web/vite.config.ts` or `web/vitest.setup.ts`
- [x] 1.6 Add `@types/node` if needed for MSW Node integration in Vitest

## 2. Core View Integration Tests

- [x] 2.1 Write `ProjectListView.test.tsx` — projects load, empty state, click navigates (replaces existing mock-heavy version)
- [x] 2.2 Write `ProjectDetailView.test.tsx` — runs tab fetches + `mapRun` phase mapping + empty run list
- [x] 2.3 Write `RunListView.test.tsx` — runs load, status filter via CustomSelect, null response regression (replaces existing)
- [x] 2.4 Write `NewRunView.test.tsx` — form submit fires `POST /api/v1/runs` with correct payload; model CustomSelect selectable
- [x] 2.5 Write `SettingsView.test.tsx` — namespace change saved; litellmURL change hides LiteLLM service row (replaces existing)
- [x] 2.6 Write `RunDetailView.test.tsx` — phase badges render from MSW run fixture
- [x] 2.7 Write `Layout.test.tsx` — active route nav item highlighted

## 3. Additional View Tests

- [x] 3.1 Write `ChainNewView.test.tsx` — form submission calls `POST /api/v1/chains`
- [x] 3.2 Write `ScheduleNewView.test.tsx` — form submission calls `POST /api/v1/schedules`
- [x] 3.3 Write `ChainListView.test.tsx` — list renders from MSW
- [x] 3.4 Write `TemplateNewView.test.tsx` — project ref CustomSelect selectable; form submission
- [x] 3.5 Write `TemplateListView.test.tsx` — list renders from MSW
- [x] 3.6 Write `ScheduleListView.test.tsx` — list renders from MSW
- [x] 3.7 Write `ScheduleDetailView.test.tsx` — detail renders from MSW
- [x] 3.8 Write `ChainRunListView.test.tsx` — list renders from MSW
- [x] 3.9 Write `ChainRunDetailView.test.tsx` — detail renders from MSW
- [x] 3.10 Write `FeatureDetailView.test.tsx` — detail renders with run list

## 4. Copilot Integration Test

- [x] 4.1 Add MSW SSE handler for `POST /api/v1/chat/stream` that streams 3 tokens then closes
- [x] 4.2 Write `CopilotBottomPanel.test.tsx` — message sent, tokens accumulate in reply, 502 shows toast

## 5. CustomSelect Component Tests

- [x] 5.1 Write `CustomSelect.test.tsx` — option selected via click; closes on outside click; disabled state

## 6. Playwright E2E Infrastructure

- [x] 6.1 Add `@playwright/test` dev dependency to `web/package.json`
- [x] 6.2 Create `web/playwright-app.config.ts` — CDP connect config, test directory, retries (2), 30s timeout
- [x] 6.3 Create `web/e2e-app/smoke.spec.ts` — app loads, navigate to Projects/Runs/Settings without crash
- [x] 6.4 Create `web/e2e-app/custom-select.spec.ts` — CustomSelect opens and selects in real WKWebView
- [x] 6.5 Create `web/e2e-app/copilot.spec.ts` — send message, response appears within 30s (requires live cluster)
- [x] 6.6 Create `web/e2e-app/settings-persist.spec.ts` — namespace change survives window hide/show

## 7. Taskfile and CI

- [x] 7.1 Add `test:ui` target to `tasks/test.yml` — runs `cd web && npx vitest run`
- [x] 7.2 Add `test:e2e:app` target — checks UNCWORKS.app running, then runs `cd web && npx playwright test`
- [x] 7.3 Add `build:app:devtools` target — builds UNCWORKS with `--devtools` flag for CDP e2e attach
- [x] 7.4 Add `test:ui` to the PR CI workflow — `ci/main.go` `Check()` already runs `npx vitest run`
- [ ] 7.5 Add `test:e2e:app` as a manual trigger + release-tag gate in CI (requires macOS runner with live app; deferred)

## 8. Migrate Existing Tests

- [x] 8.1 Remove `vi.mock('../hooks/useClient')` from `ProjectListView.test.tsx` — now uses MSW handler
- [x] 8.2 `RunListView.test.tsx` — hybrid: MSW for null-response regression; useClient mock retained for listAgentRuns (hook wraps gRPC, not plain fetch)
- [x] 8.3 `SettingsView.test.tsx` — uses `useSettings` mock (correct; no useClient dependency)
- [x] 8.4 Verify `task test:ui` passes with zero skipped tests — 28 files, 122 tests, 0 skipped ✓
