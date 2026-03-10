## ADDED Requirements

### Requirement: Commit messages follow conventional commits format
All commit messages SHALL follow the Conventional Commits specification: `type(scope): description` or `type: description`. The type SHALL be one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert.

#### Scenario: Valid commit message
- **WHEN** a developer writes a commit message `feat(controller): add TTL enforcement`
- **THEN** the commit-msg hook passes and the commit succeeds

#### Scenario: Valid commit message without scope
- **WHEN** a developer writes a commit message `fix: resolve race condition in event bus`
- **THEN** the commit-msg hook passes and the commit succeeds

#### Scenario: Invalid commit message
- **WHEN** a developer writes a commit message `updated some stuff`
- **THEN** the commit-msg hook fails with a descriptive error explaining the expected format

#### Scenario: Breaking change indicator
- **WHEN** a developer writes a commit message `feat!: redesign AgentRun API`
- **THEN** the commit-msg hook passes and the `!` is recognized as a breaking change indicator

### Requirement: Commitlint enforces conventional commits via git hook
Commitlint SHALL be installed as a dev dependency and invoked by lefthook's commit-msg hook. Configuration SHALL use `@commitlint/config-conventional`.

#### Scenario: Commitlint config exists
- **WHEN** a developer checks the project root
- **THEN** a `commitlint.config.js` file exists extending `@commitlint/config-conventional`

#### Scenario: Commitlint invoked on commit
- **WHEN** a developer makes a commit
- **THEN** lefthook runs `npx commitlint --edit` against the commit message file
