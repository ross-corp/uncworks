# archive-cleanup Specification

## Purpose
TBD - created by archiving change archive-cleanup. Update Purpose after archive.
## Requirements
### Requirement: Expired runs have Deployment and PVC deleted

The controller SHALL delete the Deployment and PVC for AgentRuns that have been in a terminal phase (Succeeded, Failed, Cancelled) for longer than the configured retention period.

#### Scenario: Run older than retention period is cleaned up
- **WHEN** an AgentRun has `status.phase` in {Succeeded, Failed, Cancelled}
- **AND** `status.completedAt` is older than `AOT_RETENTION_DAYS` (default 7)
- **THEN** the controller deletes the Deployment named in `status.deploymentName`
- **AND** the controller deletes the PVC named `workspace-<agentrun-name>`
- **AND** the controller sets annotation `aot.uncworks.io/archived: "true"` on the CRD

#### Scenario: Run within retention period is not cleaned up
- **WHEN** an AgentRun has `status.phase` = Succeeded
- **AND** `status.completedAt` is less than `AOT_RETENTION_DAYS` ago
- **THEN** the Deployment and PVC are NOT deleted
- **AND** no archived annotation is set

#### Scenario: Already archived run is skipped
- **WHEN** an AgentRun has annotation `aot.uncworks.io/archived: "true"`
- **THEN** the cleanup loop skips it without attempting deletion

#### Scenario: Missing Deployment or PVC is tolerated
- **WHEN** the Deployment or PVC for an expired run has already been manually deleted
- **THEN** the controller ignores the NotFound error
- **AND** still sets the archived annotation

### Requirement: Retention period is configurable

#### Scenario: Custom retention via environment variable
- **WHEN** `AOT_RETENTION_DAYS` is set to `14`
- **THEN** only runs completed more than 14 days ago are cleaned up

#### Scenario: Default retention
- **WHEN** `AOT_RETENTION_DAYS` is not set
- **THEN** the default retention period is 7 days

### Requirement: CRD is preserved after cleanup

#### Scenario: Archived run remains queryable
- **WHEN** a run's Deployment and PVC have been deleted by the cleanup loop
- **THEN** the AgentRun CRD still exists
- **AND** `kubectl get agentruns` still shows the run
- **AND** the API server can still return the run in list/get responses

