## ADDED Requirements

### Requirement: Dagger module builds all Docker images
The Dagger module SHALL expose a `BuildImages` function that builds all 5 Docker images using the existing Dockerfiles via Dagger's `DockerBuild` API.

#### Scenario: BuildImages builds all images
- **WHEN** `BuildImages` is called with the source directory
- **THEN** it SHALL build all 5 images in parallel:
  - `aot-controlplane` from `docker/Dockerfile.controlplane` with context `.`
  - `aot-init` from `docker/Dockerfile.hydration` with context `.`
  - `aot-sidecar` from `docker/Dockerfile.sidecar` with context `.`
  - `aot-agent` from `docker/Dockerfile.agent-base` with context `.`
  - `aot-web` from `docker/Dockerfile.web` with context `web/`
- **THEN** each image SHALL be returned as a `*dagger.Container`

#### Scenario: BuildImages uses existing Dockerfiles unchanged
- **WHEN** `BuildImages` is called
- **THEN** it SHALL reference the Dockerfiles at their current paths under `docker/`
- **THEN** no Dockerfiles SHALL be modified or duplicated for Dagger compatibility

#### Scenario: Individual image build function
- **WHEN** `BuildImage` is called with an image name (e.g., `aot-controlplane`)
- **THEN** it SHALL build only that single image using the corresponding Dockerfile and context
- **THEN** it SHALL return the built container

### Requirement: Dagger module pushes images to GHCR
The Dagger module SHALL expose a `PushImages` function that pushes all built images to GHCR with appropriate tags.

#### Scenario: PushImages pushes all images with version tags
- **WHEN** `PushImages` is called with a version string and a registry auth secret
- **THEN** it SHALL build all 5 images via `BuildImages`
- **THEN** it SHALL push each image to `ghcr.io/uncworks/<image-name>` with tags for the full semver, major.minor, and git SHA

#### Scenario: PushImages requires authentication
- **WHEN** `PushImages` is called
- **THEN** it SHALL accept a `registryAuth` secret parameter for GHCR authentication
- **THEN** the secret SHALL be passed via Dagger's `WithRegistryAuth` method, never written to disk or logs

#### Scenario: PushImages fails fast on auth error
- **WHEN** `PushImages` is called with invalid or missing registry credentials
- **THEN** the function SHALL fail with a clear error before attempting any pushes
