## Context

The sidecar (`internal/sidecar/gateway.go`) already captures pi-coding-agent stdout/stderr via `StreamOutput` RPC and broadcasts to subscribers via `AgentProcess.outputs` channels. The control plane has an in-memory EventBus that publishes to `WatchAgentRun` streaming RPC subscribers. The shared TS client has a `watchAgentRun()` method. None of these are connected — the web UI does not subscribe to `WatchAgentRun`, and the control plane does not bridge sidecar logs into the EventBus. Pods are deleted immediately on workflow completion via a `defer` block in `AgentRunWorkflow`.

The K8s exec API (`client-go/tools/remotecommand`) supports interactive exec into pod containers. The API server already has the K8s client. No WebSocket infrastructure exists yet.

## Goals / Non-Goals

**Goals:**
- Stream agent logs (stdout/stderr with ANSI colors) to the web UI in real-time
- Browse agent pod filesystem from the web UI (directory tree + file preview)
- Drop into an interactive shell (bash) in the agent pod from the browser
- Keep pods alive after completion long enough for inspection (configurable retention)
- Persist log output permanently so logs survive pod deletion
- Redesign detail panel with tabs: Info | Logs | Files | Shell
- Cover new features with E2E tests (Playwright + Go)

**Non-Goals:**
- Recording/replaying shell sessions
- Multi-user concurrent shell access to same pod
- File editing through the browser (read-only browsing only)
- Streaming logs from the hydration init container (agent logs only)
- OpenTelemetry/Jaeger integration (trace ID field exists but is out of scope)

## Decisions

### 1. Log transport: Sidecar → API server via Temporal activity polling

**Decision**: A new `CollectAgentLogs` long-running Temporal activity runs alongside the status polling loop. It connects to the sidecar's `StreamOutput` RPC, receives stdout/stderr lines, and publishes them to the EventBus as `AGENT_RUN_EVENT_TYPE_LOG` events. The `WatchAgentRun` streaming RPC naturally delivers these to web UI subscribers.

**Rationale**: Reuses the existing EventBus → WatchAgentRun pipeline. The sidecar already captures and broadcasts logs — we just need a consumer on the control plane side. A Temporal activity is the right place because it has access to the pod IP and runs for the duration of the agent's execution.

**Alternative considered**: Direct sidecar-to-browser connection — rejected because the sidecar runs inside the cluster and isn't exposed externally. The API server must proxy.

### 2. Log rendering: xterm.js (not plain text)

**Decision**: Use xterm.js in the browser to render log output. Pi-coding-agent emits ANSI escape codes (colors, bold, cursor movement). xterm.js renders these faithfully, giving the same experience as watching the agent in a real terminal.

**Rationale**: Plain text log viewers strip ANSI codes, losing valuable formatting. Pi-coding-agent's output is designed for terminal consumption. xterm.js is lightweight (~100KB), widely used, and handles all terminal escape sequences.

### 3. File explorer: REST endpoints using K8s exec

**Decision**: Two new REST endpoints on the API server:
- `GET /api/v1/runs/{id}/files?path=...` — execs `ls -la --time-style=long-iso <path>` in the agent pod, parses output into JSON directory listing
- `GET /api/v1/runs/{id}/files/content?path=...` — execs `cat <path>` in the agent pod, returns raw file content

Both use the K8s exec API (`client-go/tools/remotecommand`) targeting the `rpc-gateway` container (which has access to the `/workspace` volume and standard Unix tools).

**Rationale**: No new RPC needed on the sidecar. Standard Unix tools are available in the sidecar image. The API server already has a K8s client. REST is simpler than gRPC for file downloads and fits the request-response pattern.

**Alternative considered**: Adding file RPCs to the sidecar proto — rejected as unnecessary complexity when exec works.

### 4. Interactive shell: WebSocket ↔ K8s SPDY exec bridge

**Decision**: New WebSocket endpoint `GET /api/v1/runs/{id}/exec` that:
1. Upgrades HTTP to WebSocket
2. Looks up the pod name from the AgentRun CRD
3. Opens a SPDY exec connection to the pod's `rpc-gateway` container (`bash -l`)
4. Bridges stdin/stdout/stderr between WebSocket and SPDY
5. Handles terminal resize messages (JSON `{type: "resize", cols: N, rows: N}`)

The web UI connects via xterm.js with the `attach` addon, sending keystrokes over WebSocket and rendering output.

