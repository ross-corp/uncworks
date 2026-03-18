## Context

The first live spec-driven run proved the pipeline architecture works (Plan → Execute transition succeeded) but exposed that free cloud models are too slow/rate-limited, and all stages share the same hardcoded config. The sidecar ExecCommand RPC, cleanup retry cap, and poll timeout fixes are already implemented and deployed.

## Goals / Non-Goals

**Goals:**
- Per-stage configuration (model, timeout, retries, onFailure) for spec-driven runs
- Sensible defaults that work out of the box with the current model setup
- First successful end-to-end spec-driven run in aot-local
- Web UI shows stage config and allows basic customization

**Non-Goals:**
- Per-task config within a stage (all tasks in execute share the same model)
- Dynamic model selection based on task complexity (future AI feature)
- Streaming stage output in real-time
- Knowledge system integration

## Decisions

### Decision 1: PipelineConfig as a nested struct on AgentRunSpec

```go
type PipelineConfig struct {
    Plan    StageConfig `json:"plan,omitempty"`
    Execute StageConfig `json:"execute,omitempty"`
    Verify  StageConfig `json:"verify,omitempty"`
}

type StageConfig struct {
    Model          string `json:"model,omitempty"`          // LiteLLM model name
    TimeoutSeconds int32  `json:"timeoutSeconds,omitempty"` // stage timeout
    MaxRetries     int32  `json:"maxRetries,omitempty"`     // max retries for this stage
    OnFailure      string `json:"onFailure,omitempty"`      // "retry" | "fail" | "skip"
}
```

**Defaults** (applied when fields are zero/empty):
```
Plan:    model=default-cloud, timeout=300s (5min), retries=2, onFailure=fail
Execute: model=default-cloud, timeout=900s (15min), retries=3, onFailure=retry
Verify:  model=default-cloud, timeout=180s (3min), retries=1, onFailure=fail
```

**Rationale:** Nested struct keeps config close to where it's used. Zero values mean "use defaults" — you only configure what you need to override. The `onFailure` field controls what happens when a stage exhausts its retries: `retry` feeds failure context to the next attempt, `fail` marks the run as Failed, `skip` proceeds to the next stage (useful for verify in dev mode).

### Decision 2: Config flows through WorkflowInput to activities

```
CreateAgentRun API
  → specProtoToCRD maps PipelineConfig to CRD
  → Controller passes it to Temporal WorkflowInput
  → runSpecDrivenPipeline reads per-stage config
  → PlanRun/StartAgent/VerifyRun use stage-specific model + timeout
  → Sidecar uses PI_MODEL env var override per agent invocation
```

The sidecar's `StartAgent` already supports a `PI_MODEL` env var. We pass the stage's model config as an env var in the `StartAgentRequest.env_vars` field — no proto change needed for the sidecar.

### Decision 3: Longer defaults for spec-driven mode

The current run TTL (from the user/UI) is separate from per-stage timeouts. The workflow TTL is the outer boundary; stage timeouts are inner boundaries. If a stage timeout fires, that stage fails — the workflow may retry or fail depending on onFailure. If the workflow TTL fires, everything stops.

For spec-driven runs, the default TTL should be longer (15-20 min) since the pipeline has 3 stages each taking minutes.

### Decision 4: ExecCommand RPC (already implemented)

Already done in previous session. Lightweight bash exec replaces agent-spawning for CLI commands.

## Risks / Trade-offs

- **Config complexity** — Users must understand stage config to tune performance. Mitigated by sensible defaults that work without any config.
- **Model cost** — Separate models per stage could increase cost. Mitigated by defaulting to the same model; users opt into premium models explicitly.
- **TTL interaction** — Workflow TTL and stage timeouts can conflict. Stage timeouts should always be shorter than the remaining workflow TTL.
