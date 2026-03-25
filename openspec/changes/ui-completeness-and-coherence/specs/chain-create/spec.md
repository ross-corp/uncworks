## ADDED Requirements

### Requirement: Create Chain route
The system SHALL provide a form at `/chains/new` for creating a Chain CRD.

#### Scenario: Navigate to create
- **WHEN** user clicks "+ new chain" in ChainListView header
- **THEN** navigates to /chains/new

### Requirement: Chain form fields
The form SHALL include: name (slug, required), displayName, description, projectRef (select from existing projects), and a steps builder section.

#### Scenario: Form renders
- **WHEN** /chains/new loads
- **THEN** all fields render; available templates are fetched from GET /api/v1/templates

### Requirement: Steps builder
The system SHALL allow adding steps one at a time. Each step has: name (unique within form, required), templateRef (select from loaded templates, required), and dependsOn (multi-select from other step names already added).

#### Scenario: Add a step
- **WHEN** user clicks "+ add step"
- **THEN** a new step row appears with name input, templateRef select, and dependsOn multi-select

#### Scenario: dependsOn options
- **WHEN** user opens dependsOn for step N
- **THEN** options show all step names defined before step N in the list

#### Scenario: Remove a step
- **WHEN** user clicks remove on a step
- **THEN** step is removed; any dependsOn references to it in other steps are also cleared

### Requirement: Chain creation submission
On submit, the system SHALL POST to /api/v1/chains with the constructed ChainSpec and redirect to /chains on success.

#### Scenario: Successful create
- **WHEN** name and at least one step with templateRef are filled and user submits
- **THEN** POST /api/v1/chains called, redirect to /chains, toast.success shown

#### Scenario: API error (e.g. DAG validation failure)
- **WHEN** API returns 400 (invalid DAG)
- **THEN** toast.error shows the API error message; form stays open

#### Scenario: Submit disabled
- **WHEN** name is empty OR no steps defined
- **THEN** submit button is disabled

### Requirement: Delete Chain
The system SHALL provide a delete button per chain in ChainListView with 409 conflict handling.

#### Scenario: Successful delete
- **WHEN** user confirms delete and no schedules reference the chain
- **THEN** DELETE /api/v1/chains/:name called, list refreshes

#### Scenario: Delete blocked
- **WHEN** DELETE returns 409
- **THEN** toast.error shows the conflict message
