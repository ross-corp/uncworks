## ADDED Requirements

### Requirement: Hydrator generates uncspace.yaml
The hydrator SHALL generate a `/workspace/uncspace.yaml` file after cloning all repos and before running devbox setup. The file MUST be a valid YAML document.

#### Scenario: Multi-repo workspace
- **WHEN** the hydrator clones 2 repos (api at `src/api`, web at `src/web`) on branches `main` and `develop`
- **THEN** `/workspace/uncspace.yaml` is created with a `repos` array containing entries for each repo with `path`, `url`, and `branch` fields

#### Scenario: Zero-repo workspace
- **WHEN** the hydrator runs with an empty repos list
- **THEN** `/workspace/uncspace.yaml` is created with `repos: []`

#### Scenario: Single-repo workspace
- **WHEN** the hydrator clones 1 repo
- **THEN** `/workspace/uncspace.yaml` is created with a single entry in `repos`

### Requirement: uncspace.yaml includes devbox source metadata
The `uncspace.yaml` file SHALL include a `devbox` section that lists which repo-level `devbox.json` files were composed into the root environment.

#### Scenario: Repos with devbox configs
- **WHEN** repos `src/api` and `src/web` both contain `devbox.json` files
- **THEN** `uncspace.yaml` contains `devbox.composed: true` and `devbox.sources` listing `src/api/devbox.json` and `src/web/devbox.json`

#### Scenario: No repos have devbox configs
- **WHEN** no cloned repos contain a `devbox.json` file
- **THEN** `uncspace.yaml` contains `devbox.composed: false` and `devbox.sources` is empty

#### Scenario: Mixed repos
- **WHEN** `src/api` has a `devbox.json` but `src/web` does not
- **THEN** `devbox.sources` lists only `src/api/devbox.json`

### Requirement: Repo paths are relative to workspace root
All paths in `uncspace.yaml` SHALL be relative to `/workspace`.

#### Scenario: Path format
- **WHEN** a repo is cloned to `/workspace/src/my-api`
- **THEN** the path field in `uncspace.yaml` reads `src/my-api`, not `/workspace/src/my-api`

### Requirement: Agent starts at workspace root
The workflow SHALL pass `/workspace` as the agent's working directory instead of the first repo's worktree path.

#### Scenario: Multi-repo agent start
- **WHEN** an agent run has 2+ repos
- **THEN** the `StartAgent` activity receives `RepoPath: "/workspace"`

#### Scenario: Single-repo agent start
- **WHEN** an agent run has 1 repo
- **THEN** the `StartAgent` activity still receives `RepoPath: "/workspace"`

### Requirement: Sidecar uses workspace root
The sidecar SHALL start the agent process with working directory `/workspace` and run `devbox run` from there.

#### Scenario: Agent process startup
- **WHEN** the sidecar receives a `StartAgent` request
- **THEN** `cmd.Dir` is set to `/workspace` (or the value of `RepoPath` which is now always `/workspace`)
