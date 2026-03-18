## Purpose

Define the container images that comprise the AOT platform, their contents, build process, and publication lifecycle.
## Requirements
### Requirement: Controlplane image contains all server binaries
The `aot-controlplane` image SHALL contain `apiserver`, `controller`, and `temporal-worker` binaries, selectable via container `command`.

#### Scenario: Run as apiserver
- **WHEN** a container uses `aot-controlplane` with `command: ["/usr/local/bin/apiserver"]`
- **THEN** it runs the API server

#### Scenario: Run as controller
- **WHEN** a container uses `aot-controlplane` with `command: ["/usr/local/bin/controller"]`
- **THEN** it runs the AOT controller

### Requirement: Web image serves built dashboard
The `aot-web` image SHALL contain the built web dashboard static files served by nginx.

#### Scenario: Serves dashboard
- **WHEN** a container runs `aot-web`
- **THEN** nginx serves the dashboard on port 3000

### Requirement: Images published to ghcr.io on release
All container images SHALL be built and published to `ghcr.io/uncworks/aot-*` on tagged releases.

#### Scenario: Tag triggers publish
- **WHEN** a git tag matching `v*` is pushed
- **THEN** GitHub Actions builds and pushes `aot-controlplane`, `aot-web`, `aot-init`, `aot-sidecar`, and `aot-agent` images tagged with the version

### Requirement: Dockerfiles live in docker/ directory
All Dockerfiles SHALL live in the `docker/` directory of the main repo.

#### Scenario: Controlplane Dockerfile exists
- **WHEN** `docker/Dockerfile.controlplane` is built with the repo root as context
- **THEN** it produces a working `aot-controlplane` image

#### Scenario: Web Dockerfile exists
- **WHEN** `docker/Dockerfile.web` is built with `web/` as context after `npx vite build`
- **THEN** it produces a working `aot-web` image

### Requirement: Sidecar image includes agent tooling
The `aot-sidecar` image SHALL contain the pi-coding-agent npm package, Node.js runtime, and the OpenSpec CLI, enabling the sidecar to run planning, execution, and verification agents with OpenSpec skill support.

#### Scenario: OpenSpec CLI available in sidecar
- **WHEN** a container runs `aot-sidecar`
- **THEN** the `openspec` command is available in the container's PATH
- **AND** `openspec --version` exits with code 0

