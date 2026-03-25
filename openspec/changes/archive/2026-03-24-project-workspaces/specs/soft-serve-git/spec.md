## ADDED Requirements

### Requirement: Soft-Serve Deployment
Soft-serve SHALL be deployed as a single pod with a PersistentVolumeClaim for repository storage. The pod SHALL expose an SSH endpoint accessible to other pods within the cluster.

#### Scenario: Soft-serve pod is running and reachable
- **WHEN** the soft-serve pod is deployed
- **THEN** it SHALL be reachable via its cluster service DNS name on the SSH port from any pod in the cluster

#### Scenario: Repository data survives pod restart
- **WHEN** the soft-serve pod restarts
- **THEN** all previously created repositories and their contents SHALL be intact on the PVC

### Requirement: Per-Project Repository
Each Project resource SHALL have a dedicated repository in soft-serve. The repository name SHALL match the Project resource name.

#### Scenario: Repository created for new project
- **WHEN** the project controller processes a new Project resource named `my-project`
- **THEN** a soft-serve repository named `my-project` SHALL exist and be cloneable via SSH within the cluster

### Requirement: Repository Scaffolding
When a project repository is created, it SHALL be scaffolded with a `devbox.json` file reflecting the project's `devboxPackages`, an `openspec/` directory for specification files, and a `.devcontainer/devcontainer.json` with default editor configuration.

#### Scenario: Scaffolded repo contains required files
- **WHEN** a new project repository is created for a project with devboxPackages `["python3", "nodejs"]`
- **THEN** the initial commit SHALL contain a `devbox.json` listing `python3` and `nodejs`, an empty `openspec/` directory, and a valid `.devcontainer/devcontainer.json`

### Requirement: Controller-Managed Repo Lifecycle
The project controller SHALL manage the full lifecycle of soft-serve repositories. The controller SHALL create repos on Project creation and delete repos on Project deletion. Manual repo creation outside the controller SHALL NOT be required.

#### Scenario: Repo deleted when project is removed
- **WHEN** a Project resource is deleted
- **THEN** the corresponding soft-serve repository SHALL be deleted and its storage reclaimed
