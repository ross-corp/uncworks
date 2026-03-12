## 1. Environment Files

- [x] 1.1 Create `deploy/systemd/env/` directory with `.env.example` showing all required variables
- [x] 1.2 Create env files for each service: `temporal.env`, `controller.env`, `worker.env`, `apiserver.env`, `web.env`
- [x] 1.3 Add `deploy/systemd/env/*.env` to `.gitignore` (keep `.env.example` tracked)

## 2. Systemd Unit Files

- [x] 2.1 Create `deploy/systemd/aot-temporal.service` — Temporal dev server with SQLite on port 7233
- [x] 2.2 Create `deploy/systemd/aot-controller.service` — AOT controller with KUBECONFIG, TEMPORAL_HOST, METRICS_ADDR=:8095
- [x] 2.3 Create `deploy/systemd/aot-worker.service` — temporal-worker with local image env vars
- [x] 2.4 Create `deploy/systemd/aot-apiserver.service` — API server on port 50055
- [x] 2.5 Create `deploy/systemd/aot-web.service` — vite dev server on port 3000
- [x] 2.6 Create `deploy/systemd/aot-cluster.target` — target group for all services

## 3. Taskfile Targets

- [x] 3.1 Add `task cluster:setup` — enable linger, install units, build/import images, deploy Ollama, pull model, start target
- [x] 3.2 Add `task cluster:status` — show status of all aot-* units and listening ports
- [x] 3.3 Add `task cluster:teardown` — stop target, disable and remove units
- [x] 3.4 Add `task cluster:logs` — journalctl for all aot-* units

## 4. Validation

- [x] 4.1 Run `task cluster:setup` and verify all services start
- [x] 4.2 Run `task cluster:status` and verify all services show active
- [x] 4.3 Verify web UI accessible at http://localhost:3000
- [x] 4.4 Verify API server responds at http://localhost:50055
- [x] 4.5 Run `task cluster:teardown` and verify clean shutdown
