## Context

UNCWORKS runs AI agents inside Kubernetes pods built from `docker/Dockerfile.agent-base`. The sidecar binary monitors agent stdout for pi streaming events (JSONL) and creates trace spans for tool executions, LLM turns, and thinking blocks. The web dashboard renders these spans in a waterfall timeline (`web/src/components/TraceTimeline.tsx`). Today there is no visibility into context management — engineers cannot see whether the agent's conversation context is growing unbounded or being intelligently managed.

Pi-DCP is a pi extension that hooks into the `context` event (fired before every LLM call) and applies a three-phase pruning workflow: Prepare, Process, Filter. It ships four built-in rules — deduplication, superseded writes, error purging, and recency protection — and emits structured log lines showing what was pruned.

## Goals / Non-Goals

**Goals:**
- Pre-install pi-dcp in the agent base image so all agent pods get context pruning by default
- Detect DCP pruning events in the sidecar stdout parser and create `dcp` trace spans
- Render DCP spans in the trace timeline with distinct styling
- Show pruning statistics (messages pruned, kept, ratio) in the span detail panel
- Allow disabling DCP via agent startup flags

**Non-Goals:**
- Custom DCP rule authoring via the UNCWORKS UI (future work)
- Persisting DCP configuration per-workspace or per-project
- Modifying pi-dcp source code — we use it as-is from upstream
- Real-time DCP stats in the run overview (only in trace detail view)

## Decisions

### 1. Install via git clone in Dockerfile

Pi-dcp installs by cloning into `~/.pi/agent/extensions/pi-dcp`. We add this to `docker/Dockerfile.sidecar` (where pi runs, not agent-base):

```dockerfile
# Install pi-dcp extension for dynamic context pruning
RUN git clone --depth 1 https://github.com/zenobi-us/pi-dcp.git /root/.pi/agent/extensions/pi-dcp \
    && cd /root/.pi/agent/extensions/pi-dcp && npm install
```

This runs at image build time, so no per-pod clone overhead. We pin to a specific commit SHA in production via `git clone --branch <tag>` once pi-dcp publishes releases.

*Alternative*: npm install as a global package. Rejected because pi extensions must live in the extensions directory, not node_modules.

### 2. Detect DCP events via log line pattern matching in gateway.go

Pi-dcp writes pruning events to stdout in a predictable format:

```
[pi-dcp] Pruned 12 / 45 messages
```

And with debug mode:

```
[pi-dcp] Dedup: marking duplicate message at index 15 (hash: k2l9x)
[pi-dcp] SupersededWrites: marking superseded write at index 23: src/index.ts
[pi-dcp] Filter phase complete: 12 pruned, 33 kept (45 total)
```

The sidecar already has a fallback path for plain-text prefix matching after the JSONL parser. We add a new prefix check for `[pi-dcp]` lines in `maybeCaptureStreamEvent` (or in the existing plain-text fallback block). When a pruning summary line is matched, we extract counts via regex and create a span.

The detection logic in `internal/sidecar/gateway.go`:

```go
// dcpPruneRe matches: [pi-dcp] Pruned 12 / 45 messages
var dcpPruneRe = regexp.MustCompile(`^\[pi-dcp\] Pruned (\d+) / (\d+) messages$`)

// dcpRuleRe matches debug-mode per-rule lines
var dcpRuleRe = regexp.MustCompile(`^\[pi-dcp\] (\w+): .*`)

// dcpFilterRe matches: [pi-dcp] Filter phase complete: 12 pruned, 33 kept (45 total)
var dcpFilterRe = regexp.MustCompile(`^\[pi-dcp\] Filter phase complete: (\d+) pruned, (\d+) kept \((\d+) total\)`)
```

When `dcpPruneRe` matches, we create a DCP span:

