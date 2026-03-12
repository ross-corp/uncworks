## ADDED Requirements

### Requirement: Release Please automates versioning from conventional commits
A GitHub Actions workflow SHALL run release-please on every push to `main`. Release-please SHALL parse conventional commits to determine version bumps: `fix:` → patch, `feat:` → minor, `!` or `BREAKING CHANGE:` → major.

#### Scenario: Feature commit lands on main
- **WHEN** a commit with type `feat` is pushed to main
- **THEN** release-please creates or updates a Release PR with a minor version bump

#### Scenario: Fix commit lands on main
- **WHEN** a commit with type `fix` is pushed to main
- **THEN** release-please creates or updates a Release PR with a patch version bump

#### Scenario: Breaking change lands on main
- **WHEN** a commit with `feat!:` or a `BREAKING CHANGE:` footer is pushed to main
- **THEN** release-please creates or updates a Release PR with a major version bump

### Requirement: Release PR contains auto-generated changelog
The Release PR created by release-please SHALL include an updated `CHANGELOG.md` with entries grouped by commit type (Features, Bug Fixes, Performance, Code Refactoring). Documentation, chore, test, CI, and build commits SHALL be excluded from the changelog.

#### Scenario: Release PR merged
- **WHEN** the Release PR is merged to main
- **THEN** release-please creates a GitHub Release with the changelog as release notes
- **THEN** release-please tags the commit with the new semver version (e.g., `v0.2.0`)

#### Scenario: Multiple commits accumulate
- **WHEN** multiple commits land on main before the Release PR is merged
- **THEN** release-please updates the existing Release PR with all accumulated changes

### Requirement: Release Please configuration files exist
The project SHALL have `release-please-config.json` and `.release-please-manifest.json` at the root. Release type SHALL be `go`. Initial version SHALL be `0.1.0`.

#### Scenario: Config files present
- **WHEN** release-please runs
- **THEN** it reads `release-please-config.json` for configuration and `.release-please-manifest.json` for current version state

### Requirement: GitHub Actions workflow for release-please
A workflow file SHALL exist at `.github/workflows/release-please.yml` that runs on push to main with contents:write and pull-requests:write permissions.

#### Scenario: Workflow triggers
- **WHEN** a commit is pushed to the `main` branch
- **THEN** the release-please GitHub Action runs automatically
