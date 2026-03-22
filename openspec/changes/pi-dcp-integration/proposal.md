## Why

Agent runs accumulate large conversation contexts as they work through multi-step tasks — duplicate tool outputs, superseded file writes, resolved errors, and stale messages all consume tokens without adding value. This increases cost, slows inference, and can push agents past context windows. Pi-DCP (Dynamic Context Pruning) is a pi extension that intelligently prunes conversation context before each LLM call, but today UNCWORKS has no visibility into when pruning happens, what was removed, or how it affects the run. We need to install pi-dcp in the agent base image, detect its pruning events in the sidecar, and surface them as spans in the trace timeline so engineers can understand context management during a run.

## What Changes

- Install pi-dcp in the agent base image (`docker/Dockerfile.agent-base`) so every agent pod has context pruning available out of the box
- Configure pi-dcp in the sidecar gateway to detect DCP log events (`[pi-dcp] Pruned N / M messages`) in the JSONL output stream
- Create a new `dcp` span type in the trace system that captures pruning metadata: messages pruned, messages kept, total messages, rules applied
- Add DCP-specific styling to the TraceTimeline waterfall — a distinct color (cyan/teal) so context pruning events are visually distinguishable from tool calls, LLM turns, and thoughts
- Show per-rule breakdown in the span detail panel when a DCP span is selected

## Capabilities

### New Capabilities
- `dcp-agent-install`: Pi-DCP extension pre-installed in agent base image, activated automatically during agent runs
- `dcp-trace-events`: Sidecar captures DCP pruning events as trace spans with pruning metadata (counts, rules, ratios)
- `dcp-timeline-display`: TraceTimeline renders DCP spans with distinct styling and inline pruning stats

### Modified Capabilities
- None

## Impact

- **Modified**: `docker/Dockerfile.agent-base` — clone pi-dcp into extensions directory
- **Modified**: `internal/sidecar/gateway.go` — new event type detection in `maybeCaptureStreamEvent`, new DCP span creation
- **Modified**: `web/src/components/TraceTimeline.tsx` — new `dcp` entry in `OP_COLORS`, DCP metadata rendering in detail panel
- **New**: `web/src/components/DcpSpanDetail.tsx` — detail panel component for DCP spans showing per-rule stats
- **Dependencies**: pi-dcp (MIT, https://github.com/zenobi-us/pi-dcp)
- **No changes** to the Go API server, controller, temporal workflows, or protobuf schemas — DCP spans use the existing TraceSpan structure
