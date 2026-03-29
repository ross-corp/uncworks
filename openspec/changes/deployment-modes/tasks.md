## 1. Foundations

- [x] 1.1 Add `cmd/uncworks/` entrypoint with cobra or flag-based subcommand routing
- [x] 1.2 Add `~/.config/uncworks/config.yaml` read/write helpers (XDG-aware on macOS and Linux)
- [x] 1.3 Add prerequisite validation: check `kubectl` and `helm` in PATH, print install URLs on failure
- [x] 1.4 Add kubeconfig context enumeration: list contexts with server URLs, identify active context
- [x] 1.5 Add `go.mod` entries for Bubble Tea, lipgloss, and any new deps; verify CGO_ENABLED=0 still builds for all four targets

## 2. uncworks CLI Subcommands

- [x] 2.1 Implement `uncworks setup` — non-interactive path accepting all flags (`--context`, `--llm-key`, `--github-token`, `--temporal-host`)
- [x] 2.2 Implement `uncworks setup` — interactive TUI path: context selection list, resource preflight check, masked secret prompts
- [x] 2.3 Implement `uncworks teardown` with `--purge` flag (Helm uninstall; PVC deletion only with flag)
- [x] 2.4 Implement `uncworks status` — query pod status for all UNCWORKS components, print table
- [x] 2.5 Implement `uncworks open` — start `kubectl port-forward` subprocess, write PID file, kill stale PID on re-run, open browser
- [x] 2.6 Implement `uncworks connect <address>` — write gRPC server address to config file

## 3. Setup Wizard Details

- [x] 3.1 Implement cluster resource preflight: parse `kubectl get nodes -o json` for allocatable CPU/memory, apply warn/fail thresholds
- [x] 3.2 Implement `helm upgrade --install` invocation: build values from collected config, pass as `--set` or temp values file
- [x] 3.3 Implement post-install output: detect NodePort/port-forward URL, print web UI address and `uncworks open` instructions
- [x] 3.4 Implement no-cluster exit path: detect OS (`runtime.GOOS`), print macOS vs Linux install recommendations

## 4. Local Helm Values

- [x] 4.1 Create `deploy/helm/values.local.yaml` with NodePort on 30300, reduced resource requests, `ollama.enabled: false`, no storage class override
- [ ] 4.2 Verify all UNCWORKS pods schedule and reach Running on Docker Desktop (4 CPU / 4Gi) with local values
- [x] 4.3 Add `values.local.yaml` reference to `docs/getting-started.md`

## 5. Helm Chart OCI Publishing

- [x] 5.1 Add `helm package` + `helm push oci://ghcr.io/uncworks/charts/aot` step to `ci/main.go` release target
- [ ] 5.2 Test OCI install: `helm install uncworks oci://ghcr.io/uncworks/charts/aot --version <tag>` succeeds against a real cluster

## 6. TUI Client

- [x] 6.1 Create `cmd/uncworks-tui/` (or integrate as `uncworks tui` subcommand); wire Bubble Tea app skeleton
- [x] 6.2 Implement gRPC connection setup: use stored address from config; fall back to port-forward for local
- [x] 6.3 Implement run list view: fetch runs via ListAgentRuns gRPC, render paginated list with status indicators
- [x] 6.4 Implement live status polling: re-fetch run list on interval, update rows in-place
- [x] 6.5 Implement log streaming view: stream run output via gRPC, scrollable viewport, auto-follow toggle
- [x] 6.6 Implement submit run form: repo URL, branch, spec content fields; submit via CreateAgentRun gRPC
- [x] 6.7 Implement keyboard nav and `?` help overlay (j/k, Enter, q/Escape, n for new run)
- [x] 6.8 Implement graceful connection failure message with hint text
- [ ] 6.9 Test rendering in macOS Terminal.app (box-drawing chars, 256-color)

## 7. macOS App (Wails)

- [x] 7.1 Add Wails v2 to `go.mod`; create `cmd/uncworks-app/` with Wails app entrypoint
- [x] 7.2 Configure Wails to embed `web/dist` as the frontend; verify React app renders in WKWebView
- [x] 7.3 Implement menu bar status icon with Running/Stopped states; poll `uncworks status` or gRPC health check
- [x] 7.4 Implement "Start" menu item: invoke `uncworks setup` subprocess, stream output to app window
- [x] 7.5 Implement "Stop" menu item: invoke `uncworks teardown` subprocess
- [x] 7.6 Set minimum macOS deployment target to 13.0 in Wails config
- [x] 7.7 Add `wails build` target to `Taskfile.yml` (`task app:build`)
- [x] 7.8 Add Wails build step to `ci/main.go` on darwin runner; produce `.app` artifact
- [x] 7.9 Add DMG packaging step to CI release target; upload as GitHub release asset
- [ ] 7.10 Create Homebrew cask formula in `uncworks/homebrew-tap` repository (or document as follow-on if tap repo doesn't exist yet)

## 8. Deprecation Cleanup

- [x] 8.1 Add deprecation warning to `task k0s:setup`, `task k0s:teardown`, `task k0s:*` targets pointing to `uncworks setup`
- [x] 8.2 Update `docs/getting-started.md` to use `uncworks setup` as the primary local path
- [x] 8.3 Update `README.md` quick start section to reflect new CLI flow
- [x] 8.4 Archive `hack/k0s-*.sh` and `hack/k0s-config.yaml` (move to `hack/archive/` or delete — do not remove until k0s tasks are removed)

## 9. Cross-Platform Binary Releases

- [x] 9.1 Add `GOOS/GOARCH` matrix build for `uncworks` binary to `ci/main.go` release target: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- [x] 9.2 Verify all four targets build cleanly with CGO_ENABLED=0
- [x] 9.3 Upload binaries as GitHub release assets with naming convention `uncworks-<os>-<arch>`
- [ ] 9.4 Add Homebrew tap formula for `uncworks` CLI (separate from cask) — or document as follow-on
