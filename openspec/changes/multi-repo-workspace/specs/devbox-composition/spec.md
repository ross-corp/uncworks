## ADDED Requirements

### Requirement: Auto-compose devbox from repo configs
The hydrator SHALL scan each cloned repo's worktree for a `devbox.json` file and generate a root-level `/workspace/devbox.json` that includes all discovered configs using devbox's `include` directive.

#### Scenario: Multiple repos with devbox configs
- **WHEN** `src/api/devbox.json` and `src/web/devbox.json` exist
- **THEN** `/workspace/devbox.json` is generated with `"include": ["src/api/devbox.json", "src/web/devbox.json"]`
- **AND** `devbox install` runs at `/workspace`

#### Scenario: Single repo with devbox config
- **WHEN** only `src/api/devbox.json` exists
- **THEN** `/workspace/devbox.json` is generated with `"include": ["src/api/devbox.json"]`
- **AND** `devbox install` runs at `/workspace`

#### Scenario: No repos have devbox configs
- **WHEN** no cloned repos contain a `devbox.json`
- **THEN** no `/workspace/devbox.json` is generated
- **AND** `devbox install` is not run

### Requirement: Explicit devbox_config overrides auto-composition
When `AgentRunSpec.devbox_config` is set, the hydrator SHALL use that path directly instead of auto-composing from repo configs.

#### Scenario: Explicit devbox config specified
- **WHEN** `devbox_config` is set to `devbox.json` and a repo at `src/api` contains that file
- **THEN** the hydrator uses `src/api/devbox.json` directly (existing behavior)
- **AND** auto-composition does NOT run

#### Scenario: No explicit devbox config
- **WHEN** `devbox_config` is empty
- **THEN** auto-composition runs, scanning all repos for `devbox.json`

### Requirement: Devbox install runs at workspace root
When auto-composition is used, `devbox install` SHALL run at `/workspace` (where the composed `devbox.json` lives) rather than in a specific repo directory.

#### Scenario: Composed devbox install
- **WHEN** a composed `/workspace/devbox.json` has been generated
- **THEN** `devbox install` runs with working directory `/workspace`

### Requirement: Devbox composition errors surface clearly
If `devbox install` fails due to conflicting packages across repo configs, the error MUST be surfaced in the hydration phase with context about which source configs were composed.

#### Scenario: Conflicting package versions
- **WHEN** `src/api/devbox.json` requires `go@1.21` and `src/web/devbox.json` requires `go@1.22`
- **AND** `devbox install` fails
- **THEN** the hydration error includes the list of source configs that were composed

### Requirement: Agent can install additional devbox packages
The agent SHALL be able to run `devbox add <package>` during a run to install packages not covered by the composed environment.

#### Scenario: Agent discovers missing tool
- **WHEN** the agent needs `jq` but no repo config includes it
- **THEN** the agent can run `devbox add jq` from `/workspace` and the package becomes available
