## Why

AOT currently has no installable distribution — deploying it requires cloning the repo, building from source, and manually applying raw YAML manifests. There's no upgrade path, no configuration knobs, and no published container images. We need a Helm chart and published images so that anyone with a Kubernetes cluster can install AOT with a single command.

## What Changes

- Create a Helm chart at `deploy/helm/aot/` containing all AOT components: CRD, controller, temporal-worker, API server, web dashboard, RBAC
- The chart has **no bundled dependencies** — Temporal and LLM endpoints are external, configured via `values.yaml`
- Add Dockerfiles for control plane (`aot-controlplane`) and web dashboard (`aot-web`) to the main repo under `docker/`
- Publish all container images to `ghcr.io/uncworks/aot-*`
- Add a GitHub Actions workflow for building and pushing images on release
- Add a GitHub Actions workflow for packaging and pushing the Helm chart to `ghcr.io/uncworks/charts/aot`
- Update `aot-local/` dev cluster to consume the Helm chart with a `dev-values.yaml` instead of raw manifests
- Add installation documentation: quick start, prerequisites, configuration reference, architecture overview

## Capabilities

### New Capabilities
- `helm-chart`: Helm chart structure, templates, values schema, and chart metadata for installing AOT on any Kubernetes cluster
- `container-images`: Dockerfiles, image build pipeline, and ghcr.io publishing for all AOT container images
- `install-docs`: User-facing documentation for installing, configuring, and upgrading AOT

### Modified Capabilities

None.

## Impact

- New directory: `deploy/helm/aot/` (chart)
- New files: `docker/Dockerfile.controlplane`, `docker/Dockerfile.web`
- New files: `.github/workflows/release-images.yaml`, `.github/workflows/release-chart.yaml`
- Modified: `aot-local/Taskfile.yml` — switch from raw manifests to `helm install` with dev-values
- Modified: repo root documentation
- Dependencies: Helm 3, ghcr.io OCI registry access
- No changes to application Go code or TypeScript code
