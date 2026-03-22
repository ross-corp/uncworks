## Architecture

### Token Flow — Security Boundary

```
                    TRUSTED ZONE                          UNTRUSTED ZONE
                    (has credentials)                     (no credentials)

  ┌─────────────────────────────────┐     ┌─────────────────────────────┐
  │         Worker Pod              │     │        Agent Pod            │
  │                                 │     │                             │
  │  GITHUB_TOKEN (from Secret)     │     │  Init Container             │
  │  ├── PushChanges activity       │     │  ├── GITHUB_TOKEN (clone)   │
  │  │   └── git commit+push       │     │  └── git clone --bare       │
  │  │       (via PVC host path)    │     │                             │
  │  ├── CreatePR activity          │     │  Agent Container            │
  │  │   └── GitHub REST API        │     │  ├── NO GITHUB_TOKEN        │
  │  └── Webhook handler            │     │  ├── Runs pi (LLM agent)    │
  │      └── Fetch file content     │     │  └── Writes to workspace    │
  │                                 │     │                             │
  │  API Server Pod                 │     │  Sidecar Container          │
  │  ├── GITHUB_TOKEN (from Secret) │     │  ├── NO GITHUB_TOKEN        │
  │  └── Spec push/pull             │     │  ├── ExecCommand RPC        │
  │                                 │     │  └── Trace span capture     │
  └─────────────────────────────────┘     └─────────────────────────────┘
```

Key principle: **the agent process (LLM) never has access to GitHub credentials.** The init container uses the token for clone and then exits. Git push happens worker-side via the PVC.

### GitHubTokenProvider Interface

```go
package github

import "context"

// TokenProvider returns a valid GitHub API token.
type TokenProvider interface {
    Token(ctx context.Context) (string, error)
}

// PATProvider returns a static personal access token.
type PATProvider struct {
    token string
}

func NewPATProvider(token string) *PATProvider {
    return &PATProvider{token: token}
}

func (p *PATProvider) Token(_ context.Context) (string, error) {
    if p.token == "" {
        return "", fmt.Errorf("GITHUB_TOKEN not configured")
    }
    return p.token, nil
}

// AppProvider mints installation tokens from a GitHub App.
// Stub — full implementation deferred to multi-user milestone.
type AppProvider struct {
    appID          int64
    installationID int64
    privateKey     []byte
    mu             sync.Mutex
    cached         string
    expiresAt      time.Time
}

func (a *AppProvider) Token(ctx context.Context) (string, error) {
    a.mu.Lock()
    defer a.mu.Unlock()
    if time.Now().Before(a.expiresAt) {
        return a.cached, nil
    }
    // TODO: JWT → installation token exchange
    return "", fmt.Errorf("GitHub App token provider not yet implemented")
}
```

### PushChanges — Worker-Side Git

Current (broken): `execInSidecar("git push")` — no auth in pod.

New: Worker reads the workspace from PVC host path and does git operations locally:

```go
func (a *Activities) PushChanges(ctx context.Context, input PushChangesInput) (*PushChangesOutput, error) {
    // Get the PVC host path (same as file/trace handlers)
    workDir := getPVCHostPath(input.AgentRunName)
    repoDir := filepath.Join(workDir, "<reponame>")

    // Get token from provider
    token, err := a.GitHubProvider.Token(ctx)

    // Configure git credentials for this push
    // Use GIT_ASKPASS with a script that returns the token
    // OR use credential.helper with store
    cmd := exec.Command("git", "push", "origin", input.BranchName)
    cmd.Dir = repoDir
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("GIT_ASKPASS=%s", askPassScript(token)),
        // OR: use the token in the URL
        // git remote set-url origin https://x-access-token:<token>@github.com/org/repo.git
    )
}
```

The simplest secure approach: temporarily set the remote URL with the token embedded, push, then reset:

```bash
git remote set-url origin https://x-access-token:${TOKEN}@github.com/org/repo.git
git push origin aot/ar-xyz123
git remote set-url origin https://github.com/org/repo.git  # restore
```

This works for both PATs and GitHub App installation tokens.

### Init Container — Scoped Token Injection

The init container (hydration) needs the token for private repo cloning. It's set as an env var ONLY on the init container, not on the agent or sidecar containers:

```go
// In BuildAgentPod — init container gets GITHUB_TOKEN
InitContainers: []corev1.Container{{
    Name: "hydration",
    Env: append(envVars, corev1.EnvVar{
        Name: "GITHUB_TOKEN",
        ValueFrom: &corev1.EnvVarSource{
            SecretKeyRef: &corev1.SecretKeySelector{
                LocalObjectReference: corev1.LocalObjectReference{Name: "github-token"},
                Key: "token",
            },
        },
    }),
}},
// Agent and sidecar containers do NOT get GITHUB_TOKEN
```

The hydration code uses it:
```go
func (h *Hydrator) cloneRepo(ctx context.Context, repoURL, bareDir string) error {
    // Inject token into URL for auth
    token := os.Getenv("GITHUB_TOKEN")
    if token != "" {
        repoURL = injectTokenInURL(repoURL, token)
    }
    args := []string{"clone", "--bare", repoURL, bareDir}
    ...
}
```

### Helm Secret Configuration

```yaml
# values.yaml
github:
  # Name of the k8s Secret containing the GitHub token
  # Secret must have key "token"
  tokenSecretName: ""  # e.g., "github-token"

# dev-values.yaml
github:
  tokenSecretName: "github-token"
```

Create the secret manually:
```bash
kubectl -n aot create secret generic github-token --from-literal=token=ghp_xxx
```

### Future: GitHub App + Multi-User

When we add GitHub App support:
1. `AppProvider` mints installation tokens (1hr expiry, auto-refresh)
2. The app is installed on specific repos — scoped access
3. For multi-user: each user authenticates via GitHub OAuth
4. User's OAuth token used for user-scoped operations (PRs show as their identity)
5. Installation token used for bot operations (webhooks, CI)
