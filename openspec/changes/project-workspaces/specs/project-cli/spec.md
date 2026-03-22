## ADDED Requirements

### Requirement: Project CRUD Commands
The CLI SHALL provide `project create`, `project list`, `project get`, and `project delete` commands. `project create` SHALL accept flags for name, repos, devbox packages, default model, and authorized keys. `project list` SHALL display all projects with their status. `project get` SHALL display the full configuration and status of a single project. `project delete` SHALL remove the project and its associated resources.

#### Scenario: Create a project via CLI
- **WHEN** a user runs `uncworks project create --name my-project --repo https://github.com/org/app --devbox python3,nodejs --model claude-sonnet`
- **THEN** the CLI SHALL create a Project resource with the specified configuration and display a confirmation message

#### Scenario: List projects via CLI
- **WHEN** a user runs `uncworks project list`
- **THEN** the CLI SHALL display a table of all projects with columns for name, configRepoReady, runCount, and totalCost

### Requirement: Project Clone and Push
The CLI SHALL provide a `project clone` command that clones the project's config repo from soft-serve to the local filesystem. The CLI SHALL provide a `project push` command that commits and pushes local changes back to the project's config repo.

#### Scenario: Clone a project config repo
- **WHEN** a user runs `uncworks project clone my-project`
- **THEN** the CLI SHALL clone the project's soft-serve config repo into a local directory named `my-project`

#### Scenario: Push local changes to config repo
- **WHEN** a user runs `uncworks project push` from within a cloned project directory
- **THEN** the CLI SHALL commit any uncommitted changes and push them to the project's soft-serve config repo

### Requirement: Project Run Command
The CLI SHALL provide a `run --project --spec` command that creates an AgentRun with the specified project and spec references. The `--project` flag SHALL set the `projectRef` and the `--spec` flag SHALL set the `specRef` on the created AgentRun.

#### Scenario: Start a run with project and spec
- **WHEN** a user runs `uncworks run --project my-project --spec openspec/auth/spec.md`
- **THEN** the CLI SHALL create an AgentRun with `projectRef: my-project` and `specRef: openspec/auth/spec.md` and stream the run output

### Requirement: SSH and IDE Access Commands
The CLI SHALL provide an `ssh` command that connects to a project's IDE pod through the SSH gateway. The CLI SHALL provide an `ide open` command that opens the project's code-server UI in the user's default browser.

#### Scenario: SSH into a project
- **WHEN** a user runs `uncworks ssh my-project`
- **THEN** the CLI SHALL open an SSH connection to the gateway on port 30022 with the username set to `my-project`

#### Scenario: Open IDE in browser
- **WHEN** a user runs `uncworks ide open my-project`
- **THEN** the CLI SHALL open the user's default browser to the code-server URL for the project's IDE pod
