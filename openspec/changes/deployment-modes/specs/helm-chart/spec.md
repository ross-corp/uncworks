## ADDED Requirements

### Requirement: OCI Helm chart published to GHCR on release
The CI pipeline SHALL publish the Helm chart as an OCI artifact to `ghcr.io/uncworks/charts/aot` on each tagged release.

#### Scenario: Chart published on tag
- **WHEN** a version tag (e.g., `v0.3.0`) is pushed to the repository
- **THEN** the CI pipeline runs `helm package` and `helm push oci://ghcr.io/uncworks/charts/aot` with the matching version

#### Scenario: Install from OCI registry
- **WHEN** `helm install uncworks oci://ghcr.io/uncworks/charts/aot --version 0.3.0` is run
- **THEN** Helm pulls the chart from GHCR and installs it successfully
