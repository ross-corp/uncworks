## ADDED Requirements

### Requirement: Local image build
The system SHALL provide a Taskfile target `docker:build` that builds all 3 agent pod images (agent-base, hydration, sidecar) locally using Docker.

#### Scenario: Build all images
- **WHEN** a developer runs `task docker:build`
- **THEN** all 3 Docker images are built and tagged as `aot-agent:local`, `aot-init:local`, `aot-sidecar:local`

### Requirement: k0s image import
The system SHALL provide a Taskfile target `k0s:images` that exports Docker images and imports them into the k0s containerd runtime.

#### Scenario: Import images into k0s
- **WHEN** a developer runs `task k0s:images` (requires sudo)
- **THEN** all 3 images are available in k0s containerd and agent pods can use them with `imagePullPolicy: Never`

### Requirement: Configurable image names
The temporal-worker SHALL support environment variables `AOT_AGENT_IMAGE`, `AOT_SIDECAR_IMAGE`, and `AOT_INIT_IMAGE` to override the default image references.

#### Scenario: Override agent image
- **WHEN** the temporal-worker is started with `AOT_AGENT_IMAGE=aot-agent:local`
- **THEN** agent pods are created using `aot-agent:local` instead of `ghcr.io/uncworks/aot-agent:latest`

#### Scenario: ImagePullPolicy for local images
- **WHEN** the image name does not contain a registry prefix (no `/`)
- **THEN** the pod spec SHALL set `imagePullPolicy: Never`
