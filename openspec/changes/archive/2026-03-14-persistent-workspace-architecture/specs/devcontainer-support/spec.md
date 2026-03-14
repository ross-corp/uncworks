## ADDED Requirements

### Requirement: devcontainer.json generation
The hydrator SHALL generate a `.devcontainer/devcontainer.json` file in the workspace during setup.

#### Scenario: devcontainer.json created
- **WHEN** hydration completes
- **THEN** `/workspace/.devcontainer/devcontainer.json` exists
- **AND** it references the agent base image, workspace folder, and devbox postStartCommand

### Requirement: VS Code Remote attachment to running pods
Users SHALL be able to attach VS Code to a running agent pod to pair-program alongside the agent.

#### Scenario: Attach VS Code to live run
- **WHEN** a run is in Running phase
- **THEN** the API provides connection info at `GET /api/v1/runs/{id}/connect`
- **AND** the user can attach VS Code via Remote-SSH or kubectl port-forward

### Requirement: VS Code attachment to debug pods
Users SHALL be able to attach VS Code to debug pods for post-completion inspection.

#### Scenario: Attach VS Code to debug session
- **WHEN** a debug pod is running (Deployment replicas=1, mode=debug)
- **THEN** VS Code can attach via the same mechanism as live runs

### Requirement: Trace timeline with diff view
The web UI SHALL display a distributed-trace-style timeline of agent execution with clickable spans that show file diffs.

#### Scenario: Timeline renders agent activity
- **WHEN** the user opens the Traces tab for a run
- **THEN** a timeline shows spans: LLM thoughts, tool calls, file edits, human input events
- **AND** spans are ordered chronologically with duration bars

#### Scenario: Click span shows diff
- **WHEN** the user clicks a tool call span that modified files
- **THEN** a diff view shows the before/after changes for each file modified by that tool call

#### Scenario: Timeline for completed runs
- **WHEN** the user opens the Traces tab for a completed run
- **THEN** the full timeline is available from persisted trace data (PostgreSQL or PVC)

### Requirement: Trace data collection
The sidecar SHALL collect trace spans from the agent process and store them for timeline rendering.

#### Scenario: Tool call spans captured
- **WHEN** the agent invokes a tool (file edit, shell command, etc.)
- **THEN** a span is recorded with: tool name, arguments, duration, start/end time, and associated file diffs

#### Scenario: LLM response spans captured
- **WHEN** the agent receives an LLM response
- **THEN** a span is recorded with: model, prompt summary, response summary, token count, duration
