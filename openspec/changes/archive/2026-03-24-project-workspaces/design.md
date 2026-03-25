## Context

UNCWORKS currently treats projects as string labels on AgentRun CRDs. There is no persistent project state, no shared configuration between runs, and no way to browse or edit specs outside of a run. The hydration init container clones repos from GitHub, optionally runs devbox, and writes spec content passed inline. Workspaces are ephemeral PVCs deleted after a retention period.

The platform needs projects as first-class entities with persistent git-backed configuration, IDE access for human editing, and the ability to spawn runs from versioned specs. This requires a new CRD, an in-cluster git server, IDE pod infrastructure, and an SSH gateway.

## Goals / Non-Goals

**Goals:**
- Project CRD as the persistent entity owning repos, devbox config, defaults, and SSH keys
- soft-serve as in-cluster git server for project configuration repos
- Runs can reference a project and inherit its configuration
- IDE pods with code-server and SSH for editing project files
- Shared SSH gateway routing by project name
- Spec browser and editor in the web UI
- CLI for project management and local editing

**Non-Goals:**
- GitHub sync (two-way mirror between soft-serve and GitHub) — future phase
- Custom IDE extensions marketplace
- Multi-tenant access control (single-user for now)
- Replacing GitHub as the application code host — soft-serve is for project config only

## Decisions

### 1. soft-serve as the git server

Deploy [soft-serve](https://github.com/charmbracelet/soft-serve) (Charm.sh, Go, MIT) as a single pod in the `aot` namespace. It provides SSH + HTTP git transport, repository management via SSH admin commands, and minimal resource usage.

Each project gets a repo in soft-serve. The Project controller creates repos via SSH admin commands (`ssh soft-serve repo create {name}`) and pushes an initial scaffold commit.

Within the cluster, pods clone via `ssh://soft-serve.aot.svc:23231/{project-name}`. The SSH gateway exposes external access on NodePort 30022.

*Alternative considered:* gitgres (PostgreSQL-backed git). Rejected because it requires a custom binary on every client for the `gitgres://` protocol. The HTTP layer (omni_git) pulls in the entire omnigres stack. soft-serve is simpler, proven, and uses standard git protocols.

*Alternative considered:* Gitea. Rejected as overkill — we don't need a full forge (issues, PRs, CI). soft-serve is a minimal git server, which is all we need.

### 2. Shared SSH gateway (Go, `golang.org/x/crypto/ssh`)

A single Go binary that:
1. Listens on port 22 (NodePort 30022 externally)
2. Accepts SSH connections where username = project name
3. Looks up the Project CRD to get authorized SSH public keys
4. If the project's IDE pod isn't running, scales its Deployment to 1 and waits for ready
5. Proxies the SSH session to the IDE pod's sshd on port 2222

Uses `golang.org/x/crypto/ssh` for the SSH server and client. The gateway itself is stateless — all state is in Kubernetes (Project CRDs, Deployments).

For git operations (`git push`/`pull`), the gateway can detect the SSH command (`git-receive-pack`, `git-upload-pack`) and route to soft-serve instead of the IDE pod.

### 3. Project CRD owns a PVC, not just labels

The Project CRD controller creates a PVC (`project-{name}`) that persists the project config repo checkout and IDE state. This PVC is shared between the IDE pod and run pods (ReadWriteMany or sequential access).

Run pods that reference a project clone the project config from soft-serve into the workspace, but don't mount the project PVC directly — this avoids concurrent access issues. Instead, the hydration init container fetches from soft-serve (the authoritative source).

### 4. IDE pod architecture

A single-container pod (or two-container: code-server + sshd) built from a new `aot-ide` image:

```
Base: debian:bookworm-slim
+ Node.js (for code-server)
+ code-server (or openvscode-server)
+ openssh-server
+ neovim
+ devbox
+ openspec CLI
+ git, curl, build-essential
```

The pod mounts the project PVC at `/workspace` and clones the project's application repos into it on first start. The devbox shell is configured as the default login shell for SSH.

Idle detection: a sidecar or cron checks for active SSH sessions and VS Code WebSocket connections. After `idleTimeoutMinutes` of inactivity, the controller scales the Deployment to 0.

### 5. Run inheritance from Project

When an AgentRun has `projectRef` set, the controller:
1. Reads the referenced Project CRD
2. For each field in the run spec that is empty/zero, fills it from the project's config:
   - `repos` ← `project.spec.repos`
   - `modelTier` ← `project.spec.defaults.modelTier`
   - `manageModelTier` ← `project.spec.defaults.manageModelTier`
   - `implementModelTier` ← `project.spec.defaults.implementModelTier`
   - `ttlSeconds` ← `project.spec.defaults.ttlSeconds`
   - `orchestrationMode` ← `project.spec.defaults.orchestrationMode`
3. If `specRef` is set, fetches the spec content from the project's soft-serve repo (`git show HEAD:openspec/specs/{specRef}/spec.md`) and sets it as `specContent`
4. Passes the project's soft-serve repo URL in the workflow input so the hydration init container can clone it

### 6. Spec management via API

The apiserver exposes REST endpoints for reading/writing files in a project's soft-serve repo:
- `GET /api/v1/projects/{name}/files` — list files in the project config repo
- `GET /api/v1/projects/{name}/files/{path}` — read a file
- `PUT /api/v1/projects/{name}/files/{path}` — write a file (creates a commit in soft-serve)

The UI uses these endpoints to render a spec browser and Monaco editor. Commits are made server-side (the apiserver has SSH access to soft-serve).

## Risks / Trade-offs

- **[soft-serve stability]** — soft-serve is maintained by Charm.sh but is less battle-tested than Gitea. Mitigated by soft-serve's simplicity (single Go binary) and our low-volume usage (project config, not application code).
- **[SSH gateway security]** — the gateway accepts SSH connections from the internet. Mitigated by SSH key verification against Project CRD and no password auth. Rate limiting on failed attempts.
- **[IDE pod resource usage]** — code-server uses ~300MB RAM at idle. Mitigated by idle timeout and scale-to-zero. Multiple projects won't run simultaneously unless explicitly opened.
- **[PVC access patterns]** — ReadWriteOnce PVCs can't be shared between pods on different nodes. Mitigated by using the soft-serve repo as the authoritative source (not the PVC) and sequential pod access.
- **[Scope creep]** — this is the largest change proposed for UNCWORKS. Mitigated by phasing: Project CRD + soft-serve first, IDE pods second, full editing flow third.

## Open Questions

- Should the IDE image be based on a devcontainer base image (e.g., `mcr.microsoft.com/devcontainers/base`) to get feature support for free?
- Should we use openvscode-server instead of code-server? openvscode-server is closer to upstream VS Code but has weaker auth (token-in-URL only).
- Should the SSH gateway also handle mosh? Mosh uses UDP which complicates K8s routing. Alternative: install mosh-server in IDE pods and use `mosh --ssh="ssh -p 30022"` which establishes via SSH then switches to UDP directly to the pod.
