## Context

UNCWORKS currently has no end-user installation story. The local setup path requires k0s (a Linux-only systemd service), sudo access, and manual `task k0s:*` invocations. k0s does not run natively on macOS, so macOS developers are either using a Linux VM or blocked entirely. There is no CLI binary for end-users, no TUI client, and no native desktop app.

The goal is to deliver three surfaces: a cross-platform CLI (`uncworks`), a native macOS app (Wails), and a TUI client (Bubble Tea) — all built on a "bring your own Kubernetes cluster" model where the user provides the cluster runtime and UNCWORKS installs into it via Helm.

Images are already published to `ghcr.io/uncworks/...`, so no local image registry or import step is required for end-user installs.

## Goals / Non-Goals

**Goals:**
- Cross-platform `uncworks` CLI (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64)
- `uncworks setup` wizard: detect existing local k8s context, validate resources, deploy Helm chart
- `uncworks tui`: Bubble Tea client connecting to any UNCWORKS gRPC API server
- Native macOS `.app` via Wails v2, distributed via GitHub Releases and Homebrew cask
- `deploy/helm/values.local.yaml` for lightweight local installs
- OCI Helm chart publish to `ghcr.io/uncworks/charts/aot` in CI
- Deprecate k0s-based local setup (`hack/k0s-*.sh`, `task k0s:*`)

**Non-Goals:**
- Windows support (not in scope for this change)
- Bundling a Kubernetes runtime (users bring their own: Docker Desktop, OrbStack, Colima, etc.)
- Installing Docker or any container runtime on the user's behalf
- Bundling Temporal, Postgres, LiteLLM as Helm sub-charts (helm-chart spec already excludes this)
- Full feature parity between TUI and web UI in this change (TUI v1 = monitoring + submit)
- Linux desktop app (CLI is sufficient for Linux)

## Decisions

### D1: Bring-your-own-cluster vs bundling k3d

**Decision**: Bring-your-own-cluster. `uncworks setup` detects an existing kubeconfig context and installs into it. We do not create or manage the cluster runtime.

**Rationale**: Docker Desktop, OrbStack, Rancher Desktop, and Colima are all common developer prerequisites with built-in Kubernetes. Bundling k3d would add complexity (Docker socket detection, cluster lifecycle, port-forward-at-creation-time requirement on macOS) without meaningful benefit — the user almost certainly already has one of the above. For users with nothing installed, we print clear instructions and exit.

**Alternative considered**: Ship `uncworks` with embedded k3d management. Rejected because it adds a hard Docker dependency, requires port-binding at cluster creation time (macOS), and duplicates what existing tools already do well.

### D2: Wails v2 for macOS app

**Decision**: Use Wails v2 to build the native macOS app.

**Rationale**: The project is already Go + React/TypeScript. Wails v2 embeds a webview using macOS's native WebKit (WKWebView) and produces a standard `.app` bundle. No Rust (Tauri), no 200MB runtime (Electron). The existing `web/` frontend is reused as-is — the Wails app just wraps it in a webview and adds a menu bar icon for cluster status. The Go host process can shell out to `uncworks setup/teardown` to manage the local cluster.

**Alternative considered**: Tauri. Rejected because it adds Rust to the stack. Electron. Rejected for bundle size and performance.

### D3: Service exposure via `kubectl port-forward`

**Decision**: `uncworks open` and `uncworks tui` use `kubectl port-forward` to reach services, managed as a subprocess by the CLI.

**Rationale**: NodePort accessibility varies by cluster tool:
- Docker Desktop / OrbStack / Rancher Desktop: NodePort on localhost ✓
- Colima / minikube: NodePort on a non-localhost VM IP
- kind: NodePort not on host at all

`kubectl port-forward` works identically across all tools on both macOS and Linux. The CLI starts it on-demand, keeps the PID, and tears it down on exit. This is predictable behavior the user understands.

**Alternative considered**: Use NodePort directly with IP detection (`kubectl get nodes -o wide`). Rejected because Colima returns a dynamic IP that varies per session, making the stored URL stale.

### D4: XDG config directory on both platforms

