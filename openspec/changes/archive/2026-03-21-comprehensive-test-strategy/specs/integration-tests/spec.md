## ADDED Requirements

### Requirement: Sidecar span capture matches structured log entries
The system SHALL have an integration test that verifies the number of tool spans created by the sidecar matches the number of tool_call entries in the structured log parser for the same JSONL input.

#### Scenario: Span count equals tool call count
- **WHEN** a JSONL file with N tool_execution_start/end events is processed by both maybeCaptureStreamEvent and parseAgentJSONL
- **THEN** the number of "tool" type spans equals the number of "tool_call" type log entries

### Requirement: Hydrator workspace layout verification
The system SHALL have an integration test that verifies the workspace directory structure after hydration for both single-repo and multi-repo configurations.

#### Scenario: Single repo layout
- **WHEN** the hydrator runs with one repo (url: "https://github.com/org/repo")
- **THEN** `/workspace/repo/` contains a `.git` directory and the repo's files

#### Scenario: Multi repo layout
- **WHEN** the hydrator runs with two repos
- **THEN** each repo is at `/workspace/<reponame>/` with its own `.git` directory

### Requirement: JSONL parser consistency
The system SHALL have an integration test that verifies parseAgentJSONL and parseThinkingFromLines produce consistent results from the same JSONL input.

#### Scenario: Parsers agree on agent activity
- **WHEN** the same agent.jsonl file is parsed by both the structured log parser and the thinking parser
- **THEN** the thinking parser does not report "thinking=true" for any message that the structured log parser already shows as completed

### Requirement: Loop detection kills agent after N identical tool calls
The system SHALL have an integration test that verifies the sidecar's loop detection terminates an agent process when it makes N consecutive identical tool calls.

#### Scenario: Agent killed after 5 identical writes
- **WHEN** the sidecar's stdout reader processes 5 consecutive message_end events with identical tool_use name and input length
- **THEN** the agent process is killed and the sidecar logs "Loop detected"

### Requirement: resolveWorkDir covers all workspace layouts
The system SHALL have an integration test that verifies resolveWorkDir returns the correct path for every workspace layout.

#### Scenario: Single repo in workspace root
- **WHEN** `/workspace/.git` exists (single repo cloned into root)
- **THEN** `resolveWorkDir("/workspace")` returns "/workspace"

#### Scenario: Repo in subdirectory
- **WHEN** `/workspace/neph.nvim/.git` exists (repo in subdirectory)
- **THEN** `resolveWorkDir("/workspace")` returns "/workspace/neph.nvim"

#### Scenario: Legacy src layout
- **WHEN** `/workspace/src/neph.nvim/.git` exists (old layout)
- **THEN** `resolveWorkDir("/workspace")` returns "/workspace/src/neph.nvim"

#### Scenario: Explicit path passthrough
- **WHEN** repoPath is "/workspace/custom/path"
- **THEN** `resolveWorkDir("/workspace/custom/path")` returns "/workspace/custom/path" unchanged
