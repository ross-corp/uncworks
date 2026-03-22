## Phase 1: Project CRD + soft-serve + Project Runs

### 1. Project CRD Definition

- [x] 1.1 Create `api/v1alpha1/project_types.go` with `Project` and `ProjectSpec`/`ProjectStatus` structs (repos, devbox packages, defaults, IDE config, SSH keys, status fields)
- [x] 1.2 Create `deploy/crds/project-crd.yaml` with full OpenAPI schema matching the Go types
- [x] 1.3 Add `ProjectRef` and `SpecRef` fields to `AgentRunSpec` in `types.go` and `agentrun-crd.yaml`
- [x] 1.4 Generate deepcopy for Project types
- [x] 1.5 Add Project CRD contract test in `test/contract/` — verify Go types match YAML schema

### 2. Deploy soft-serve

- [x] 2.1 Create Helm template `deploy/helm/aot/templates/soft-serve.yaml` — Deployment, PVC, Service
- [x] 2.2 Add `soft-serve` values to `deploy/helm/aot/values.yaml`
- [x] 2.3 Add Taskfile command `k0s:soft-serve` to deploy soft-serve to the local cluster
- [x] 2.4 Verify soft-serve starts, repo create/clone/push/delete all work via SSH admin

### 3. Project Controller

- [x] 3.1 Create `internal/controller/project_controller.go` with reconciler that watches Project CRDs
- [x] 3.2 On Project create: SSH to soft-serve and create repo
- [x] 3.3 On Project create: push initial scaffold commit (devbox.json, openspec/, .devcontainer/)
- [x] 3.4 On Project delete: delete the soft-serve repo (finalizer)
- [x] 3.5 Update status: set `configRepoReady`, `configRepoURL` after successful repo creation
- [x] 3.6 Register Project controller in `cmd/controller/main.go`
- [x] 3.7 Add softserve.Client package with SSH admin and git operations

### 4. Project-aware Run Creation

- [x] 4.1 Create `ResolveProjectDefaults` in `project_resolve.go` — reads Project CRD, fills empty run fields
- [x] 4.2 Resolve `SpecRef` — reads spec content from soft-serve repo via client
- [x] 4.3 Returns project config repo URL for hydration
- [ ] 4.4 Wire `ResolveProjectDefaults` into AgentRun controller startup path
- [x] 4.5 Add 5 tests for inheritance logic (repos, models, TTL, override, standalone, label)

### 5. Project API (ConnectRPC + REST)

- [x] 5.1 REST Project CRUD: GET/POST /api/v1/projects, GET/DELETE /api/v1/projects/{name}
- [x] 5.2 REST file endpoints: GET files, GET files/{path}, PUT files/{path} with commit
- [x] 5.3 File read: clone from soft-serve, read file
- [x] 5.4 File write: clone, write, commit, push back to soft-serve
- [x] 5.5 Register project handler in apiserver main with soft-serve client

### 6. Project UI

- [x] 6.1 Create `ProjectListView.tsx` — list projects with create form
- [x] 6.2 Create `ProjectDetailView.tsx` — specs tab (file tree + Monaco editor) + settings tab
- [x] 6.3 Spec browser panel with file tree from soft-serve
- [x] 6.4 "Run this spec" button navigates to /new with project+spec params
- [x] 6.5 Project creation form in list view
- [x] 6.6 Settings tab shows repos, devbox packages, config repo URL
- [x] 6.7 Routes: `/projects` list, `/projects/{name}` detail
- [x] 6.8 "Projects" button in run list header
- [ ] 6.9 Update new run form to optionally select a project (auto-fills repos, model, etc.)

### 7. Verification

- [ ] 7.1 Create a project via the UI, verify soft-serve repo is created with scaffolded files
- [ ] 7.2 Create a run with `projectRef`, verify it inherits project repos and defaults
- [ ] 7.3 Create a run with `specRef`, verify it fetches spec content from the project repo
- [ ] 7.4 Edit a spec via the UI Monaco editor, verify the commit appears in soft-serve
- [ ] 7.5 Delete a project, verify soft-serve repo is cleaned up

## Phase 2: IDE Pods + SSH Gateway (future — not implemented in this pass)

- [ ] P2.1 Create `docker/Dockerfile.ide` — code-server + sshd + neovim + devbox + openspec CLI
- [ ] P2.2 Create SSH gateway Go binary (`cmd/ssh-gateway/main.go`)
- [ ] P2.3 IDE pod controller — create/scale Deployment on "Open IDE", idle timeout scale-to-zero
- [ ] P2.4 SSH gateway routing — username=project, key verification, wake-and-proxy
- [ ] P2.5 "Open IDE" button in project detail view
- [ ] P2.6 Deploy SSH gateway with Helm template + NodePort service
