## ADDED Requirements

### Requirement: API server can read PVC host path data
The API server pod SHALL have a hostPath volume mount for `/opt/local-path-provisioner/` so that file, log, and trace endpoints can read PVC-backed data from any run.

#### Scenario: File endpoint reads from PVC path
- **WHEN** a client calls the file browsing endpoint for a completed run
- **THEN** the API server reads the run's workspace files from the PVC host path
- **AND** returns the directory listing or file contents

#### Scenario: Log endpoint reads from PVC path
- **WHEN** a client requests logs for a completed run that are stored on disk
- **THEN** the API server reads the log files from the PVC host path
- **AND** returns the log contents

### Requirement: All Docker images are current
All Docker images (controlplane, sidecar, hydration, agent-base) SHALL be built from the current source and deployed to the k0s cluster. No image SHALL be stale relative to the committed code.

#### Scenario: Sidecar has log tee and trace collection
- **WHEN** a run is created and the sidecar starts
- **THEN** the sidecar captures agent stdout/stderr via log tee
- **AND** the sidecar collects trace data

#### Scenario: Hydration generates required directories
- **WHEN** the hydration init container runs for a new run
- **THEN** it generates `.devcontainer` and `.aot` directories in the workspace

### Requirement: Logs stream in real-time
The Logs tab SHALL display real-time log output from a running agent.

#### Scenario: Live log streaming
- **WHEN** a user opens the Logs tab for a running agent
- **THEN** log lines appear as the agent produces them
- **AND** new lines are appended without requiring a page refresh

#### Scenario: Historical logs for completed run
- **WHEN** a user opens the Logs tab for a completed run
- **THEN** all log output from the run is displayed

### Requirement: Files are browsable after completion
The Files tab SHALL display the workspace file tree for a completed run.

#### Scenario: Browse workspace files
- **WHEN** a user opens the Files tab for a completed run
- **THEN** the workspace directory tree is displayed
- **AND** individual files can be viewed

### Requirement: Shell access works
Shell or Debug Run access SHALL work for runs.

#### Scenario: Shell into running agent
- **WHEN** a user opens a shell for a running agent
- **THEN** an interactive terminal session is established with the agent's container

#### Scenario: Debug completed run
- **WHEN** a user starts a Debug Run for a completed run
- **THEN** a new container is started with the run's workspace mounted
- **AND** the user gets an interactive terminal

### Requirement: Traces are visible
The traces view SHALL display trace data collected by the sidecar.

#### Scenario: View traces for completed run
- **WHEN** a user opens the traces view for a completed run
- **THEN** trace spans are displayed showing the agent's execution timeline

### Requirement: Single deploy command exists
A Taskfile task `deploy:all` SHALL exist that builds all Docker images, imports them into k0s, and restarts all deployments.

#### Scenario: Full redeploy
- **WHEN** a developer runs `task deploy:all`
- **THEN** all Docker images are rebuilt from current source
- **AND** all images are imported into k0s
- **AND** all deployments are rollout restarted
