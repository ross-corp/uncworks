## ADDED Requirements

### Requirement: Pi-DCP is pre-installed in the agent base image
The agent base image SHALL include pi-dcp cloned into the pi extensions directory so that context pruning is available without per-run setup.

#### Scenario: Agent pod starts with DCP available
- **WHEN** an agent pod is created from the base image
- **THEN** the directory `~/.pi/agent/extensions/pi-dcp/` exists and contains the pi-dcp extension source files

#### Scenario: DCP activates automatically during agent runs
- **WHEN** the pi agent starts inside the pod
- **THEN** pi-dcp initializes and logs `[pi-dcp] Initialized with 4 rules: deduplication, superseded-writes, error-purging, recency`

### Requirement: Sidecar detects DCP pruning events in JSONL output
The sidecar gateway SHALL detect pi-dcp log lines in the agent stdout stream and create trace spans for each pruning event.

#### Scenario: DCP prunes messages during an LLM call
- **WHEN** the sidecar reads a line matching `[pi-dcp] Pruned (\d+) / (\d+) messages` from agent stdout
- **THEN** a trace span of type `dcp` is created with metadata `pruned`, `total`, and `kept` fields

#### Scenario: DCP logs appear interleaved with pi events
- **WHEN** DCP log lines appear between `message_start` and `tool_execution_start` events
- **THEN** the DCP span is created with the correct parent span (the active LLM span) and does not interfere with other span tracking

### Requirement: DCP spans include pruning metadata
Each DCP trace span SHALL capture structured pruning statistics in its metadata.

#### Scenario: Pruning stats are captured
- **WHEN** a DCP pruning event is detected
- **THEN** the span metadata contains:
  - `pruned` (integer): number of messages removed
  - `total` (integer): total messages before pruning
  - `kept` (integer): total messages after pruning
  - `ratio` (float): percentage of messages pruned (pruned / total)

#### Scenario: Debug-mode rule breakdown is captured
- **WHEN** DCP debug mode is enabled and per-rule lines like `[pi-dcp] Dedup: marking duplicate message at index N` appear before a pruning summary
- **THEN** the span metadata includes a `rules` map with per-rule prune counts (e.g., `{"deduplication": 3, "superseded-writes": 2, "error-purging": 1}`)

### Requirement: DCP spans appear in the trace timeline
The TraceTimeline component SHALL render DCP spans with a distinct visual style so they are immediately recognizable.

#### Scenario: DCP span renders with distinct color
- **WHEN** a trace includes spans of type `dcp`
- **THEN** the span bar uses cyan/teal coloring (e.g., `bg-cyan-500/30 border-l-2 border-cyan-500`) distinct from tool (emerald), thought (blue), and write (violet) spans

#### Scenario: DCP span shows inline pruning stats
- **WHEN** a DCP span is rendered in the waterfall
- **THEN** the span label displays the pruning summary inline (e.g., `DCP: -12/45 msgs`)

### Requirement: DCP span detail panel shows pruning breakdown
The span detail panel SHALL display DCP-specific information when a DCP span is selected.

#### Scenario: Selecting a DCP span shows pruning details
- **WHEN** a user clicks on a DCP span in the timeline
- **THEN** the detail panel shows:
  - Messages pruned / total / kept
  - Pruning ratio as a percentage
  - Per-rule breakdown (if available from debug mode)
  - The raw log line that triggered the span

### Requirement: DCP can be disabled via configuration
Agents SHALL be able to disable pi-dcp via startup flags without modifying the base image.

#### Scenario: DCP disabled at startup
- **WHEN** the agent is started with `--dcp-enabled=false`
- **THEN** no DCP pruning occurs and no DCP spans appear in the trace

#### Scenario: DCP enabled by default
- **WHEN** the agent starts with no DCP-related flags
- **THEN** DCP is enabled and actively prunes context before each LLM call
