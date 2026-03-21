## Why

The trace/observability system has experienced multiple regression bugs across its lifecycle:

1. **Wrong token field names** -- sidecar wrote `input_tokens`/`output_tokens` but the frontend expected `gen_ai.usage.input_tokens`/`gen_ai.usage.output_tokens`, causing token counts to silently show as zero.
2. **Missing parentId linking** -- child spans were emitted without `parentId`, so the waterfall rendered a flat list instead of a nested hierarchy under stage parents.
3. **Stale span names** -- tool spans were named `manage.tool`/`implement.tool` instead of `manage.write`/`implement.bash`, making it impossible to distinguish tool types at a glance.
4. **Diff not fetching** -- spans with `hasDiff=true` showed a DIFF badge but clicking them returned empty because the `/diff` endpoint wasn't wired or the span ID lookup failed.

These bugs were caught manually and fixed individually. There are no automated regression tests to prevent recurrence. A single rename or field restructuring in pi's event format can silently break the entire trace pipeline without any test failing.

## What Changes

Add a comprehensive regression test suite across three layers:

- **Contract tests (Go):** Verify TraceSpan JSON serialization includes all required fields (`traceId`, `status`, `parentId`) and that token usage field names match the pi format (`gen_ai.usage.input_tokens`, not `input_tokens`).
- **Integration tests (Go):** Verify `extractToolFromEvent` handles the `toolcall_start` format correctly, `createGitCheckpoint` sets `hasDiff` on tool spans, and `appendTraceSpan` produces valid JSONL.
- **Playwright e2e tests:** Verify the trace waterfall renders collapsible stage hierarchy, clicking a span with a DIFF badge shows diff content in the detail panel, and token usage appears in span detail when present.

## Capabilities

### New Capabilities
- `trace-regression-tests`: A multi-layer regression test suite that catches serialization mismatches, event parsing regressions, checkpoint/diff integration failures, and frontend rendering bugs in the trace system.

### Modified Capabilities
- None

## Impact

- **Test files** (`test/contract/`): New contract tests for TraceSpan field names and token usage metadata.
- **Test files** (`internal/sidecar/`): New integration tests for trace span writing and tool event extraction edge cases.
- **Test files** (`web/e2e/` or `e2e/`): New Playwright tests for trace timeline interaction, diff panel, and token display.
- **No production code changes.** This proposal is test-only.
