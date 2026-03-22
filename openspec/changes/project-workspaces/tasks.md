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

- [ ] 3.1 Create `internal/controller/project_controller.go` with reconciler that watches Project CRDs
- [ ] 3.2 On Project create: SSH to soft-serve and create repo (`ssh soft-serve repo create {name}`)
- [ ] 3.3 On Project create: push initial scaffold commit (devbox.json from spec.devbox.packages, openspec/openspec.yaml, .devcontainer/devcontainer.json)
- [ ] 3.4 On Project delete: delete the soft-serve repo (`ssh soft-serve repo delete {name}`)
- [ ] 3.5 Update status: set `configRepoReady`, `configRepoURL` after successful repo creation
- [ ] 3.6 Register Project controller in `cmd/controller/main.go`
- [ ] 3.7 Add RBAC rules for Project CRD in controller

### 4. Project-aware Run Creation

- [ ] 4.1 Update `BuildWorkflowInput` in `mapping.go` to resolve `ProjectRef` — read Project CRD, fill empty run fields from project defaults
- [ ] 4.2 Resolve `SpecRef` — SSH to soft-serve, run `git show HEAD:openspec/specs/{specRef}/spec.md`, set as `SpecContent`
- [ ] 4.3 Pass project config repo URL in `WorkflowInput` so hydration can clone it
- [ ] 4.4 Update hydration init container to clone project config repo from soft-serve when URL is provided
- [ ] 4.5 Add test for inheritance logic — project defaults fill empty run fields, explicit run fields override

### 5. Project API (ConnectRPC + REST)

- [ ] 5.1 Add `Project` message to proto, with `CreateProject`, `GetProject`, `ListProjects`, `UpdateProject`, `DeleteProject` RPCs
- [ ] 5.2 Implement Project CRUD handlers in apiserver (similar pattern to AgentRun handlers)
- [ ] 5.3 Add REST endpoints for project config repo files: `GET /api/v1/projects/{name}/files`, `GET /api/v1/projects/{name}/files/{path}`, `PUT /api/v1/projects/{name}/files/{path}`
- [ ] 5.4 File read: SSH to soft-serve, `git show HEAD:{path}` — return content
- [ ] 5.5 File write: clone repo to temp dir, write file, commit, push back to soft-serve

### 6. Project UI

- [ ] 6.1 Create `web/src/views/ProjectListView.tsx` — list projects with name, repo count, run count, last run, total cost
- [ ] 6.2 Create `web/src/views/ProjectDetailView.tsx` — project overview with config, spec browser, run list filtered by project
- [ ] 6.3 Add spec browser panel — tree view of `openspec/specs/` from project config repo, click to view in Monaco
- [ ] 6.4 Add "Run this spec" button in spec browser — creates run with `projectRef` + `specRef`
- [ ] 6.5 Add project creation form — name, repos, devbox packages, defaults
- [ ] 6.6 Add project settings page — edit devbox.json, defaults, SSH keys
- [ ] 6.7 Add routing: `/projects` list, `/projects/{name}` detail, `/projects/{name}/settings`
- [ ] 6.8 Update run list to show project link when `projectRef` is set
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
