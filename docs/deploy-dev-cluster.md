# Deploying AOT to a Dev Cluster (tristinfra)

This guide migrates AOT from `aot-local` (k0s on your laptop with local images) to a proper dev cluster using the same infrastructure patterns as the homelab repo.

## Architecture

```
tristinfra cluster (k0s single-node)
├── ArgoCD            (GitOps, app-of-apps)
├── 1Password Operator (secrets → K8s Secrets)
├── Tailscale Operator (zero-trust ingress)
├── local-path-prov.  (PVCs for agent workspaces)
│
├── aot-controlplane  (apiserver + controller + worker)
├── aot-web           (dashboard)
├── temporal           (workflow engine)
├── ollama             (local LLM: qwen3:8b)
├── litellm            (LLM proxy/routing)
└── postgresql         (future: knowledge system)
```

## Prerequisites

- A machine running Ubuntu Server (22.04+) with 16GB+ RAM, 100GB+ disk
- Tailscale account with admin access
- 1Password account with Connect Server credentials
- GitHub account with GHCR access
- Domain: `*.{tailnet}.ts.net` (auto-provisioned by Tailscale)

## 1. Create the tristinfra Repo

Clone the homelab structure:

```
tristinfra/
├── ansible/
│   ├── 00-system/      # OS setup (same as homelab)
│   ├── 01-k0s/         # k0s cluster setup (same as homelab)
│   └── 02-k8s/         # K8s apps (modified for AOT)
├── manifests/
│   ├── app-of-apps.yaml
│   └── apps/           # ArgoCD Application resources
├── tailscale/
│   └── acl.jsonc
├── scripts/
├── .github/workflows/
├── Taskfile.yml
├── shell.nix
└── README.md
```

The `00-system/` and `01-k0s/` playbooks are identical to homelab. Only `02-k8s/` and `manifests/` change.

## 2. Provision the Machine

### 2a. Base OS (Ansible 00-system)

Same as homelab — installs Docker, Helm, essential tools, power management, SSH config:

```bash
task ansible-up -- 00-system
```

### 2b. k0s Cluster (Ansible 01-k0s)

Same as homelab — installs k0s with Calico CNI, creates kubeconfig:

```bash
task ansible-up -- 01-k0s
```

Verify:
```bash
kubectl get nodes
# NAME         STATUS   ROLES           AGE   VERSION
# <hostname>   Ready    control-plane   1m    v1.32.x+k0s
```

### 2c. Bootstrap K8s Apps (Ansible 02-k8s)

Modified from homelab to include AOT dependencies:

```bash
task ansible-up -- 02-k8s
```

This playbook:
1. Installs ArgoCD via Helm
2. Creates 1Password secrets from `op` CLI
3. Configures ArgoCD with GitHub credentials
4. Applies the app-of-apps root application
5. Creates GHCR pull secret for AOT images

## 3. 1Password Items

Create these items in your 1Password vault (tag: `tristinfra`):

| Item Name | Fields | Purpose |
|-----------|--------|---------|
| `tristinfra.op.credentials-file` | `credential` (file) | 1Password Connect credentials JSON |
| `tristinfra.op.access-token` | `credential` | 1Password Operator token |
| `tristinfra.argocd` | `password` | ArgoCD admin password |
| `tristinfra.github.pat` | `credential` | GitHub PAT (repo + packages read) |
| `tristinfra.aot.api-key` | `credential` | AOT API key for bearer auth |
| `tristinfra.aot.openrouter-key` | `credential` | OpenRouter API key (optional, for cloud LLM) |
| `tristinfra.aot.github-webhook-secret` | `credential` | GitHub webhook HMAC secret |
| `tristinfra.tailscale.oauth-client` | `clientId`, `clientSecret` | Tailscale OAuth for K8s auth |

The 1Password Operator auto-syncs these to K8s Secrets via `OnePasswordItem` CRDs.

## 4. ArgoCD Applications

### 4a. App-of-Apps Root

