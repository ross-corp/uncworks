## ADDED Requirements

### Requirement: Git Worktree Provisioning
The system SHALL provision an isolated Git Worktree for every new `AgentRun`.

#### Scenario: Worktree creation on startup
- **WHEN** the `AgentRun` Pod starts up
- **THEN** the Hydration Init-Container SHALL create a new `git worktree` in the ephemeral local path

### Requirement: Devbox Shell Environment
The execution sandbox SHALL be initialized with the tools and versions specified in `devbox.json`.

#### Scenario: Tool version enforcement
- **WHEN** the agent harness executes a command
- **THEN** the command SHALL be run within the `devbox shell` context to ensure environmental parity