```go
span := TraceSpan{
    ID:        uuid.New().String(),
    TraceID:   getTraceID(),
    ParentID:  getParentSpanID(),
    Name:      spanPrefix() + ".dcp",
    Type:      "dcp",
    StartTime: now,
    EndTime:   now, // instant span — pruning is near-zero latency
    Metadata: map[string]interface{}{
        "pruned": pruned,
        "total":  total,
        "kept":   total - pruned,
        "ratio":  float64(pruned) / float64(total),
        "stage":  currentStage,
        "role":   spanPrefix(),
    },
}
```

*Alternative*: Have pi-dcp emit structured JSONL events with a `"type": "dcp_prune"` field. This would be cleaner but requires forking pi-dcp. We may propose this upstream later.

### 3. DCP spans are instant (zero-duration) spans

Context pruning runs in <1ms — it is a synchronous filter before the LLM call. We model DCP spans with `StartTime == EndTime`, the same pattern used for other near-instant events. In the waterfall, these render as thin vertical bars (the same treatment already applied to 0ms spans).

### 4. DCP span styling in TraceTimeline

Add a `dcp` entry to the `OP_COLORS` map in `TraceTimeline.tsx`:

```typescript
const OP_COLORS: Record<string, { bar: string; text: string }> = {
  thought: { bar: "bg-blue-500/30 border-l-2 border-blue-500",     text: "text-blue-400" },
  bash:    { bar: "bg-emerald-500/30 border-l-2 border-emerald-500", text: "text-emerald-400" },
  write:   { bar: "bg-violet-500/30 border-l-2 border-violet-500", text: "text-violet-400" },
  read:    { bar: "bg-slate-400/20 border-l-2 border-slate-400",   text: "text-slate-400" },
  started: { bar: "bg-amber-500/30 border-l-2 border-amber-500",   text: "text-amber-400" },
  dcp:     { bar: "bg-cyan-500/30 border-l-2 border-cyan-500",     text: "text-cyan-400" },
};
```

Cyan is chosen because it is not used by any existing span type and is visually associated with "data/info" operations.

The span label shows inline stats: `DCP: -12/45 msgs` (pruned count / total, mimicking the diff stats pattern used for file changes).

### 5. Detail panel for DCP spans

When a DCP span is selected, the detail panel shows:

- **Pruned**: 12 messages removed
- **Kept**: 33 messages retained
- **Total**: 45 messages before pruning
- **Ratio**: 26.7% pruned
- **Rules** (if debug data available): per-rule breakdown table

This reuses the existing span detail panel infrastructure — no new component needed, just conditional rendering based on `span.type === "dcp"`.

### 6. Accumulating debug-mode rule data

When DCP debug mode is active, per-rule log lines appear before the summary. We track these in a temporary buffer:

```go
var (
    dcpDebugRulesMu sync.Mutex
    dcpDebugRules   = map[string]int{} // rule name -> prune count
)
```

Each `[pi-dcp] RuleName: marking ...` line increments the counter for that rule. When the pruning summary line arrives, we attach the accumulated `dcpDebugRules` map to the span metadata and reset the buffer.

## Risks / Trade-offs

- **Log format coupling**: We parse pi-dcp's stdout format, which has no stability guarantee. If the format changes, span detection breaks silently. Mitigated by pinning the pi-dcp version in the Dockerfile and adding integration tests.
- **Debug mode overhead**: When debug logging is enabled, pi-dcp emits many more lines per LLM call. The sidecar regex matching is O(1) per line, so this is negligible.
- **Instant spans in the waterfall**: Zero-duration spans can be hard to spot in a busy timeline. Mitigated by the distinct cyan color and by showing DCP stats inline on the span label.
- **Extension directory structure**: Pi-dcp assumes `~/.pi/agent/extensions/` exists. The base image runs as root, so the path is `/root/.pi/agent/extensions/`. If the agent runtime changes the user, the extension path must be updated.

## Open Questions

- Should we propose structured JSONL output (`{"type": "dcp_prune", ...}`) as an upstream PR to pi-dcp? This would make detection more robust.
- Should DCP stats be aggregated at the stage level (total pruned per stage) in addition to per-event spans?
- Should the `/dcp-stats` command output also be captured as a span (showing session-level cumulative stats)?
