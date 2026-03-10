## ADDED Requirements

### Requirement: Lefthook manages all git hooks
The project SHALL use lefthook as the git hook manager. A `lefthook.yml` configuration file SHALL exist at the project root defining all hook stages.

#### Scenario: Hooks installed on devbox shell entry
- **WHEN** a developer runs `devbox shell`
- **THEN** lefthook hooks are automatically installed via the init_hook

#### Scenario: Hooks installed manually
- **WHEN** a developer runs `lefthook install`
- **THEN** git hooks are installed in `.git/hooks/`

### Requirement: Pre-commit hook runs fast quality checks on staged files
The pre-commit hook SHALL run formatting and linting checks on staged files only. All checks SHALL execute in parallel. The hook SHALL complete in under 10 seconds on a warm cache.

#### Scenario: Go files staged
- **WHEN** a developer commits with staged `.go` files
- **THEN** `go fmt` runs on staged Go files and auto-fixes formatting
- **THEN** `golangci-lint run` runs on staged Go files and fails on violations

#### Scenario: Proto files staged
- **WHEN** a developer commits with staged `.proto` files
- **THEN** `buf lint` runs and fails on violations

#### Scenario: TypeScript files staged
- **WHEN** a developer commits with staged TypeScript files
- **THEN** `tsc --noEmit` runs for affected packages and fails on type errors

#### Scenario: No relevant files staged
- **WHEN** a developer commits with only non-code files staged (e.g., markdown)
- **THEN** no linting commands run and the commit proceeds immediately

### Requirement: Pre-push hook runs heavier checks
The pre-push hook SHALL run the full Go test suite and protobuf breaking-change detection before code leaves the developer's machine.

#### Scenario: Push to any branch
- **WHEN** a developer pushes to any branch
- **THEN** `go test ./api/... ./internal/...` runs and fails on test failures
- **THEN** `buf breaking --against '.git#branch=main'` runs and fails on breaking proto changes

### Requirement: Lefthook available in devbox
Lefthook SHALL be declared as a package in `devbox.json` so it is available without manual installation.

#### Scenario: Fresh devbox shell
- **WHEN** a developer runs `devbox shell` for the first time
- **THEN** the `lefthook` binary is available on PATH
