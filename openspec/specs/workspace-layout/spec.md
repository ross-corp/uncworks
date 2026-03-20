# workspace-layout Specification

## Purpose
TBD - created by archiving change agent-architecture-v2. Update Purpose after archive.
## Requirements
### Requirement: repos as workspace-root worktrees
The system SHALL clone repos as git worktrees directly under `/workspace/` instead of under `/workspace/src/`.

#### Scenario: single repo workspace
- **WHEN** a run is created with one repo (e.g., `neph.nvim`)
- **THEN** the hydration init container SHALL create the worktree at `/workspace/neph.nvim/`

#### Scenario: multi repo workspace
- **WHEN** a run is created with multiple repos
- **THEN** each repo SHALL have its worktree at `/workspace/<repo-name>/`

### Requirement: openspec at workspace level
The system SHALL initialize OpenSpec at `/workspace/.openspec/` (not inside any repo directory).

#### Scenario: openspec init location
- **WHEN** the plan stage runs `openspec init`
- **THEN** the config SHALL be created at `/workspace/.openspec/config.yaml`

#### Scenario: change artifacts location
- **WHEN** an OpenSpec change is scaffolded
- **THEN** artifacts SHALL be at `/workspace/.openspec/changes/<name>/`

#### Scenario: repo openspec isolation
- **WHEN** the target repo has its own `openspec/` directory
- **THEN** the pipeline SHALL NOT modify the repo's openspec directory — it uses the workspace-level one

### Requirement: resolveWorkDir compatibility
The system SHALL update the sidecar's `resolveWorkDir` function to detect the new layout.

#### Scenario: sidecar resolves workspace
- **WHEN** pi starts with `RepoPath=/workspace`
- **THEN** `resolveWorkDir` SHALL detect the repo worktree (e.g., `/workspace/neph.nvim/`) and set it as the working directory

