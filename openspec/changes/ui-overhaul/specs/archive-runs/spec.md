## ADDED Requirements

### Requirement: Runs can be archived
The system SHALL support archiving runs, which hides them from the default run list view and deletes their associated PVC.

#### Scenario: Archive a single run
- **WHEN** a user archives a run via the UI or API
- **THEN** the run's status SHALL include `archived: true` AND the run SHALL not appear in the default run list

#### Scenario: Archived runs are still queryable
- **WHEN** a user enables the "show archived" toggle
- **THEN** archived runs SHALL appear in the list with a visual indicator that they are archived

#### Scenario: Archived runs cannot be retried
- **WHEN** a user views an archived run
- **THEN** the retry button SHALL be disabled or hidden AND the clone button SHALL remain available

#### Scenario: PVC is deleted on archive
- **WHEN** a run is archived
- **THEN** the system SHALL delete the PersistentVolumeClaim associated with the run's deployment

### Requirement: Mass archive via multi-select
The system SHALL allow selecting multiple runs and archiving them in bulk.

#### Scenario: Select multiple runs with checkboxes
- **WHEN** a user enters selection mode
- **THEN** checkboxes SHALL appear next to each run row AND a bulk actions bar SHALL appear

#### Scenario: Bulk archive selected runs
- **WHEN** a user selects multiple runs and clicks "Archive selected"
- **THEN** all selected runs SHALL be archived AND their PVCs deleted

### Requirement: Archive API
The system SHALL expose API endpoints for archiving and unarchiving runs.

#### Scenario: Archive via API
- **WHEN** a PATCH request is sent to `/api/v1/runs/{id}/archive` with `{"archived": true}`
- **THEN** the run's status SHALL be updated to archived

#### Scenario: Bulk archive via API
- **WHEN** a POST request is sent to `/api/v1/runs/bulk-archive` with a list of run IDs
- **THEN** all specified runs SHALL be archived
