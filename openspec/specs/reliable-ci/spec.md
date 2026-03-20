# reliable-ci Specification

## Purpose
TBD - created by archiving change fix-ci-pipeline. Update Purpose after archive.
## Requirements
### Requirement: CI passes on every commit
The system SHALL have a CI pipeline that passes reliably on every push to main and every PR.

#### Scenario: Go build succeeds
- **WHEN** CI runs on a push to main
- **THEN** `go build ./cmd/... ./internal/...` SHALL succeed

#### Scenario: Go tests pass
- **WHEN** CI runs on a push to main
- **THEN** `go test ./api/... ./internal/...` SHALL pass with envtest properly configured

#### Scenario: TypeScript checks pass
- **WHEN** CI runs on a push to main
- **THEN** `npx tsc --noEmit` SHALL pass for web, shared, and extension packages

#### Scenario: lint passes
- **WHEN** CI runs on a push to main
- **THEN** `golangci-lint run` SHALL report 0 issues

