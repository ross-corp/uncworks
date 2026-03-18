## ADDED Requirements

### Requirement: Create agent run form
The web UI SHALL provide a form to create new agent runs. The form SHALL include: a repos section (at least one repo with url required, branch and path optional), a prompt textarea (required), a backend selector (Pod/KubeVirt/External), and collapsible optional fields (devboxConfig, ttlSeconds, envVars as key-value pairs, image).

#### Scenario: Submit a single-repo run
- **WHEN** user fills in one repo URL and a prompt, then clicks Create
- **THEN** the system calls AOTClient.createAgentRun with the form data
- **AND** on success, navigates to the new run's detail page

#### Scenario: Submit a multi-repo run
- **WHEN** user adds multiple repos with URLs, branches, and paths
- **THEN** all repos are included in the createAgentRun request's repos array

#### Scenario: Validation prevents empty submission
- **WHEN** user clicks Create without filling required fields (repo url, prompt)
- **THEN** the form shows validation errors and does not submit

#### Scenario: Server error on create
- **WHEN** the createAgentRun call fails
- **THEN** the form displays the error message and remains editable
