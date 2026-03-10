## MODIFIED Requirements

### Requirement: Five-Stage Test Pipeline
The test pipeline SHALL enforce a layered testing strategy with five distinct stages.

#### Scenario: Pipeline stages
- **GIVEN** the CI test pipeline
- **THEN** it SHALL have five stages: schema gate, unit, contract, integration, E2E
- **AND** each stage SHALL gate the next (a failure blocks subsequent stages)

#### Scenario: Independent execution
- **GIVEN** a developer working locally
- **WHEN** they need to run a specific test stage
- **THEN** each stage SHALL be independently runnable via task targets

### Requirement: Taskfile Test Targets
The Taskfile SHALL provide targets for each test stage.

#### Scenario: Required task targets
- **GIVEN** the Taskfile
- **THEN** it SHALL include the following targets:
  - `test:unit` -- Run unit tests (Go + TypeScript, no infrastructure)
  - `test:contract` -- Run GripMock service contract tests (Docker only)
  - `test:temporal` -- Run Temporal workflow tests (unit + integration)
  - `test:integration` -- Run integration tests (envtest, testcontainers, temporal-cli)
  - `test:e2e` -- Run full E2E tests (requires k0s cluster)
  - `test:e2e:setup` -- Deploy all E2E dependencies to k0s cluster

### Requirement: testcontainers for PostgreSQL
PostgreSQL integration tests SHALL use real PostgreSQL via testcontainers.

#### Scenario: Brain store integration tests
- **GIVEN** integration tests for the brain store
- **WHEN** the tests require PostgreSQL
- **THEN** they SHALL use `github.com/testcontainers/testcontainers-go` to spin up a real PostgreSQL container
- **AND** this SHALL replace any mock or in-memory DB approaches

### Requirement: envtest for Kubernetes Controllers
Kubernetes controller tests SHALL use envtest (unchanged from existing).

#### Scenario: Controller integration tests
- **GIVEN** integration tests for Kubernetes controllers
- **WHEN** the tests require a Kubernetes API server
- **THEN** they SHALL use `sigs.k8s.io/controller-runtime/pkg/envtest`

### Requirement: Go Test Tag Separation
Go test tags SHALL be used to separate test stages.

#### Scenario: Unit tests (default)
- **GIVEN** Go unit tests
- **THEN** they SHALL run without any build tags (default `go test`)

#### Scenario: Integration tests
- **GIVEN** Go integration tests
- **THEN** they SHALL require the `-tags integration` build tag

#### Scenario: E2E tests
- **GIVEN** Go E2E tests
- **THEN** they SHALL require the `-tags e2e` build tag
