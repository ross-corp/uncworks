## ADDED Requirements

### Requirement: Logs teed to PVC
The sidecar SHALL tee agent stdout/stderr to `/workspace/.aot/logs/agent.log` on the PVC in addition to capturing via pipe.

#### Scenario: Log file written during execution
- **WHEN** the agent produces output
- **THEN** each line appears in both the StreamOutput RPC and the log file on disk

#### Scenario: Log file readable after Pod deletion
- **WHEN** the Pod is scaled to 0 and a client requests logs
- **THEN** the API server reads `/workspace/.aot/logs/agent.log` from the PVC host path

### Requirement: File/log API seamless across Pod states
The file and log API endpoints SHALL serve content regardless of whether the Pod is running or scaled to 0.

#### Scenario: Files served while Pod running
- **WHEN** a file request is made and the Deployment has replicas=1
- **THEN** the API uses K8s exec to read from the Pod (fast, live data)

#### Scenario: Files served after Pod scaled down
- **WHEN** a file request is made and the Deployment has replicas=0
- **THEN** the API reads directly from the PVC host path on disk

#### Scenario: Seamless transition
- **WHEN** a run transitions from Running to Completed
- **THEN** the file/log API endpoints continue to return data without interruption
- **AND** the UI does not show any "unavailable" state

### Requirement: Trace and diff persistence
The system SHALL store agent execution traces (tool calls, LLM responses, file changes) on the PVC and in PostgreSQL for timeline rendering.

#### Scenario: Traces written during execution
- **WHEN** the agent makes tool calls or receives LLM responses
- **THEN** trace spans are recorded to `/workspace/.aot/traces/` on the PVC
- **AND** spans are persisted to PostgreSQL for query

#### Scenario: Diffs computed per tool call
- **WHEN** the agent modifies files via a tool call
- **THEN** a git diff is captured and associated with the trace span
- **AND** the diff is stored alongside the span for retrieval
