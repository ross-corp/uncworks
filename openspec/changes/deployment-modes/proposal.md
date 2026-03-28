## Why

UNCWORKS is Kubernetes-native but has no cohesive story for how users actually install and run it — locally or in the cloud. The current local setup requires k0s (Linux-only, sudo, manual steps) and there is no packaged release artifact for end-users. This change delivers three concrete deployment targets: a polished remote Helm install, a cross-platform CLI setup wizard for local "bring your own Kubernetes" use, a TUI client for headless/remote operation, and a native macOS desktop app.

## What Changes

- **`uncworks` CLI binary** — new top-level command that wraps setup, status, and client operations. Replaces ad-hoc `task k0s:*` scripts as the user-facing interface.
- **Setup wizard** (`uncworks setup`) — interactive TUI that detects local Kubernetes contexts (Docker Desktop, OrbStack, Rancher Desktop, Colima, k3d, kind, minikube), validates resources, and deploys the Helm chart with environment-appropriate values.
- **Local Helm values profile** (`values.local.yaml`) — lightweight values preset: Ollama disabled by default, reduced resource requests, NodePort exposure, configurable storage class.
- **TUI client** (`uncworks tui`) — Bubble Tea terminal UI for monitoring active runs, streaming logs, and submitting new runs. Connects via gRPC to any server (local or remote).
- **Native macOS app** — Wails v2 application that bundles the web UI in a webview, manages local cluster lifecycle from the menu bar, and distributes as a signed `.app`.
- **Remote deployment docs + Helm publish** — finalize OCI chart publishing to `ghcr.io/uncworks/charts/aot`, document remote k8s deployment (EKS/GKE/AKS).
- **BREAKING**: `task k0s:setup` / `task k0s:teardown` are deprecated in favor of `uncworks setup` / `uncworks teardown`. k0s is dropped as the local cluster runtime; users bring their own cluster.

## Capabilities

### New Capabilities

- `uncworks-cli`: Top-level CLI binary (`cmd/uncworks`), `uncworks setup/teardown/status/open/connect` subcommands, kubeconfig context detection, Helm lifecycle management (install/upgrade/uninstall).
- `setup-wizard`: Interactive preflight wizard — context selection, resource validation (min CPU/memory), secrets collection (LLM key, GitHub token), progress feedback, post-install URL output.
- `tui-client`: Bubble Tea TUI — active run list, log streaming, run submission, gRPC address config (`~/.config/uncworks/config.yaml`).
- `macos-app`: Wails v2 `.app` bundle — embeds existing React frontend as webview, menu bar status icon, manages local cluster via `uncworks` CLI subprocess, distributed via GitHub Releases and Homebrew cask.
- `local-values`: `deploy/helm/values.local.yaml` preset and documentation for local cluster installs.

### Modified Capabilities

- `helm-chart`: Add OCI chart release step to CI pipeline; no behavioral requirement changes.
- `cluster-management`: Prior spec defined k0s + systemd as the cluster model. This change supersedes that with a bring-your-own-cluster model. The systemd/k0s path is removed.

## Impact

- **New**: `cmd/uncworks/` — CLI entrypoint and subcommands
- **New**: `cmd/uncworks-app/` — Wails macOS app entrypoint
- **New**: `deploy/helm/values.local.yaml`
- **Modified**: `deploy/helm/aot/` — chart publishing, values cleanup
- **Modified**: `Taskfile.yml` — deprecate k0s tasks, add `uncworks:build`, `app:build`
- **Modified**: `ci/main.go` — add Wails build target, darwin/arm64 + darwin/amd64 binary release
- **Removed**: `hack/k0s-setup.sh`, `hack/k0s-teardown.sh`, `hack/k0s-config.yaml` (or archived)
- **Dependencies**: Wails v2 (Go module), Bubble Tea + lipgloss (Go modules), Helm Go SDK or `helm` CLI subprocess
- **Platforms**: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64
