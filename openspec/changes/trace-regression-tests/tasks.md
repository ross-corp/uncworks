## 1. Contract Tests — TraceSpan JSON Fields

- [ ] 1.1 Contract test: TraceSpan JSON includes `traceId`, `status`, and `parentId` fields when set, and omits them when empty (extends existing `boundary_rest_types_test.go`)
- [ ] 1.2 Contract test: token usage metadata field names match OTel GenAI convention (`gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`), rejecting bare `input_tokens`/`output_tokens`
- [ ] 1.3 Contract test: all production span types (`llm`, `tool`, `thought`, `input`, `delegate`, `lifecycle`, `stage`) are present in a `validSpanTypes` allowlist, and no unknown types pass validation

## 2. Integration Tests — Event Parsing

- [ ] 2.1 Integration test: `extractToolFromEvent` correctly handles pi's `toolcall_start` format (`partial.content[].type=="toolCall"`) and returns both tool name and JSON arguments
- [ ] 2.2 Integration test: `extractToolFromEvent` returns empty strings for events with no tool data (text-only assistant messages, malformed partial JSON)
- [ ] 2.3 Integration test: span names follow `{role}.{toolName}` convention — verify that for a `bash` tool call during EXECUTE stage, the span name is `implement.bash` (not `implement.tool`)

## 3. Integration Tests — Git Checkpoint & Diff

- [ ] 3.1 Integration test: `createGitCheckpoint` returns non-empty SHA and non-nil `SpanDiff` with correct file paths when workspace has uncommitted changes
- [ ] 3.2 Integration test: `createGitCheckpoint` returns empty SHA and nil diff when workspace has no changes (idempotency check)
- [ ] 3.3 Integration test: consecutive `createGitCheckpoint` calls produce incremental diffs (second diff contains only file2, not file1 from first checkpoint)

## 4. Integration Tests — Span JSONL Round-Trip

- [ ] 4.1 Integration test: `appendTraceSpan` writes valid JSONL to the spans file, and `readSpansFile` (server package) deserializes all fields correctly including `traceId`, `status`, `parentId`, `hasDiff`, and nested `diff.files`

## 5. Playwright E2E Tests — Trace Hierarchy & Interaction

- [ ] 5.1 Playwright test: trace timeline renders stage parent rows (PLAN, EXECUTE, VERIFY) with `type="stage"` styling (bold, amber color, taller bar)
- [ ] 5.2 Playwright test: clicking the collapse toggle on a stage row hides all child spans; clicking expand restores them
- [ ] 5.3 Playwright test: clicking a span row with a DIFF badge opens the detail panel and displays file paths with diff content (green/red lines)
- [ ] 5.4 Playwright test: clicking a `*.thought` span shows token usage (`Input Tokens`, `Output Tokens`) with non-zero values in the detail panel
