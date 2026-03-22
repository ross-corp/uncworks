# ConnectRPC API Reference

The UNCWORKS API is defined as a ConnectRPC (gRPC-compatible) service in `proto/aot/api/v1/api.proto`. The API server listens on port 50055 by default.

## Service: AOTService

### CreateAgentRun

Creates a new agent run.

```
rpc CreateAgentRun(CreateAgentRunRequest) returns (CreateAgentRunResponse)
```

**Request:**
```
CreateAgentRunRequest {
  spec: AgentRunSpec {
    backend: Backend          // POD (required)
    repos: [Repository]       // At least one repository (required)
    prompt: string            // Task description (required)
    model_tier: string        // LiteLLM model name (default: "default")
    orchestration_mode: enum  // SINGLE, AUTO, MANUAL, SPEC_DRIVEN
    ttl_seconds: int32        // Max lifetime (default: 3600)
    env_vars: map<string,string>
    spec_content: string      // OpenSpec content for spec-driven mode
    pipeline_config: PipelineConfig  // Per-stage config for spec-driven
  }
}
```

**Response:** `CreateAgentRunResponse { agent_run: AgentRun }`

### GetAgentRun

Retrieves current state of a run by ID.

```
rpc GetAgentRun(GetAgentRunRequest { id: string }) returns (AgentRun)
```

### ListAgentRuns

Lists runs with optional filters.

```
rpc ListAgentRuns(ListAgentRunsRequest) returns (ListAgentRunsResponse)
```

**Filters:** `phase_filter`, `spec_run_id`, `parent_run_id`, `stage_filter`, `limit`, `cursor`

**Response:** `ListAgentRunsResponse { agent_runs: [AgentRun], next_cursor: string }`

### WatchAgentRun

Streams real-time events for a run.

```
rpc WatchAgentRun(WatchAgentRunRequest { id: string }) returns (stream AgentRunEvent)
```

**Event types:** `PHASE_CHANGED`, `LOG`, `TOOL_CALL`, `WAITING_FOR_INPUT`, `COMPLETED`

### CancelAgentRun

Requests cancellation of a running agent.

```
rpc CancelAgentRun(CancelAgentRunRequest { id: string }) returns (CancelAgentRunResponse)
```

### SendHumanInput

Provides human-in-the-loop input to a paused agent.

```
rpc SendHumanInput(SendHumanInputRequest) returns (SendHumanInputResponse)
```

**Request:** `{ agent_run_id: string, input: string }`
**Response:** `{ accepted: bool }`

### GetRunGraph

Returns the parent/child tree for a spec execution.

```
rpc GetRunGraph(GetRunGraphRequest { id: string }) returns (RunGraph)
```

**Response:** `RunGraph { nodes: [RunGraphNode], edges: [RunGraphEdge] }`

### SearchPastWork

Searches the knowledge base for relevant past work.

```
rpc SearchPastWork(SearchPastWorkRequest) returns (SearchPastWorkResponse)
```

**Filters:** `query`, `repo_url`, `created_after`, `created_before`, `source_filter` (CODE, TRACE, ALL), `limit`

## Internal Service: AgentSidecarService

Defined in `proto/aot/agent/v1/agent.proto`. Used by the control plane to communicate with agent pods. Not intended for external clients.

RPCs: `StartAgent`, `StreamOutput`, `SendInput`, `GetStatus`, `StopAgent`, `ExecCommand`
