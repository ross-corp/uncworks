## ADDED Requirements

### Requirement: Real-time log streaming from agent to web UI
The system SHALL stream agent stdout/stderr output in real-time from the sidecar through the control plane to web UI subscribers via the WatchAgentRun streaming RPC.

#### Scenario: Log events arrive during agent execution
- **WHEN** an agent run is in Running phase and a client subscribes to WatchAgentRun
- **THEN** the client receives `AGENT_RUN_EVENT_TYPE_LOG` events containing stdout/stderr lines from pi-coding-agent
- **AND** events include ANSI escape codes (colors, formatting) unmodified

#### Scenario: Logs render with terminal formatting
- **WHEN** log events are rendered in the web UI Logs tab
- **THEN** xterm.js displays the output with full ANSI color and formatting support
- **AND** the display auto-scrolls to show the latest output

#### Scenario: Log streaming stops when agent completes
- **WHEN** the agent process exits (Succeeded, Failed, or Cancelled)
- **THEN** a final `AGENT_RUN_EVENT_TYPE_COMPLETED` event is sent
- **AND** the log viewer shows "Agent completed" at the end of the stream

### Requirement: Log persistence after pod deletion
The system SHALL persist agent log output to the AgentRun CRD status before pod cleanup, making logs available permanently.

#### Scenario: Logs available after pod deletion
- **WHEN** a completed run's pod has been deleted (retention expired)
- **AND** the user opens the Logs tab in the detail panel
- **THEN** the persisted log output is displayed in the xterm.js viewer

#### Scenario: Log truncation for large output
- **WHEN** agent output exceeds 1MB
- **THEN** only the last 1MB of output is persisted
- **AND** a "[log truncated]" prefix is shown in the viewer
