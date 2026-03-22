## ADDED Requirements

### Requirement: IDE Pod Provisioning
The system SHALL create one IDE pod per project. The IDE pod SHALL run code-server on port 8080 and sshd on port 2222. The pod SHALL have Neovim and devbox shell pre-installed in its base image.

#### Scenario: IDE pod starts with required services
- **WHEN** an IDE pod is created for a project
- **THEN** code-server SHALL be accessible on port 8080 and sshd SHALL accept connections on port 2222 within the cluster

#### Scenario: Neovim and devbox are available in the pod
- **WHEN** a user opens a shell in the IDE pod
- **THEN** the `nvim` and `devbox` commands SHALL be available on the PATH

### Requirement: Project PVC Mount
Each IDE pod SHALL mount the project's PersistentVolumeClaim at a workspace directory. The project's config repo SHALL be cloned into this workspace on first start. The `devcontainer.json` in the project's config repo SHALL configure code-server extensions.

#### Scenario: Workspace contains project config repo
- **WHEN** an IDE pod starts for the first time for a project
- **THEN** the project's config repo SHALL be cloned into the workspace directory on the mounted PVC

#### Scenario: Code-server extensions installed from devcontainer.json
- **WHEN** the project's `.devcontainer/devcontainer.json` lists extensions `["ms-python.python", "golang.go"]`
- **THEN** code-server SHALL install those extensions on pod startup

### Requirement: Idle Timeout and Scale-to-Zero
The IDE pod SHALL scale to zero replicas after a configurable idle timeout period with no active SSH sessions or HTTP connections. The system SHALL track last-activity timestamps to determine idle state.

#### Scenario: Pod scales down after idle timeout
- **WHEN** an IDE pod has had no SSH sessions and no HTTP connections for longer than the configured idle timeout
- **THEN** the system SHALL scale the pod's replica count to zero

### Requirement: Wake on Access
When a user triggers "Open IDE" or connects via SSH to a scaled-down IDE pod, the system SHALL scale the pod back to one replica and wait for it to become ready before routing the connection.

#### Scenario: Open IDE wakes a scaled-down pod
- **WHEN** a user clicks "Open IDE" for a project whose IDE pod is scaled to zero
- **THEN** the system SHALL scale the pod to one replica and redirect the user to code-server once the pod is ready
