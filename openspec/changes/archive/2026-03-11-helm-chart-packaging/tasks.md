## 1. Dockerfiles

- [x] 1.1 Move `aot-local/Dockerfile.controlplane` to `docker/Dockerfile.controlplane` (builds apiserver + controller + temporal-worker)
- [x] 1.2 Move `aot-local/Dockerfile.web` to `docker/Dockerfile.web` (nginx serving built dashboard)
- [x] 1.3 Verify both Dockerfiles build successfully
- [x] 1.4 Update `Taskfile.yml` `docker:build` to include controlplane and web images

## 2. Helm Chart Structure

- [x] 2.1 Create `deploy/helm/aot/Chart.yaml` with chart metadata (name, version, appVersion, description)
- [x] 2.2 Create `deploy/helm/aot/values.yaml` with all configurable values and defaults
- [x] 2.3 Create `deploy/helm/aot/templates/_helpers.tpl` with common template helpers (labels, names, selectors)
- [x] 2.4 Create `deploy/helm/aot/templates/crd.yaml` — AgentRun CRD
- [x] 2.5 Create `deploy/helm/aot/templates/rbac.yaml` — ServiceAccount, ClusterRole, ClusterRoleBinding
- [x] 2.6 Create `deploy/helm/aot/templates/controller.yaml` — Controller Deployment
- [x] 2.7 Create `deploy/helm/aot/templates/worker.yaml` — Temporal Worker Deployment
- [x] 2.8 Create `deploy/helm/aot/templates/apiserver.yaml` — API Server Deployment + Service
- [x] 2.9 Create `deploy/helm/aot/templates/web.yaml` — Web Dashboard Deployment + Service + ConfigMap (nginx proxy config)
- [x] 2.10 Add `temporal.host` required value validation via `templates/NOTES.txt` or `_helpers.tpl`
- [x] 2.11 Verify `helm template` renders all manifests correctly
- [x] 2.12 Verify `helm install` works against the local k0s cluster

## 3. CI/CD Workflows

- [x] 3.1 Create `.github/workflows/release-images.yaml` — build and push all images on `v*` tag
- [x] 3.2 Create `.github/workflows/release-chart.yaml` — package and push Helm chart on `v*` tag
- [x] 3.3 Verify workflows reference correct Dockerfiles and chart paths

## 4. Dev Cluster Update

- [x] 4.1 Replace `aot-local/manifests/` raw YAMLs with `helm install` using a `dev-values.yaml`
- [x] 4.2 Create `aot-local/dev-values.yaml` with local image overrides and `pullPolicy: Never`
- [x] 4.3 Update `aot-local/Taskfile.yml` to use `helm install/upgrade/uninstall` instead of `kubectl apply/delete`
- [x] 4.4 Verify `task up` and `task down` work with Helm-based deployment

## 5. Documentation

- [x] 5.1 Create `deploy/helm/aot/README.md` with quick start, prerequisites, and configuration reference
- [x] 5.2 Add architecture diagram showing component relationships and external dependencies
- [x] 5.3 Add upgrade instructions with CRD caveats
- [x] 5.4 Update root `README.md` or relevant docs to link to Helm chart installation
