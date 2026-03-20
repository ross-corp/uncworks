## Context

Coverage numbers from `go test -cover`: server 21.9%, sidecar 10.2%, temporal 9.1%. The untested code paths include JSONL parsing (dedup bugs cause duplicate entries in the UI), file listing (hidden dir filtering), sidecar exec (wrong workdir causes agent failures), loop detection (infinite loops burn tokens), and the full plan/verify pipeline (spec scaffolding, gate logic).

## Goals / Non-Goals

**Goals:**
- Test the 6 most critical code paths listed in the proposal
- Each test is self-contained with no external dependencies (no k8s, no temporal server)
- Tests use table-driven patterns for readability

**Non-Goals:**
- 100% coverage (targeting the highest-value paths only)
- Integration tests requiring a running cluster
- Mocking the LLM (temporal tests mock at the activity level)

## Decisions

### Decision 1: Test ExecCommand with exact workdir, not resolveWorkDir

The sidecar `ExecCommand` test should verify the exact working directory passed to `exec.Command`, not the `resolveWorkDir` helper. The helper has its own logic, but bugs at the `ExecCommand` level (e.g., ignoring the resolved dir) are more dangerous.

### Decision 2: Test VerifyRun gate logic with mock spec content

The 5-gate logic in `VerifyRun` (format, scaffold, lint, test, verify) should be tested by providing mock spec YAML and asserting which gates pass/fail. No real LLM calls.

### Decision 3: Use testdata/ directories for JSONL fixtures

JSONL parsing tests should use fixture files in `testdata/` rather than inline strings, since the JSONL format is complex and multi-line.
