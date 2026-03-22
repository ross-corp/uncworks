## ADDED Requirements

### Requirement: Project Custom Resource Definition
The system SHALL define a Project CRD with fields for `repos` (list of Git repository URLs), `devboxPackages` (list of Nix packages), `defaults` (default model, budget, and timeout for runs), `ideConfig` (editor extensions and settings), and `authorizedKeys` (list of SSH public keys). Each Project resource SHALL have a unique name within its namespace.

#### Scenario: Create a project with full configuration
- **WHEN** a user creates a Project resource with repos, devboxPackages, defaults, ideConfig, and authorizedKeys populated
- **THEN** the resource SHALL be persisted and all fields SHALL be retrievable via the Kubernetes API

#### Scenario: Reject a project with missing required fields
- **WHEN** a user attempts to create a Project resource without a name
- **THEN** the API server SHALL reject the request with a validation error

### Requirement: Project Config Repo Lifecycle
The project controller SHALL create a soft-serve config repository for each Project resource upon creation. The controller SHALL delete the corresponding soft-serve repository when the Project resource is deleted.

#### Scenario: Controller provisions config repo on project creation
- **WHEN** a new Project resource is created
- **THEN** the controller SHALL create a matching repository in soft-serve and set `status.configRepoReady` to `true` once the repo is available

#### Scenario: Controller removes config repo on project deletion
- **WHEN** a Project resource is deleted
- **THEN** the controller SHALL delete the corresponding soft-serve repository and release any associated storage

### Requirement: Project Status Tracking
The Project status SHALL include `configRepoReady` (boolean indicating the config repo is provisioned), `runCount` (total number of AgentRuns associated with the project), and `totalCost` (cumulative cost of all runs in the project).

#### Scenario: Status reflects completed runs
- **WHEN** an AgentRun with a `projectRef` pointing to a project completes with a reported cost
- **THEN** the project's `status.runCount` SHALL increment by one and `status.totalCost` SHALL increase by the run's cost

### Requirement: Project CRUD Operations
The system SHALL support list, create, update, and delete operations on Project resources. Updates to `repos`, `devboxPackages`, `defaults`, `ideConfig`, or `authorizedKeys` SHALL take effect on subsequent runs and IDE pod restarts.

#### Scenario: List all projects
- **WHEN** a user requests a list of projects in a namespace
- **THEN** the system SHALL return all Project resources in that namespace with their current status

#### Scenario: Update project devbox packages
- **WHEN** a user updates the `devboxPackages` field of an existing Project
- **THEN** the change SHALL be persisted and the next IDE pod restart or agent run SHALL use the updated package list
