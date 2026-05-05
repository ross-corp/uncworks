## Why

The UNCWORKS frontend has 21 view components but only 3 are tested (~14% coverage), and those tests mock all dependencies — they verify component state changes, not real user flows. Every regression in recent sessions (broken selects, project click crash, settings not persisting, copilot silent failure) was a frontend bug that no test would have caught, and was only discovered through manual use. The app cannot be used to build itself until we can ship frontend changes with confidence.

## What Changes

- Add Playwright-based end-to-end tests that drive the real Wails app window against the live cluster
- Add integration-level component tests using MSW (Mock Service Worker) to intercept real API calls at the network boundary — no mocked hooks or mocked clients
- Replace existing component tests that mock `useClient`/`apiFetch` with tests that use MSW handlers so the data pipeline (serialization, type mapping, `mapRun`, etc.) is actually exercised
- Add a `task test:ui` target that runs the full frontend test suite
- Add a `task test:e2e:app` target that runs Playwright against the installed `/Applications/UNCWORKS.app`
- CI gates: frontend tests run on every PR; e2e tests run on release tags

## Capabilities

### New Capabilities

- `ui-integration-tests`: MSW-based integration tests for all 21 view components covering the critical user flows (load project, view runs, create run, send copilot message, save settings, navigate)
- `ui-e2e-tests`: Playwright end-to-end tests that drive the full Wails desktop app window against the live dev cluster

### Modified Capabilities

- `test-coverage`: Extend coverage requirements to include frontend integration and e2e test layers; existing backend requirements unchanged

## Impact

- `web/` — new `src/**/__tests__/` files; MSW handlers in `src/mocks/`; Playwright config at `web/playwright.config.ts`
- `package.json` in `web/` — adds `msw`, `@playwright/test` dev dependencies
- `Taskfile.yml` — new `test:ui` and `test:e2e:app` targets
- CI workflow — frontend test step added to PR checks
- No changes to production code; no API or schema changes