`manifests/app-of-apps.yaml`:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: app-of-apps
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "-2"
spec:
  project: default
  source:
    repoURL: https://github.com/<org>/tristinfra.git
    path: manifests/apps
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 3
```

### 4b. AOT Application

`manifests/apps/aot.yaml`:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: aot
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "4"
spec:
  project: default
  source:
    repoURL: https://github.com/ross-corp/uncworks.git
    path: deploy/helm/aot
    targetRevision: main
    helm:
      valueFiles:
        - values.yaml
      values: |
        temporal:
          host: temporal.aot.svc.cluster.local:7233

        llm:
          baseUrl: http://litellm.aot.svc.cluster.local:4000

        images:
          controlplane:
            repository: ghcr.io/uncworks/aot-controlplane
            tag: latest
          init:
            repository: ghcr.io/uncworks/aot-init
            tag: latest
          sidecar:
            repository: ghcr.io/uncworks/aot-sidecar
            tag: latest
          agent:
            repository: ghcr.io/uncworks/aot-agent
            tag: latest
          web:
            repository: ghcr.io/uncworks/aot-web
            tag: latest

        apiserver:
          apiKey: ""          # Set via 1Password OnePasswordItem
          allowedOrigins: "https://aot.<tailnet>.ts.net"
          service:
            type: ClusterIP   # Tailscale handles ingress

        web:
          service:
            type: ClusterIP

        pipeline:
          maxRetries: 3
          planTimeout: 300
  destination:
    server: https://kubernetes.default.svc
    namespace: aot
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 4c. Infrastructure Applications

Each gets its own file in `manifests/apps/`:

**`temporal.yaml`** — Temporal server:
```yaml
# Use the official Temporal Helm chart or a simple Deployment
# with SQLite backend (same as aot-local for dev)
# For production: use PostgreSQL backend
```

**`ollama.yaml`** — Local LLM inference:
```yaml
# Helm chart: ollama/ollama
# Pull models on init: qwen3:8b, llama3.1:8b
# GPU passthrough if available (nvidia device plugin)
# Storage: hostPath for model cache (~15GB)
```

**`litellm.yaml`** — LLM proxy:
```yaml
# Deployment with ConfigMap for litellm-config.yaml
# Points to ollama ClusterIP + OpenRouter (with API key from 1Password)
```

**`1password-connect.yaml`**, **`1password-operator.yaml`**, **`tailscale-operator.yaml`** — Same as homelab.

## 5. Tailscale Ingress

`manifests/aot-ingress.yaml`:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aot-web
  namespace: aot
spec:
  ingressClassName: tailscale
  rules:
    - host: aot
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: aot-web
                port:
                  number: 3000
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aot-api
  namespace: aot
spec:
  ingressClassName: tailscale
  rules:
    - host: aot-api
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: aot-apiserver
                port:
                  number: 50055
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: temporal-ui
  namespace: aot
spec:
  ingressClassName: tailscale
  rules:
    - host: temporal
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: temporal-ui
                port:
                  number: 8233
```

This creates:
- `aot.<tailnet>.ts.net` → web dashboard
- `aot-api.<tailnet>.ts.net` → gRPC/Connect API
- `temporal.<tailnet>.ts.net` → Temporal UI

## 6. AOT Secrets via 1Password

`manifests/aot-secrets.yaml`:
```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: aot-api-key
  namespace: aot
spec:
  itemPath: "vaults/tristinfra/items/tristinfra.aot.api-key"
---
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: aot-openrouter-key
  namespace: aot
spec:
  itemPath: "vaults/tristinfra/items/tristinfra.aot.openrouter-key"
---
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: aot-github-webhook-secret
  namespace: aot
spec:
  itemPath: "vaults/tristinfra/items/tristinfra.aot.github-webhook-secret"
```

Then reference in the Helm values:
```yaml
apiserver:
  apiKey: <from secret aot-api-key>
```

Or mount as env vars in the Helm templates using `envFrom: secretRef`.

