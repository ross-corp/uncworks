## MODIFIED Requirements

### Requirement: Lint task uses golangci-lint
The `task lint` command SHALL run `golangci-lint run ./...` instead of `go vet ./...` for Go linting. TypeScript type checks SHALL remain unchanged.

#### Scenario: Lint task runs golangci-lint
- **WHEN** a developer runs `task lint`
- **THEN** golangci-lint executes with the project's `.golangci.yml` configuration
- **THEN** TypeScript type checks run for web, shared, and pi-aot-extension packages

#### Scenario: golangci-lint config exists
- **WHEN** golangci-lint runs
- **THEN** it reads `.golangci.yml` from the project root for linter selection and configuration