**Rationale**: This is exactly how `kubectl exec -it` works under the hood — SPDY exec to the kubelet. The WebSocket-to-SPDY bridge is a well-understood pattern (used by the K8s dashboard, Lens, k9s). The `rpc-gateway` container already has bash and access to `/workspace`.

### 5. Pod retention: configurable delay before cleanup

**Decision**: Add `retain_pod_minutes` field to proto `AgentRunSpec` (default 30). After the workflow reaches a terminal state, instead of immediately deleting the pod, the cleanup defer block waits for `retainPodMinutes` before calling `CleanupPod`. During retention, logs/files/shell remain accessible. A `RetainUntil` timestamp is added to CRD status so the UI can show a countdown.

**Rationale**: Simplest approach — no new infrastructure. The pod stays alive with the sidecar running, so all exec-based operations keep working. The workspace volume persists. After retention expires, normal cleanup runs.

**Trade-off**: Pods consume cluster resources during retention. 30 minutes default is a reasonable balance. Users can set 0 to restore immediate cleanup behavior.

### 6. Log persistence: collect before pod deletion

**Decision**: Before the cleanup defer deletes the pod, a new `CollectLogs` activity reads the sidecar's accumulated log buffer (or the pod's container logs via K8s API) and stores them on the AgentRun CRD status as a `LogOutput` string field (truncated to 1MB). After pod deletion, the web UI falls back to this persisted log data.

**Rationale**: CRD storage is simple and already used for status. 1MB covers most agent sessions. For longer sessions, logs are available in real-time via streaming while the pod is alive; persistence is for post-mortem review.

**Alternative considered**: External log storage (S3, PVC) — rejected as premature. CRD field is sufficient for v1.

### 7. Detail panel tabs: Info | Logs | Files | Shell

**Decision**: Transform `AgentRunDetailPanel` from a single scrollable metadata view into a tabbed interface:
- **Info** — current metadata view (phase, repos, prompt, env vars, status message, HITL input)
- **Logs** — xterm.js log viewer with auto-scroll, subscribes to WatchAgentRun while pod alive, falls back to persisted logs
- **Files** — tree view (left) + Monaco read-only preview (right), lazy-loads directory contents on expand
- **Shell** — xterm.js interactive terminal, connects via WebSocket, disabled when pod is gone

Tabs show availability indicators: Logs always available (live or persisted), Files/Shell only when pod exists (with "Pod expired" message otherwise).

**Rationale**: Tabs keep the panel clean while providing deep access. The current metadata view becomes the Info tab with zero loss. Each capability gets focused UI real estate.

### 8. E2E test strategy

**Decision**: Add tests at both layers:

**Go E2E** (`e2e/`):
- `TestE2E_LogStreaming` — create run, subscribe to WatchAgentRun, verify LOG events arrive
- `TestE2E_FileExplorer` — create run, wait for Running, GET files endpoint, verify directory listing
- `TestE2E_FileContent` — GET file content endpoint, verify file body
- `TestE2E_ExecEndpoint` — WebSocket connect to exec, send `echo hello`, verify response
- `TestE2E_PodRetention` — create run with retain_pod_minutes=1, verify pod exists after completion
- `TestE2E_LogPersistence` — create run, wait for completion + pod deletion, verify logs on CRD status

**Playwright** (`web/e2e/`):
- `observability.spec.ts` — verify Logs tab renders with content, Files tab shows tree, Shell tab shows terminal
- Integration with existing `lifecycle.spec.ts` — verify log output appears during run execution

## Risks / Trade-offs

**Pod retention resource usage** — Keeping pods alive 30 minutes post-completion uses cluster resources. → Mitigation: Configurable per-run, default reasonable, UI shows countdown to expiry.

**Log persistence size** — 1MB CRD field limit may truncate long sessions. → Mitigation: Acceptable for v1. Most agent sessions produce <100KB of output. Real-time streaming captures everything; persistence is for convenience.

**WebSocket complexity** — The exec bridge adds WebSocket infrastructure to a previously REST/gRPC-only server. → Mitigation: Well-understood pattern. gorilla/websocket is battle-tested. The bridge is a single handler function.

**K8s exec permissions** — The API server's service account needs exec permissions on agent pods. → Mitigation: Already has pod create/delete via the controller. Add exec to the RBAC ClusterRole.

**xterm.js bundle size** — Adds ~100KB to the JS bundle. → Mitigation: Lazy-load the Logs/Shell tabs (same pattern as Monaco lazy loading for SpecEditor).
