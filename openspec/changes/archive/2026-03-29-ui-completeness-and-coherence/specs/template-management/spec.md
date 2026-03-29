## ADDED Requirements

### Requirement: RunTemplate list view
The system SHALL display all RunTemplates at `/templates` with name, displayName, description, step count, and age.

#### Scenario: Navigate to templates
- **WHEN** user clicks "Templates" in GlobalNav
- **THEN** `/templates` loads and shows all templates

#### Scenario: Empty state
- **WHEN** no templates exist
- **THEN** view shows "No templates yet" with a "+ new template" button

### Requirement: Create RunTemplate
The system SHALL provide a form at `/templates/new` to create a RunTemplate with fields: name (slug, required), displayName, description, and a prompt textarea.

#### Scenario: Successful creation
- **WHEN** user fills name and submits
- **THEN** POST /api/v1/templates is called, user is redirected to /templates on success

#### Scenario: Validation
- **WHEN** name field is empty
- **THEN** submit button is disabled

### Requirement: Delete RunTemplate
The system SHALL allow deleting a template from the list view with a confirmation step and 409 conflict handling.

#### Scenario: Delete with no references
- **WHEN** user clicks delete on a template not referenced by any chain
- **THEN** DELETE /api/v1/templates/:name is called, template removed from list

#### Scenario: Delete blocked by chain reference
- **WHEN** DELETE returns 409
- **THEN** toast.error shows the conflict message from the API response
