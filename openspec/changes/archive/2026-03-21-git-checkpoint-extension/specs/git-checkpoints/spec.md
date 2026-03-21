## ADDED Requirements

### Requirement: Checkpoint commit after each tool call
The system SHALL create a git commit in the agent's workspace after each tool execution completes, capturing the full state of all file changes.

#### Scenario: Write tool creates checkpoint
- **WHEN** the agent executes a `write` tool that creates `/workspace/neph.nvim/HELLO.md`
- **THEN** a git commit with message `"aot-checkpoint: write"` SHALL be created in the workspace
- **AND** the commit SHALL include the new file

#### Scenario: No-change tool skips checkpoint
- **WHEN** the agent executes a `read` tool that doesn't modify any files
- **THEN** no checkpoint commit SHALL be created (git status is clean)

### Requirement: Diff between consecutive checkpoints
The system SHALL compute the diff between the previous checkpoint commit and the current checkpoint commit and attach it to the trace span.

#### Scenario: Span has diff from checkpoint
- **WHEN** a tool execution creates a checkpoint commit
- **AND** a previous checkpoint exists
- **THEN** the span's diff SHALL contain the output of `git diff {prevSHA}..{currentSHA}`
- **AND** `hasDiff` SHALL be true
- **AND** the diff SHALL include file paths and patch content

#### Scenario: First checkpoint diffs against parent
- **WHEN** the first tool execution creates a checkpoint commit
- **AND** no previous checkpoint exists
- **THEN** the span's diff SHALL contain the output of `git diff HEAD~1..HEAD`

### Requirement: Checkpoint metadata in spans
The system SHALL include checkpoint commit SHAs in the trace span metadata.

#### Scenario: Span metadata includes checkpoint SHAs
- **WHEN** a tool execution creates a checkpoint at SHA `abc1234`
- **AND** the previous checkpoint was `def5678`
- **THEN** the span metadata SHALL include `checkpointSHA: "abc1234"` and `prevCheckpointSHA: "def5678"`

### Requirement: Git config before first checkpoint
The system SHALL configure git user.name and user.email in the workspace before creating the first checkpoint.

#### Scenario: Git config set on agent start
- **WHEN** the agent starts in a workspace
- **THEN** `git config user.name` SHALL be set to `"aot-agent"`
- **AND** `git config user.email` SHALL be set to `"agent@aot.uncworks.io"`

### Requirement: Checkpoint state reset between stages
The system SHALL reset the checkpoint tracking state when a new agent starts (new pipeline stage).

#### Scenario: Execute stage starts fresh
- **WHEN** the plan stage completes and the execute stage starts
- **THEN** the checkpoint SHA tracker SHALL be reset
- **AND** the first execute-stage checkpoint SHALL diff against HEAD~1 (not against plan-stage checkpoints)
