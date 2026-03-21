## Architecture

### Span Hierarchy

Following OTel conventions, every trace has a single root span. Pipeline stages are parent spans. Agent events are child spans:

```
pipeline (root)                    ████████████████████████████████████████████
├─ PLAN                            ████████████████
│  ├─ manage.thought               ████          tokens: 1200 in, 340 out
│  ├─ manage.bash                  █             openspec instructions...
│  ├─ manage.write                 █             proposal.md
│  ├─ manage.thought               ███           tokens: 800 in, 520 out
│  └─ manage.bash                  █             openspec validate
├─ EXECUTE (attempt 1)             ████████████████████
│  ├─ implement.thought            ████████      tokens: 2400 in, 890 out
│  ├─ implement.write              ██  DIFF      HELLO.md
│  ├─ implement.thought            ███           tokens: 1600 in, 200 out
│  └─ implement.bash               ████          npm test
├─ VERIFY (attempt 1)              ████████  FAILED
│  ├─ manage.thought               ████          tokens: 1800 in, 150 out
│  └─ manage.bash                  █             openspec list --json
├─ EXECUTE (attempt 2)             ██████████████████████████
│  ├─ implement.thought            ██████        tokens: 3200 in, 1200 out
│  ├─ implement.write              ██  DIFF      tasks.md [x]
│  └─ implement.bash               ████          npm test
└─ VERIFY (attempt 2)              ██████████  PASSED
   ├─ manage.thought               ████          tokens: 1800 in, 300 out
   └─ manage.bash                  █             openspec validate
```

### Span Data Model

Following OTel + GenAI semantic conventions:

```go
type TraceSpan struct {
    // Identity
    ID       string `json:"id"`
    TraceID  string `json:"traceId"`       // shared across all spans in a run
    ParentID string `json:"parentId"`      // links to parent span

    // Core
    Name      string    `json:"name"`      // e.g. "PLAN", "manage.write", "implement.bash"
    Type      string    `json:"type"`      // "stage", "llm", "tool", "input"
    StartTime time.Time `json:"startTime"`
    EndTime   time.Time `json:"endTime"`
    Status    string    `json:"status"`    // "ok", "error", "unset"

    // Metadata (OTel attributes)
    Metadata map[string]interface{} `json:"metadata"`

    // Diff
    HasDiff bool     `json:"hasDiff"`
    Diff    *SpanDiff `json:"diff,omitempty"`
}
```

### Metadata by Span Type

**Root span (pipeline):**
```json
{
  "pipeline.stages": 3,
  "pipeline.attempts": 2,
  "pipeline.result": "succeeded",
  "gen_ai.usage.input_tokens": 12800,
  "gen_ai.usage.output_tokens": 3600,
  "gen_ai.usage.total_tokens": 16400,
  "gen_ai.cost.total_usd": 0.0045,
  "gen_ai.request.model": "deepseek-v3.1",
  "tool.count.total": 18,
  "tool.count.success": 16,
  "tool.count.error": 2
}
```

**Stage parent span (PLAN, EXECUTE, VERIFY):**
```json
{
  "stage": "execute",
  "attempt": 1,
  "result": "failed",
  "gen_ai.usage.input_tokens": 4000,
  "gen_ai.usage.output_tokens": 1090,
  "gen_ai.cost.stage_usd": 0.0012,
  "tool.count": 4,
  "tool.count.error": 0,
  "task.completion": "17/22"
}
```

**LLM thought span (manage.thought, implement.thought):**
```json
{
  "role": "manage",
  "stage": "plan",
  "gen_ai.request.model": "deepseek-v3.1",
  "gen_ai.usage.input_tokens": 1200,
  "gen_ai.usage.output_tokens": 340,
  "gen_ai.usage.cache_read_tokens": 800,
  "gen_ai.context.window_size": 32768,
  "gen_ai.context.utilization_pct": 42,
  "durationMs": 1154
}
```

**Tool span (manage.write, implement.bash):**
```json
{
  "role": "implement",
  "stage": "execute",
  "tool": "write",
  "toolInput": "{\"path\":\"/workspace/neph.nvim/HELLO.md\",\"content\":\"Hello World\"}",
  "tool.exit_code": 0,
  "checkpointSHA": "abc1234",
  "prevCheckpointSHA": "def5678"
}
```

### Where Spans Are Created