## 7. CI/CD: Build & Push Images

Add to `uncworks/.github/workflows/release.yml`:

```yaml
name: Build and Push Images
on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push controlplane
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile.controlplane
          push: true
          tags: |
            ghcr.io/uncworks/aot-controlplane:${{ github.ref_name }}
            ghcr.io/uncworks/aot-controlplane:latest

      - name: Build and push sidecar
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile.sidecar
          push: true
          tags: |
            ghcr.io/uncworks/aot-sidecar:${{ github.ref_name }}
            ghcr.io/uncworks/aot-sidecar:latest

      - name: Build and push init
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile.hydration
          push: true
          tags: |
            ghcr.io/uncworks/aot-init:${{ github.ref_name }}
            ghcr.io/uncworks/aot-init:latest

      - name: Build and push agent
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/Dockerfile.agent-base
          push: true
          tags: |
            ghcr.io/uncworks/aot-agent:${{ github.ref_name }}
            ghcr.io/uncworks/aot-agent:latest

      - name: Build web
        run: cd web && npm ci && npx vite build

      - name: Build and push web
        uses: docker/build-push-action@v5
        with:
          context: web
          file: docker/Dockerfile.web
          push: true
          tags: |
            ghcr.io/uncworks/aot-web:${{ github.ref_name }}
            ghcr.io/uncworks/aot-web:latest
```

Tag a release → images pushed to GHCR → ArgoCD auto-syncs.

## 8. Deployment Checklist

```
[ ] Machine provisioned with Ubuntu Server
[ ] Tailscale installed and authenticated
[ ] tristinfra repo created with ansible/ and manifests/
[ ] 1Password items created (8 items)
[ ] Ansible 00-system playbook run
[ ] Ansible 01-k0s playbook run
[ ] Ansible 02-k8s playbook run (ArgoCD + 1Password + Tailscale)
[ ] app-of-apps applied
[ ] ArgoCD syncs all applications
[ ] Temporal pod running
[ ] Ollama pod running with qwen3:8b pulled
[ ] LiteLLM pod running with correct config
[ ] AOT controlplane pods running (apiserver, controller, worker)
[ ] AOT web pod running
[ ] Tailscale ingress accessible: aot.<tailnet>.ts.net
[ ] Health check: curl https://aot-api.<tailnet>.ts.net/healthz
[ ] Readiness: curl https://aot-api.<tailnet>.ts.net/readyz
[ ] Create test run via UI
[ ] Verify agent completes successfully
```

## 9. Environment Comparison

| Setting | aot-local | tristinfra |
|---------|-----------|------------|
| `AOT_API_KEY` | unset | 1Password → K8s Secret |
| `AOT_ALLOWED_ORIGINS` | `localhost:3000,5173` | `https://aot.<tailnet>.ts.net` |
| `GITHUB_WEBHOOK_SECRET` | unset | 1Password → K8s Secret |
| `LITELLM_BASE_URL` | `http://litellm:4000` | `http://litellm.aot.svc.cluster.local:4000` |
| `TEMPORAL_HOST` | `temporal:7233` | `temporal.aot.svc.cluster.local:7233` |
| Image pull | `k0s ctr import` (local) | `ghcr.io/uncworks/*` (registry) |
| Ingress | NodePort 30300 | Tailscale `aot.<tailnet>.ts.net` |
| TLS | None | Auto (Tailscale) |
| Auth | None | Bearer token + Tailscale OIDC |

## 10. GPU Acceleration (Optional)

If the machine has an NVIDIA GPU:

```bash
# Install NVIDIA container toolkit
sudo apt install nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=containerd
sudo systemctl restart containerd

# Deploy NVIDIA device plugin
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.1/nvidia-device-plugin.yml

# Update Ollama deployment with GPU resources
# In manifests/ollama/deployment.yaml:
resources:
  limits:
    nvidia.com/gpu: 1
```

With GPU: qwen3:8b inference drops from ~10s/token to ~50ms/token.
