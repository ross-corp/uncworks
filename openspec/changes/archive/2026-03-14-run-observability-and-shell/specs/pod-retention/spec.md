## ADDED Requirements

### Requirement: Configurable pod retention after completion
The system SHALL keep agent pods alive for a configurable duration after workflow completion, enabling post-mortem inspection via logs, files, and shell.

#### Scenario: Default retention
- **WHEN** an agent run completes and no `retain_pod_minutes` is specified
- **THEN** the pod remains alive for 30 minutes after completion
- **AND** the pod is deleted after the retention period expires

#### Scenario: Custom retention
- **WHEN** an agent run is created with `retain_pod_minutes: 60`
- **THEN** the pod remains alive for 60 minutes after completion

#### Scenario: Zero retention (immediate cleanup)
- **WHEN** an agent run is created with `retain_pod_minutes: 0`
- **THEN** the pod is deleted immediately on completion (current behavior preserved)

#### Scenario: Retention countdown visible in UI
- **WHEN** a completed run's pod is still retained
- **THEN** the detail panel shows a "Pod expires in X minutes" indicator
- **AND** the Files and Shell tabs remain functional

### Requirement: Log collection before pod deletion
The system SHALL collect and persist agent log output before deleting the pod.

#### Scenario: Logs persisted on cleanup
- **WHEN** the retention period expires and the pod is about to be deleted
- **THEN** the workflow collects the last 1MB of agent log output
- **AND** stores it on the AgentRun CRD status `logOutput` field
