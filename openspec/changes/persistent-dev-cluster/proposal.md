## Why

Every time we want to test AOT, we manually start 5+ processes (temporal, controller, temporal-worker, apiserver, vite). If anything crashes or the machine reboots, everything stops and we have to remember the exact flags and env vars to restart. We need a persistent dev cluster that runs as systemd services, auto-restarts on failure, and starts on boot — so there's always a live environment to test against. This complements the ephemeral E2E test clusters with a long-lived cluster for manual testing and development.

## What Changes

- Add systemd user units for each AOT service: temporal dev server, AOT controller, temporal-worker, API server, and vite web UI dev server
- Each unit auto-restarts on failure, starts on boot (via lingering), and logs to journalctl
- Add a `task cluster:setup` target that installs all units and starts them
- Add a `task cluster:status` target that shows the health of all services
- Add a `task cluster:teardown` target that stops and removes all units
- Ensure Ollama is deployed with qwen2.5:0.5b pre-pulled as part of cluster setup
- Web UI served persistently via vite dev server (or built static files served by the API server)

## Capabilities

### New Capabilities
- `systemd-services`: Systemd user unit definitions and management for all AOT dev services
- `cluster-management`: Taskfile targets for setup, status, teardown, and log viewing of the persistent cluster

### Modified Capabilities

None.

## Impact

- New files: `deploy/systemd/*.service` unit files
- Modified: `Taskfile.yml` — new `cluster:*` targets
- Dependencies: systemd with user units, `loginctl enable-linger` for boot persistence
- k0s remains managed via its own systemd service (already exists)
- No code changes to Go binaries — configuration is via environment variables in unit files
