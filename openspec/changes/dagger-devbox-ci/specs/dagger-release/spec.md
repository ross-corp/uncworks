## ADDED Requirements

### Requirement: Dagger module packages and pushes the Helm chart
The Dagger module SHALL expose `PackageChart` and `PushChart` functions for Helm chart operations.

#### Scenario: PackageChart packages the chart with a given version
- **WHEN** `PackageChart` is called with a version string and the source directory
- **THEN** it SHALL use the devbox base container (which includes `helm`)
- **THEN** it SHALL update `deploy/helm/aot/Chart.yaml` with the given version and appVersion
- **THEN** it SHALL run `helm package deploy/helm/aot`
- **THEN** it SHALL return the packaged `.tgz` file as a `*dagger.File`

#### Scenario: PushChart pushes the packaged chart to GHCR
- **WHEN** `PushChart` is called with a version string and a registry auth secret
- **THEN** it SHALL call `PackageChart` to produce the `.tgz`
- **THEN** it SHALL run `helm push <chart>.tgz oci://ghcr.io/uncworks/charts` using the devbox base container
- **THEN** the registry credentials SHALL be provided via the secret parameter

### Requirement: Dagger module exposes a single Release function
The Dagger module SHALL expose a `Release` function that composes all checks, image builds, and chart packaging into a single pipeline.

#### Scenario: Release runs all checks before building artifacts
- **WHEN** `Release` is called with source, version, and registry credentials
- **THEN** it SHALL first run `All` (build, lint, test, check) and fail immediately if any check fails
- **THEN** only after all checks pass SHALL it proceed to build images and package the chart

#### Scenario: Release pushes images and chart after checks pass
- **WHEN** all checks in the `Release` function pass
- **THEN** it SHALL call `PushImages` with the version and credentials
- **THEN** it SHALL call `PushChart` with the version and credentials
- **THEN** both pushes SHALL run in parallel

#### Scenario: Release fails if any stage fails
- **WHEN** any check, image build, or push operation fails during `Release`
- **THEN** the function SHALL return an error with the failing stage identified
- **THEN** no partial releases SHALL be published (images without chart or vice versa)

### Requirement: GitHub Actions workflows use Dagger for releases
The GitHub Actions release workflows SHALL be simplified to delegate to Dagger functions instead of implementing build/push logic directly.

#### Scenario: release-images.yaml delegates to Dagger
- **WHEN** a version tag is pushed
- **THEN** the `release-images.yaml` workflow SHALL call `dagger call push-images` with the version and GHCR credentials
- **THEN** the workflow SHALL NOT contain a build matrix or direct calls to `docker/build-push-action`

#### Scenario: release-chart.yaml delegates to Dagger
- **WHEN** a version tag is pushed
- **THEN** the `release-chart.yaml` workflow SHALL call `dagger call push-chart` with the version and GHCR credentials
- **THEN** the workflow SHALL NOT install Helm or run `helm package`/`helm push` directly

#### Scenario: Unified release workflow replaces separate workflows
- **WHEN** the release pipeline is mature
- **THEN** `release-images.yaml` and `release-chart.yaml` MAY be merged into a single workflow that calls `dagger call release`
- **THEN** the single workflow SHALL pass the tag version and GHCR token as Dagger arguments
