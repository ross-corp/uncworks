## Context

Currently, running the AOT dev environment requires manually starting 5 processes with specific flags, env vars, and port assignments. If the machine reboots or a process crashes, everything must be manually restarted. k0s already runs as a system-level systemd service, but all AOT-specific services (temporal, controller, worker, apiserver, web UI) are ad-hoc.

Port conflicts have been a recurring issue (kube-router vs API server on 50051, metrics on 8080/8090). The persistent cluster must codify the correct port assignments.

## Goals / Non-Goals

**Goals:**
- All AOT services run as systemd user units under the developer's user account
- Services auto-restart on failure and start on boot
- Single command to install/start/stop the full stack
- Logs accessible via `journalctl --user`
- Web UI accessible on a stable port for browser testing
- Ollama with qwen2.5:0.5b available in the cluster

**Non-Goals:**
- Production deployment (this is dev-only)
- Multi-user support (single developer workstation)
- TLS or auth for dev services
- Replacing k0s's own systemd service (it already works)

## Decisions

1. **Systemd user units (not system units)**: User units don't require root, live in `~/.config/systemd/user/`, and can be managed with `systemctl --user`. k0s stays as a system service since it requires root. `loginctl enable-linger` ensures user units survive logout.

2. **Service dependency chain**: `aot-temporal.service` → `aot-controller.service` + `aot-worker.service` → `aot-apiserver.service` → `aot-web.service`. Each unit uses `After=` and `Requires=` to express dependencies. This ensures temporal is up before controller/worker try to connect.

3. **Environment files**: Each unit sources env vars from `deploy/systemd/env/<service>.env`. This keeps secrets and configuration (ports, image names, kubeconfig path) out of unit files and in one place. The env files are gitignored; a `.env.example` is committed.

4. **Port assignments**: Codify non-conflicting ports:
   - Temporal: 7233 (default)
   - API server: 50055
   - Controller metrics: 8095
   - Web UI (vite): 3000
   - kube-router metrics: 8181 (in k0s config)
   - kube-router BGP: 50051 (default, freed by API server move)

5. **Vite dev server for web UI**: Use `npx vite` in dev mode rather than building static files. This gives HMR during development. The vite proxy forwards API calls to the apiserver on 50055.

6. **Target group unit**: An `aot-cluster.target` unit that groups all services. `systemctl --user start aot-cluster.target` starts everything; `systemctl --user stop aot-cluster.target` stops everything.

## Risks / Trade-offs

- **User lingering required**: `loginctl enable-linger` must be run once. If not set, services die on logout. Mitigated by checking in `task cluster:setup`.
- **k0s is a system service, rest are user services**: Slight mismatch, but k0s needs root and the rest don't. The controller connects via kubeconfig regardless.
- **Vite dev server in systemd**: Vite isn't designed for long-running service use, but it works fine and gives us HMR. If it becomes flaky, we can switch to a static build served by a simple HTTP server.
- **SQLite Temporal DB**: The dev server uses `--db-filename .temporal.db` which is fine for dev but not durable across reinstalls. Acceptable trade-off.
