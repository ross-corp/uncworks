## Architecture

### Test Layers

The regression suite spans three complementary layers, each catching different failure modes:

```
Layer            Language   Runs In       Catches
─────────────────────────────────────────────────────────────────────
Contract tests   Go         CI (no k8s)   JSON field name mismatches,
                                          serialization regressions,
                                          type boundary violations

Integration      Go         CI (no k8s)   Event parsing errors,
tests                                     git checkpoint logic,
                                          span JSONL file I/O

Playwright e2e   TS         CI + browser  UI rendering regressions,
                                          API integration failures,
                                          click interaction bugs
```

### Contract Tests (Go)

Located in `test/contract/`. These verify the boundary between backend Go types and the frontend TypeScript interfaces. They require no running cluster.

**TraceSpan field names:**
```go
// Verify JSON keys match frontend expectations
span := server.TraceSpan{ID: "s1", TraceID: "t1", Status: "ok", ...}
data, _ := json.Marshal(span)
// Assert: contains "traceId", "status", "parentId"
// Assert: does NOT contain "trace_id", "parent_id", "start_time"
```

**Token usage field names:**
```go
// Verify metadata keys match OTel GenAI semantic conventions
metadata := map[string]interface{}{
    "gen_ai.usage.input_tokens":  1200,
    "gen_ai.usage.output_tokens": 340,
}
span := server.TraceSpan{Metadata: metadata}
data, _ := json.Marshal(span)
// Assert: contains "gen_ai.usage.input_tokens"
// Assert: does NOT contain "input_tokens" as a top-level metadata key
```

**Valid span types:**
```go
// Exhaustive check that all span types used in production are recognized
validTypes := []string{"llm", "tool", "thought", "input", "delegate", "lifecycle", "stage"}
```

### Integration Tests (Go)

Located in `internal/sidecar/`. These test internal functions with real data structures but no network or cluster dependencies.

**extractToolFromEvent:**
```go
// Test the toolcall_start format that pi uses
partial := `{"content":[{"type":"toolCall","name":"bash","arguments":{"command":"ls"}}]}`
ame := &piAssistantEvent{Partial: json.RawMessage(partial)}
name, inputJSON := extractToolFromEvent(ame)
// Assert: name == "bash", inputJSON contains "command"
```

**createGitCheckpoint:**
```go
// Test in a temp git repo that checkpoint produces hasDiff data
dir := initTestRepo(t)
os.WriteFile(filepath.Join(dir, "new.txt"), []byte("content"), 0644)
sha, diff := createGitCheckpoint(dir, "write")
// Assert: sha != "", diff != nil, diff.Files[0].Path contains "new.txt"
```

**appendTraceSpan + readSpansFile round-trip:**
```go
// Write spans via appendTraceSpan, read them back via server.readSpansFile
// Assert: all fields survive the JSON round-trip
```

### Playwright E2E Tests (TypeScript)

Located in `web/e2e/` (or `e2e/playwright/`). These run against a live dev server with fixture data or a real cluster.

**Trace hierarchy rendering:**
```typescript
// Navigate to a run detail page with known trace data
// Assert: stage rows (PLAN, EXECUTE, VERIFY) are visible
// Assert: child spans are nested under their stage parent
```

**Collapse/expand interaction:**
```typescript
// Click the collapse toggle on a stage row
// Assert: child spans disappear from the DOM
// Click the expand toggle
// Assert: child spans reappear
```

**Diff badge and panel:**
```typescript
// Find a span row with a "DIFF" badge
// Click the span row
// Assert: the detail panel opens on the right
// Assert: the detail panel contains file paths and diff content
```

**Token usage display:**
```typescript
// Click a thought span with token metadata
// Assert: the detail panel shows "Input Tokens" and "Output Tokens"
// Assert: the displayed values are non-zero numbers
```

### Fixture Data

For Playwright tests that don't require a live cluster, use fixture `spans.jsonl` files with known data:

```jsonl
{"id":"stage-1","traceId":"t1","name":"PLAN","type":"stage","startTime":"...","endTime":"...","status":"ok","hasDiff":false}
{"id":"child-1","traceId":"t1","parentId":"stage-1","name":"manage.thought","type":"thought","startTime":"...","endTime":"...","metadata":{"gen_ai.usage.input_tokens":1200,"gen_ai.usage.output_tokens":340},"hasDiff":false}
{"id":"child-2","traceId":"t1","parentId":"stage-1","name":"manage.write","type":"tool","startTime":"...","endTime":"...","metadata":{"toolInput":"{\"path\":\"/workspace/spec.md\"}"},"hasDiff":true}
```

### Test Data Isolation

- Contract tests use in-memory Go structs only.
- Integration tests use `t.TempDir()` for git repos and JSONL files.
- Playwright tests either use mocked API responses or a dedicated test run namespace.