**Decision**: Use `$XDG_CONFIG_HOME/uncworks/` (default: `~/.config/uncworks/`) on both macOS and Linux.

**Rationale**: Developer tools (kubectl, gh, etc.) use XDG on macOS. `~/Library/Application Support/` is for GUI apps. The CLI is a developer tool. XDG is widely understood and consistent across platforms.

### D5: Bubble Tea for TUI, gRPC as transport

**Decision**: TUI client uses Bubble Tea + lipgloss, communicates with the API server over the existing gRPC API.

**Rationale**: The gRPC API already provides all needed operations (ListAgentRuns, streaming logs, CreateAgentRun). Bubble Tea is the de-facto Go TUI framework. The TUI is just another gRPC client — the same as the web frontend, but terminal-rendered.

**TUI v1 scope**: run list with live status, log streaming for selected run, submit new run (repo + branch + spec). Full settings management is out of scope for v1.

### D6: Helm lifecycle via `helm` CLI subprocess

**Decision**: `uncworks setup` shells out to the `helm` CLI for install/upgrade/uninstall rather than using the Helm Go SDK.

**Rationale**: The Helm Go SDK is large and not stable API. The `helm` binary is already a dev dependency (in devbox.json). Shelling out is simpler, produces familiar output, and avoids SDK version coupling. `helm upgrade --install` handles both first-install and re-run idempotently.

**Prerequisite check**: `uncworks setup` validates that `helm` and `kubectl` are in PATH before proceeding.

### D7: Ollama opt-in in local values

**Decision**: `values.local.yaml` sets `ollama.enabled: false` by default. Users opt in explicitly.

**Rationale**: Ollama is ~4GB and CPU/GPU intensive. Most local users will use OpenRouter or OpenAI keys. The current helm chart already has no bundled Ollama sub-chart (per helm-chart spec), so this is purely a documentation/default values concern.

## Risks / Trade-offs

- **Wails webview quirks** → WKWebView behavior differs across macOS versions (especially pre-Sonoma). Mitigation: test on macOS 13 (Ventura) as minimum target. The existing React frontend already targets modern browsers so this risk is low.
- **Context detection heuristics** → kubeconfig context name patterns can be wrong (e.g., renamed contexts). Mitigation: always let the user confirm the selected context before proceeding; show server URL for verification.
- **Port-forward subprocess leak** → if `uncworks open` exits uncleanly, the port-forward subprocess may linger. Mitigation: write PID to `~/.config/uncworks/port-forward.pid`, check and kill stale processes on startup.
- **Cluster resource under-provisioning** → laptops with 8GB RAM and Docker Desktop at default limits (2 CPU / 2GB) will struggle. Mitigation: preflight check warns at <4 CPU / 4GB allocatable, hard-fails at <2 CPU / 2GB.
- **k0s deprecation** → existing contributors using k0s locally will need to migrate. Mitigation: `task k0s:*` targets print a deprecation warning pointing to `uncworks setup` for one release cycle before removal.

## Migration Plan

1. Add `uncworks setup` — new capability, no disruption to existing k0s users
2. Deprecate `task k0s:*` with warning messages (keep scripts for one release)
3. After one release cycle, remove `hack/k0s-*.sh` and k0s tasks
4. Rollback: revert deprecation warnings; k0s scripts are in git history

## Open Questions

- **Code signing for macOS app**: Apple notarization requires an Apple Developer account ($99/yr). Is this available? Without it, users get a Gatekeeper warning on first launch. We can document the workaround (`xattr -d com.apple.quarantine`) but it's not ideal.
- **Homebrew tap vs cask**: The CLI binary goes in a tap (`brew install uncworks/tap/uncworks`). The macOS `.app` goes in a cask (`brew install --cask uncworks/tap/uncworks-app`). Is maintaining a Homebrew tap in scope for this change or a follow-on?
- **Temporal dependency for local install**: The helm chart requires `temporal.host` and does not bundle Temporal. Should `uncworks setup` deploy Temporal from the official Helm chart as an optional step, or is it always out of scope (user provides their own)?
