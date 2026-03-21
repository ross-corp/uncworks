## Why

Every bug fixed in the last two sessions occurred at a component boundary where data crosses from one system to another — proto to CRD, CRD to workflow, sidecar to server, server to frontend. Unit tests pass but the system breaks because contracts between components are untested. We need three layers of tests that catch boundary bugs before they ship.

## What Changes

- Add **contract tests** that verify data flows correctly across every component boundary (no infra needed, run in CI)
- Add **integration tests** that test multi-component behavior with real I/O (local process, no k8s)
- Add **smoke tests** that verify the deployed system works end-to-end (need k8s cluster)
- Fix stale tests that reference old workspace paths (`/workspace/src/`)

## Capabilities

### New Capabilities
- `contract-tests`: Fast, deterministic tests at every component boundary — proto↔CRD, CRD↔workflow, sidecar↔server spans, nginx↔backend routes, workflow↔activities field mapping, frontend↔backend type contracts
- `integration-tests`: Multi-component tests with real I/O — sidecar span capture, hydrator workspace layout, JSONL parser consistency, loop detection, structured log + thinking parser agreement
- `smoke-tests`: Deployed system verification — full pipeline e2e, file explorer during hydration, shell WebSocket, trace span completeness, git push/PR creation

### Modified Capabilities
- None (this adds test infrastructure, doesn't change existing specs)

## Impact

- **Test files**: New test files in `test/contract/`, `test/integration/`, `e2e/`
- **CI**: Contract and integration tests run on every PR; smoke tests run on deploy
- **Dependencies**: No new dependencies — uses Go testing, existing e2e harness
- **Coverage target**: Every boundary that produced a bug in this session gets a contract test
