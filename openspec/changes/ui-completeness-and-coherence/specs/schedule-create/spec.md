## ADDED Requirements

### Requirement: Create Schedule route
The system SHALL provide a form at `/schedules/new` for creating a Schedule CRD.

#### Scenario: Navigate to create
- **WHEN** user clicks "+ new schedule" in ScheduleListView header
- **THEN** navigates to /schedules/new

### Requirement: Schedule form fields
The form SHALL include: name (slug, required), displayName, cron expression (required) with live human-readable preview via cronstrue, timezone (text input, default UTC), concurrencyPolicy (select: Allow/Forbid/Replace, default Forbid), and suspend toggle (default false).

#### Scenario: Cron preview
- **WHEN** user types a valid cron expression
- **THEN** below the input shows e.g. "Every day at 9:00 AM"

#### Scenario: Invalid cron
- **WHEN** cron expression is invalid
- **THEN** preview shows "Invalid cron expression" in red; submit disabled

### Requirement: Schedule target selector
The form SHALL include a radio toggle "Chain" / "Template" that controls which ref field is shown. Only one of chainRef or templateRef SHALL be sent to the API.

#### Scenario: Chain target
- **WHEN** user selects "Chain"
- **THEN** a select of available chains (GET /api/v1/chains) is shown; templateRef hidden

#### Scenario: Template target
- **WHEN** user selects "Template"
- **THEN** a select of available templates (GET /api/v1/templates) is shown; chainRef hidden

### Requirement: Schedule creation submission
On submit, POST to /api/v1/schedules and redirect to /schedules on success.

#### Scenario: Successful create
- **WHEN** name, cron, and a target ref are set and user submits
- **THEN** POST /api/v1/schedules called, redirect to /schedules, toast.success shown

#### Scenario: Submit disabled
- **WHEN** name empty OR cron invalid OR no target ref selected
- **THEN** submit button disabled

### Requirement: Delete Schedule
The system SHALL provide a delete button per schedule in ScheduleListView.

#### Scenario: Successful delete
- **WHEN** user confirms delete
- **THEN** DELETE /api/v1/schedules/:name called, list refreshes, toast.success shown
