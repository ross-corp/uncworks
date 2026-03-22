## Why

UNCWORKS is run-centric — you create a run, it clones repos, does work, and dies. Projects are just string labels on runs with no persistent state, no shared configuration, and no workspace continuity between runs. This means every run re-specifies repos, models, and dependencies from scratch. Specs written during one run are lost unless manually extracted. There is no way to iterate on a project's configuration, browse its specs, or connect an IDE to its workspace.

To become a real development platform — one that can replace Claude Code for managed agent orchestration — projects need to be first-class entities with persistent git-backed configuration, IDE access for human editing, and the ability to spawn runs from versioned specs.

## What Changes

### Project as a first-class CRD
- New `Project` CRD that owns repos, devbox config, default run settings, IDE config, and SSH keys
- Projects are the persistent entity; runs are ephemeral executions within them
- Runs can reference a project (`projectRef`) and inherit its configuration, or remain standalone (backwards compatible)
- Runs can reference a spec from the project's config repo (`specRef`) instead of inlining spec content

### In-cluster Git server (soft-serve)
- Deploy [soft-serve](https://github.com/charmbracelet/soft-serve) (Charm.sh) as the project config git server
- Each project gets a git repo in soft-serve containing `devbox.json`, `openspec/` specs, `.devcontainer/devcontainer.json`, and run templates
- Controller creates the soft-serve repo and pushes initial scaffold when a Project CRD is created
- Pods clone project config from `ssh://soft-serve.aot.svc:23231/{project-name}`

### IDE pods with code-server and SSH
- On-demand IDE pods per project with code-server (browser VS Code) and sshd
- Neovim, devbox shell, and openspec CLI pre-installed
- Idle timeout scales pod to 0; PVC persists across restarts
- "Open IDE" button in the project detail view opens browser IDE

### Shared SSH gateway
- Single SSH gateway pod that routes connections by username (username = project name)
- Verifies SSH keys against the Project CRD's `authorizedKeys`
- Wakes IDE pod if not running, then proxies the SSH session to it
- One NodePort (30022), unlimited projects
- Supports both interactive shell and git push/pull forwarding

### Project-aware hydration
- When a run has `projectRef`, the hydration init container clones the project config repo from soft-serve in addition to the application repos from GitHub
- Devbox packages from the project config are installed automatically
- Specs from the project repo are available to the agent at `/workspace/openspec/`

### Spec management in the UI
- Project detail view with spec browser (reads from project config repo)
- Monaco editor for viewing and editing specs, with commits pushed to soft-serve
- "Run this spec" button creates a run with `projectRef` + `specRef`
- devbox.json editor in project settings

### CLI support
- `uncworks project create/list/get/delete` — manage projects
- `uncworks project clone {name} [path]` — clone project config repo locally
- `uncworks push` — push local config changes back to soft-serve
- `uncworks run --project {name} --spec {spec-name}` — trigger a run from a spec
- `uncworks ssh {project-name}` — SSH into project workspace via gateway
- `uncworks ide open {project-name}` — open browser IDE

## Capabilities

### New Capabilities
- `project-crd`: Project as a Kubernetes CRD with repos, devbox config, defaults, IDE config, SSH keys, and lifecycle management
- `soft-serve-git`: In-cluster git server for project configuration repos, managed by the controller
- `project-runs`: Runs with `projectRef` that inherit project config; `specRef` to run a specific spec from the project repo
- `ide-pods`: On-demand IDE pods with code-server, sshd, neovim, devbox shell, and idle-timeout scale-to-zero
- `ssh-gateway`: Shared SSH gateway routing by project name, key verification against Project CRD, and IDE pod wake-on-connect
- `spec-management-ui`: Project detail view with spec browser, Monaco editor for editing, and "run this spec" workflow
- `project-cli`: CLI commands for project CRUD, config clone/push, run triggering, SSH, and IDE access

### Modified Capabilities
- None (existing run behavior is preserved; `projectRef` is optional)

## Impact

- **New CRD**: `Project` with controller, reconciler, and soft-serve repo lifecycle
- **New Deployment**: soft-serve (single pod, PVC for repo storage)
- **New Deployment**: SSH gateway (Go binary, single pod, NodePort service)
- **New Docker Image**: `aot-ide` (code-server + sshd + neovim + devbox + openspec CLI)
- **New Docker Image**: `aot-ssh-gateway` (Go SSH proxy)
- **Modified**: `internal/hydration/hydrator.go` — clone project config repo, apply project devbox
- **Modified**: `internal/controller/` — new Project reconciler alongside AgentRun reconciler
- **Modified**: `internal/temporal/workflow.go` — `WorkflowInput` gets `ProjectRef`, `SpecRef`; resolve from Project CRD
- **Modified**: `api/v1alpha1/types.go` — new Project types, `ProjectRef`/`SpecRef` on AgentRunSpec
- **Modified**: `deploy/crds/` — new `project-crd.yaml`
- **Modified**: `web/src/views/` — new ProjectListView, ProjectDetailView; run creation from project context
- **Modified**: `cmd/aot/` — new CLI subcommands for project management
- **Dependencies**: soft-serve (Go, MIT), code-server or openvscode-server (MIT), Go SSH library (`golang.org/x/crypto/ssh`)
