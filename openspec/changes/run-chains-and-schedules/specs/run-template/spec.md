## ADDED Requirements

### Requirement: RunTemplate CRD stores reusable run configurations
The system SHALL provide a RunTemplate custom resource that captures a complete, reusable run configuration. A RunTemplate SHALL reference a prompt, model tier, orchestration mode, repos, push/PR settings, and optionally a project. A RunTemplate SHALL NOT contain scheduling or dependency logic.

#### Scenario: Create a RunTemplate with all fields
- **WHEN** a user creates a RunTemplate with prompt, repos, modelTier, orchestrationMode, autoPush, autoPR, and projectRef
- **THEN** the system persists the RunTemplate in the Kubernetes API
- **AND** the RunTemplate status shows phase "Ready"

#### Scenario: Create a RunTemplate with minimal fields
- **WHEN** a user creates a RunTemplate with only a prompt and one repo URL
- **THEN** the system persists the RunTemplate with defaults for modelTier ("default"), orchestrationMode ("single"), autoPush (false), autoPR (false)
- **AND** the RunTemplate status shows phase "Ready"

#### Scenario: RunTemplate inherits project defaults
- **WHEN** a RunTemplate specifies a projectRef and omits modelTier, repos, and TTL
- **THEN** the system resolves empty fields from the referenced Project's defaults and repos at trigger time
- **AND** the resolved values are used when creating AgentRuns from the template

#### Scenario: RunTemplate validation rejects empty prompt
- **WHEN** a user creates a RunTemplate with an empty prompt and no specRef
- **THEN** the API returns a validation error indicating that either prompt or specRef is required

#### Scenario: Manually trigger a RunTemplate
- **WHEN** a user calls the REST API to trigger a RunTemplate by name
- **THEN** the system creates a new AgentRun with the template's configuration
- **AND** the AgentRun's labels include `aot.uncworks.io/run-template: {templateName}`

### Requirement: RunTemplate CRUD API
The REST API SHALL expose endpoints for creating, listing, getting, updating, and deleting RunTemplates. All endpoints SHALL be scoped to the aot-system namespace.

#### Scenario: List RunTemplates
- **WHEN** a user calls GET /api/v1/run-templates
- **THEN** the system returns all RunTemplates in the namespace
- **AND** each item includes metadata.name, spec, and status

#### Scenario: List RunTemplates filtered by project
- **WHEN** a user calls GET /api/v1/run-templates?project=my-project
- **THEN** the system returns only RunTemplates whose spec.projectRef matches "my-project"

#### Scenario: Update a RunTemplate
- **WHEN** a user calls PUT /api/v1/run-templates/{name} with updated fields
- **THEN** the system updates the RunTemplate spec
- **AND** existing Schedules or Chains referencing this template use the updated configuration on their next trigger

#### Scenario: Delete a RunTemplate referenced by a Chain
- **WHEN** a user calls DELETE /api/v1/run-templates/{name} and the template is referenced by an active Chain
- **THEN** the API returns a 409 Conflict error with a message listing the referencing Chains

### Requirement: RunTemplate UI management
The web UI SHALL provide controls for creating, editing, and deleting RunTemplates from within the project detail view and the new-run form.

#### Scenario: Save current run configuration as a template
- **WHEN** a user fills out the new-run form and clicks "Save as Template"
- **THEN** the system prompts for a template name
- **AND** creates a RunTemplate with the form's current values

#### Scenario: Start a run from a template
- **WHEN** a user selects a RunTemplate from the template picker and clicks "Run"
- **THEN** the system triggers the template and navigates to the new AgentRun detail page
