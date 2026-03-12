## ADDED Requirements

### Requirement: Temporal as External Dependency
Temporal server SHALL be an explicit external dependency, not bundled in AOT's Helm chart.

#### Scenario: AOT deployed without Temporal in Helm chart
- **WHEN** AOT's Helm chart is installed
- **THEN** the chart SHALL NOT include Temporal server components
- **AND** the chart SHALL document Temporal as a prerequisite external service

### Requirement: Temporal Connection Configuration
AOT components SHALL connect to Temporal via `TEMPORAL_HOST` and `TEMPORAL_NAMESPACE` environment variables.

#### Scenario: Connecting with default configuration
- **WHEN** `TEMPORAL_HOST` is not set
- **THEN** AOT components SHALL default to `localhost:7233`
- **AND** when `TEMPORAL_NAMESPACE` is not set, AOT components SHALL default to `default`

#### Scenario: Connecting with custom configuration
- **WHEN** `TEMPORAL_HOST` is set to a custom address (e.g., `temporal.infra.svc.cluster.local:7233`)
- **THEN** AOT components SHALL connect to Temporal at the specified address
- **AND** when `TEMPORAL_NAMESPACE` is set to a custom namespace, AOT components SHALL use that namespace

### Requirement: Local Development with temporal-cli
For local development, `temporal-cli` SHALL be available in `devbox.json`.

#### Scenario: Developer enters devbox shell
- **WHEN** a developer runs `devbox shell`
- **THEN** the `temporal` CLI binary SHALL be available on `$PATH`
- **AND** the developer SHALL be able to run `temporal server start-dev` to start a local Temporal server

### Requirement: Temporal Dev Server Task
`task temporal:dev` SHALL start a Temporal dev server using SQLite and a single binary.

#### Scenario: Starting the dev server
- **WHEN** a developer runs `task temporal:dev`
- **THEN** the command SHALL start `temporal server start-dev` with SQLite storage
- **AND** the Temporal Web UI SHALL be accessible at `http://localhost:8233`
- **AND** the Temporal Frontend SHALL be accessible at `localhost:7233`

### Requirement: k0s Production Deployment Documentation
For k0s deployment, documentation SHALL cover deploying `temporalio/helm-charts`.

#### Scenario: Deploying Temporal to k0s
- **WHEN** an operator follows the k0s deployment documentation
- **THEN** the documentation SHALL provide Helm values for deploying Temporal with PostgreSQL persistence
- **AND** the documentation SHALL include steps for creating the `temporal` database on the shared PostgreSQL instance

### Requirement: Temporal Database Isolation
Temporal database SHALL use a separate database on the shared PostgreSQL instance.

#### Scenario: Database provisioning
- **WHEN** Temporal is deployed to k0s with the shared PostgreSQL instance
- **THEN** Temporal SHALL use a database named `temporal` (separate from AOT's application database)
- **AND** Temporal visibility store SHALL use a database named `temporal_visibility`

### Requirement: Production Shard Configuration
Temporal server configuration SHALL recommend 512 shards for production deployments.

#### Scenario: Production Helm values
- **WHEN** the Temporal Helm chart is deployed for production use
- **THEN** the recommended configuration SHALL set `numHistoryShards` to 512
- **AND** the documentation SHALL note that shard count cannot be changed after initial deployment
