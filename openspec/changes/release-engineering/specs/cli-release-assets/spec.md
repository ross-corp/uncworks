## ADDED Requirements

### Requirement: CLI binaries attached to stable GitHub Releases
When a stable `v*` tag is pushed, the pipeline SHALL build the `uncworks` CLI for all 4 supported platforms and upload the resulting binaries as assets on the corresponding GitHub Release.

#### Scenario: Binaries built for all platforms
- **WHEN** a `v*` tag is pushed
- **THEN** binaries are produced for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- **THEN** each binary is named `uncworks-{os}-{arch}` (no extension)

#### Scenario: Binaries uploaded to GitHub Release
- **WHEN** binaries are built successfully
- **THEN** all 4 binaries are uploaded as assets to the GitHub Release matching the tag
- **THEN** the upload uses `gh release upload` (or equivalent)

#### Scenario: Build uses existing Dagger ReleaseBinaries function
- **WHEN** the release-binaries workflow runs
- **THEN** it calls `dagger call release-binaries --source . --version {version}` from `ci/main.go`
- **THEN** no build logic is duplicated in the workflow YAML

### Requirement: Binary version embedded at build time
Each binary SHALL have the release version string embedded via `-ldflags "-X main.version={version}"`.

#### Scenario: Version flag reports correct release version
- **WHEN** `uncworks --version` is run on a released binary
- **THEN** the output matches the git tag version (e.g. `0.3.1`)
