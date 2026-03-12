## ADDED Requirements

### Requirement: Agent runs accept multiple repositories
The system SHALL accept a list of repositories in the `AgentRunSpec`, each with a URL, optional branch, and optional path.

#### Scenario: Create agent run with multiple repos
- **WHEN** a client calls `CreateAgentRun` with `repos: [{url: "https://github.com/org/frontend.git"}, {url: "https://github.com/org/backend.git"}]`
- **THEN** the system SHALL create a CRD with both repositories in the spec
- **AND** the Temporal workflow SHALL receive both repositories in its input

#### Scenario: Create agent run with single repo
- **WHEN** a client calls `CreateAgentRun` with `repos: [{url: "https://github.com/org/repo.git", branch: "main"}]`
- **THEN** the system SHALL behave identically to the previous single-repo behavior

#### Scenario: Empty repos list rejected
- **WHEN** a client calls `CreateAgentRun` with an empty `repos` list
- **THEN** the system SHALL return a validation error

### Requirement: Hydration clones all repositories
The hydration init container SHALL clone and create worktrees for every repository in the spec.

#### Scenario: Multiple repos hydrated into separate directories
- **WHEN** the hydration init container runs with repos `[{url: "https://github.com/org/frontend.git"}, {url: "https://github.com/org/backend.git"}]`
- **THEN** the system SHALL clone each repo bare into `/workspace/.bare/<repo-name>/`
- **AND** create worktrees at `/workspace/src/frontend/` and `/workspace/src/backend/`

#### Scenario: Custom path overrides default directory name
- **WHEN** a repository has `path: "services/api"` specified
- **THEN** the worktree SHALL be created at `/workspace/src/services/api/` instead of the derived repo name

#### Scenario: Default branch detection per repo
- **WHEN** a repository has no branch specified
- **THEN** the hydration SHALL detect the default branch from `git symbolic-ref --short HEAD` on the bare clone

### Requirement: Primary repo determines workspace path
The first repository in the repos list SHALL be considered the primary repo. Its worktree path SHALL be returned as the workspace path for the agent process.

#### Scenario: Workspace path from primary repo
- **WHEN** hydration completes for repos `[{url: ".../frontend.git"}, {url: ".../backend.git"}]`
- **THEN** `WaitForHydrationOutput.WorkspacePath` SHALL be `/workspace/src/frontend`
