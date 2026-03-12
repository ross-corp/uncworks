## Context

AOT is deployed via raw YAML manifests in an ad-hoc `aot-local/` folder. Container images are built locally and imported into k0s via `docker save | k0s ctr images import`. There are no published images, no versioned releases, and no standard installation method. The existing raw manifests (namespace, deployments, services, RBAC) in `aot-local/manifests/` serve as a working reference for what the Helm chart needs to produce.

Temporal and Ollama are external dependencies — the `aot-local/` dev setup installs them separately, and the Helm chart should not bundle them.

## Goals / Non-Goals

**Goals:**
- Single `helm install` command to deploy AOT to any Kubernetes cluster
- All configuration via `values.yaml` — no forking or patching required
- Published, versioned container images on ghcr.io
- Published Helm chart as OCI artifact on ghcr.io
- Dev cluster (`aot-local/`) consumes the same Helm chart with local overrides
- Clear documentation for install, configure, upgrade

**Non-Goals:**
- Helm chart for Temporal (users install it themselves or use Temporal Cloud)
- Helm chart for Ollama or any LLM provider
- Multi-tenancy or namespace-per-user isolation
- HA / multi-replica production hardening (future work)
- Helm chart testing framework (e.g., helm-unittest) — future work

## Decisions

1. **Single controlplane image with multiple entrypoints**: One Docker image (`aot-controlplane`) containing `apiserver`, `controller`, and `temporal-worker` binaries. Each Deployment selects its binary via `command:`. This halves the number of images to build/push/pull versus 3 separate images. The tradeoff is slightly larger image size per component, but the Go binaries are small (~15MB each) and share the same base layer.

   *Alternative*: Separate images per component. Rejected because it triples the CI matrix and registry storage with minimal benefit at this scale.

2. **OCI registry for both chart and images**: Use `ghcr.io/uncworks/charts/aot` for the Helm chart (OCI format, supported since Helm 3.8) and `ghcr.io/uncworks/aot-*` for images. Single registry, single auth, tied to the GitHub org.

   *Alternative*: ChartMuseum or Artifactory. Rejected — unnecessary infrastructure when ghcr.io supports OCI natively.

3. **CRD managed by the chart**: The AgentRun CRD is installed as a chart template (not a separate `kubectl apply`). This keeps the install self-contained. Helm's `--skip-crds` flag is available for users who manage CRDs separately.

   *Alternative*: CRD as a separate Helm chart or pre-install step. Rejected for simplicity — one chart, one install.

4. **values.yaml schema**: Flat, minimal, no deep nesting. Key sections:
   - `temporal.host` (required) — Temporal Frontend address
   - `llm.baseUrl`, `llm.apiKey` — LLM endpoint config, injected as env vars
   - `images.*` — image references with `tag` and `pullPolicy`
   - `controller`, `worker`, `apiserver`, `web` — per-component `replicas`, `resources`, `nodeSelector`
   - `web.service.type` — `ClusterIP` (default) or `NodePort`/`LoadBalancer`

5. **GitHub Actions for CI/CD**: Two workflows:
   - `release-images.yaml`: Triggered on tag push (`v*`). Builds and pushes all images to ghcr.io with the tag version.
   - `release-chart.yaml`: Triggered on tag push (`v*`). Packages Helm chart and pushes to ghcr.io as OCI artifact.

   Both use `GITHUB_TOKEN` for auth — no additional secrets needed.

6. **Dev cluster consumes the chart**: `aot-local/Taskfile.yml` switches from `kubectl apply -f manifests/` to `helm install aot ../uncworks/deploy/helm/aot -f dev-values.yaml`. The raw manifests in `aot-local/manifests/` are removed. `dev-values.yaml` overrides images to `*:local` with `pullPolicy: Never`.

## Risks / Trade-offs

- **CRD upgrade is tricky with Helm**: Helm doesn't update CRDs on `helm upgrade` by design. If the CRD schema changes, users need to `kubectl apply` it manually. Mitigated by documenting this in upgrade instructions, and by keeping the CRD stable.
- **Image tag drift**: If users pin `latest` tag, they get unpredictable behavior. Mitigated by defaulting `values.yaml` to a specific version tag and documenting that `latest` is for dev only.
- **ghcr.io rate limits**: Free tier has generous limits but could hit issues with many concurrent pulls. Not a concern at current scale.
- **Controlplane image size**: Bundling 3 binaries makes the image ~50MB larger than individual images. Acceptable — still under 100MB total.

## Open Questions

- Should we support `helm test` hooks for post-install verification? (e.g., a Job that checks Temporal connectivity)
- Should the web dashboard be optional (`web.enabled: true/false`)? Leaning yes.