```
Component           Creates                     When
──────────────────────────────────────────────────────────────
Temporal workflow   Root span (pipeline)         Pipeline starts
Temporal workflow   Stage parent (PLAN)          Before PlanRun activity
Temporal workflow   Stage parent (EXECUTE)       Before StartAgent activity
Temporal workflow   Stage parent (VERIFY)        Before VerifyRun activity
Temporal workflow   Close stage span             After activity completes
Sidecar gateway     manage.started               StartAgent RPC
Sidecar gateway     manage.thought               message_end event
Sidecar gateway     manage.write                 tool_execution_end event
Sidecar gateway     implement.*                  Same as above, execute stage
```

### Token Usage Collection

Pi emits token usage in `message_end` events:

```json
{
  "type": "message_end",
  "message": {
    "role": "assistant",
    "usage": {
      "input_tokens": 1200,
      "output_tokens": 340,
      "cache_creation_input_tokens": 0,
      "cache_read_input_tokens": 800
    }
  }
}
```

The sidecar extracts `usage` from `message_end` events and includes it in `*.thought` span metadata. Stage parent spans aggregate token counts from all child thought spans.

### Cost Estimation

Cost is computed from token counts using per-model pricing:

```go
var modelPricing = map[string]struct{ InputPerM, OutputPerM float64 }{
    "deepseek-v3.1":  {0.15, 0.75},
    "qwen3-coder":    {0.22, 1.00},
    "deepseek-v3.2":  {0.26, 0.38},
}

func estimateCost(model string, inputTokens, outputTokens int) float64 {
    p := modelPricing[model]
    return (float64(inputTokens) * p.InputPerM + float64(outputTokens) * p.OutputPerM) / 1_000_000
}
```

### Workflow Integration

The Temporal workflow writes stage spans to a trace metadata file that the sidecar merges into spans.jsonl:

```
workflow_spec_driven.go:
  1. Generate traceID = uuid
  2. Write root span (pipeline) to PVC via ExecCommand: echo '{}' >> .aot/traces/spans.jsonl
  3. Before PlanRun: write PLAN parent span (start time, no end time)
  4. After PlanRun: update PLAN span with end time + aggregated metadata
  5. Before Execute: write EXECUTE parent span
  6. After Execute: close EXECUTE span
  7. Before Verify: write VERIFY parent span
  8. After Verify: close VERIFY span
  9. On retry: new EXECUTE/VERIFY spans with attempt++
  10. On completion: close root span with aggregate stats
```

The sidecar sets `parentSpanId` on all child spans to link to the current stage span. The stage span ID is passed via `StartAgentRequest.parentSpanId`.

### Frontend Waterfall

The tree-building code (`buildFlatTree`) already handles `parentId`. With stage parents, the waterfall naturally nests:

```
┌─ Label (240px) ──────────────┬─ Waterfall ────────────────────────────┐
│                              │                                       │
│ ▾ pipeline    5m 23s   $0.02 │ ████████████████████████████████████   │
│   ▾ PLAN      1m 12s         │ ████████████                          │
│     manage.thought    1.2s   │   ████                                │
│     manage.bash       10ms   │   █                                   │
│     manage.write      5ms    │   █                                   │
│   ▾ EXECUTE   2m 45s         │              ██████████████████████    │
│     implement.thought 2.1s   │              ████████                 │
│     implement.write   12ms   │              ██  DIFF                 │
│   ▾ VERIFY    1m 26s  PASS   │                            ██████████ │
│     manage.thought    0.8s   │                            ████       │
│                              │                                       │
└──────────────────────────────┴───────────────────────────────────────┘
```

**Collapsible rows:** Click the triangle on a stage parent to collapse/expand its children. Collapsed rows show only the parent bar with aggregate stats.

**Detail panel for stage parents:**
```
┌──────────────────────────────────────────┐
│ EXECUTE (attempt 1)                      │
│ ─────────────────────────────────────    │
│ Duration     2m 45s                      │
│ Status       PASSED                      │
│ Attempt      1 of 2                      │
│                                          │
│ Tokens                                   │
│   Input      4,000                       │
│   Output     1,090                       │
│   Cache      2,400 (60% hit)             │
│   Context    42% of 32K window           │
│                                          │
│ Cost         $0.0012 USD                 │
│ Model        deepseek-v3.1              │
│                                          │
│ Tools                                    │
│   Total      4                           │
│   Success    4                           │
│   Errors     0                           │
│                                          │
│ Tasks        17/22 complete              │
└──────────────────────────────────────────┘
```
