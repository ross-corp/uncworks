## 1. Infrastructure: local-path-provisioner

- [x] 1.1 Install local-path-provisioner in k0s: `kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml`
- [x] 1.2 Set `local-path` as default StorageClass: `kubectl patch storageclass local-path -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'`
- [x] 1.3 Verify PVC creation works: create a test PVC, mount it, write a file, delete Pod, verify file persists on host
- [x] 1.4 Add local-path-provisioner setup to `Taskfile.yml` as `k0s:storage` task
- [x] 1.5 Add local-path-provisioner to `cluster:setup` task deps

## 2. Proto/CRD: Schema Updates

- [x] 2.1 Remove `retain_pod_minutes` from proto `AgentRunSpec` (replaced by archive retention)
- [x] 2.2 Add `string deployment_name = 10` to proto `AgentRunStatus`
- [x] 2.3 Add `bool debug_active = 11` to proto `AgentRunStatus`
- [x] 2.4 Add `DeploymentName string` and `DebugActive bool` to CRD `AgentRunStatus` in `types.go`
- [x] 2.5 Remove `RetainPodMinutes` from CRD `AgentRunSpec`
- [x] 2.6 Regenerate proto Go + TS code
- [x] 2.7 Update `specProtoToCRD` and `crdToProto` in `grpc.go`
- [x] 2.8 Update shared TS types and `toAgentRun` mapping

## 3. Temporal: Deployment + PVC Activities

- [x] 3.1 Create `CreateAgentDeployment` activity: creates PVC (`aot-ws-{runName}`, 2Gi, local-path) + Deployment (replicas=1) with PVC mounted at `/workspace`. Deployment spec matches current `BuildAgentPod` but with PVC volume instead of emptyDir.
- [x] 3.2 Create `ScaleDownDeployment` activity: patches Deployment `replicas=0`. Does NOT delete Deployment or PVC.
- [x] 3.3 Create `ArchiveAndCleanup` activity: deletes Deployment + PVC.
- [x] 3.4 Update `WaitForHydration` activity to work with Deployment-managed Pods (look up Pod via label selector with `findPod` helper)
- [x] 3.5 Added `findPod` helper for label-based pod discovery used by all activities
- [x] 3.6 Old `BuildAgentPod`, `CreateAgentPod`, `CleanupPod` activities kept as deprecated

## 4. Temporal: Workflow Updates

- [x] 4.1 Replace `CreateAgentPod` with `CreateAgentDeployment` in `AgentRunWorkflow`
- [x] 4.2 Replace `CleanupPod` with `ScaleDownDeployment` in the defer block (immediate scale-down, not delete)
- [x] 4.3 Removed retention timer sleep from defer block (PVC persists, no longer needed)
- [x] 4.4 Store `deploymentName` in workflow state (available via query for controller sync)
- [x] 4.5 `SpawnJuniorWorkflow` uses new activities (inherits from `AgentRunWorkflow`)
- [x] 4.6 Updated all workflow tests for Deployment-based lifecycle

## 5. Sidecar: Log Tee + Debug Mode

- [x] 5.1 In `startAgentProcess`, tee stdout/stderr to `/workspace/.aot/logs/agent.log` using `io.MultiWriter`
- [x] 5.2 Create `.aot/logs/` directory at sidecar startup
- [x] 5.3 Add debug mode check: if annotation `aot.uncworks.io/mode=debug` is set, skip agent launch, just serve RPC gateway and log "Debug mode — waiting for connections"
- [x] 5.4 Read annotations from Kubernetes downward API (projected volume or environment variable)

## 6. Hydrator: devcontainer.json + Trace Setup

- [x] 6.1 Generate `/workspace/.devcontainer/devcontainer.json` in `Hydrator.Run()` after manifest generation
- [x] 6.2 Create `/workspace/.aot/traces/` directory for trace span storage
- [x] 6.3 Create `/workspace/.aot/logs/` directory for agent log file
- [x] 6.4 Write `metadata.json` to `/workspace/.aot/` with run spec snapshot (repos, prompt, model, etc.)

## 7. API: File/Log Endpoints — Dual-Mode (exec or disk)

- [x] 7.1 Add helper `getDeploymentReplicas(runID) (int32, error)` — checks if Deployment has running Pods
- [x] 7.2 Add helper `getPVCHostPath(runID) (string, error)` — reads PV spec to find host path for the run's PVC
- [x] 7.3 Update `handleListFiles`: if Pod running → exec `ls` (current). If not → read directory from PVC host path using `os.ReadDir`.
- [x] 7.4 Update `handleFileContent`: if Pod running → exec `cat`. If not → read file from PVC host path using `os.ReadFile`.
- [x] 7.5 Update `handleLogs`: if Pod running → stream container logs. If not → read `/workspace/.aot/logs/agent.log` from PVC host path.
- [x] 7.6 Add error handling for PVC not found (archived/deleted runs)

## 8. API: Debug Pod Endpoint

- [x] 8.1 Add `POST /api/v1/runs/{id}/debug` handler: patches Deployment replicas=1, adds debug annotation
- [x] 8.2 Add `DELETE /api/v1/runs/{id}/debug` handler: patches Deployment replicas=0, removes debug annotation
- [x] 8.3 Add `GET /api/v1/runs/{id}/connect` handler: returns connection info (pod name, namespace, SSH port, kubectl command)
- [x] 8.4 Register endpoints on API server mux
- [x] 8.5 Update CRD status `debugActive` when debug starts/stops

## 9. API: Trace Endpoints

- [x] 9.1 Add `GET /api/v1/runs/{id}/traces` handler: returns trace spans as JSON array (from PVC disk or PostgreSQL)
- [x] 9.2 Add `GET /api/v1/runs/{id}/traces/{span-id}/diff` handler: returns git diff associated with a span
- [x] 9.3 Define trace span schema: `{id, parentId, name, type (llm|tool|thought|input), startTime, endTime, metadata, hasDiff}`
- [x] 9.4 Define diff schema: `{spanId, files: [{path, before, after, patch}]}`

## 10. Trace Collection in Sidecar

- [x] 10.1 Add trace span recording to sidecar: capture tool call events from AgentNotificationService `NotifyEvent` as spans
- [x] 10.2 On `EVENT_TYPE_TOOL_CALL`: record span with tool name, arguments, timestamp. Run `git diff` in workspace and capture output.
- [x] 10.3 On `EVENT_TYPE_LOG` with structured LLM response: record span with model, tokens, duration
- [x] 10.4 Write spans to `/workspace/.aot/traces/spans.jsonl` (one JSON object per line, append-only)
- [x] 10.5 Optionally persist spans to PostgreSQL via API call (if configured)
- [x] 10.6 Ensure trace spans include git diffs computed between tool calls (`git diff HEAD` before and after each tool execution)

## 11. Web UI: Trace Timeline Component

- [x] 11.1 Create `TraceTimeline` component: horizontal timeline of spans with type-based coloring (blue=LLM, green=tool, purple=thought, orange=input)
- [x] 11.2 Each span is a clickable bar showing name, duration, and type icon
- [x] 11.3 Create `DiffViewer` component: side-by-side or unified diff view (use Monaco diff editor or a lightweight diff renderer)
- [x] 11.4 Clicking a tool-call span opens the DiffViewer showing file changes from that span
- [x] 11.5 Clicking an LLM span shows the prompt/response summary
- [x] 11.6 Create `useTraces` hook: fetches traces from `GET /api/v1/runs/{id}/traces`, fetches diffs on demand
- [x] 11.7 Add "Traces" tab to detail panel tab bar (Info | Logs | Files | Shell | Traces)

## 12. Web UI: Detail Panel Updates

- [x] 12.1 Update Shell tab: show "Debug Run" button when Deployment replicas=0, "Stop Debug" when debug active
- [x] 12.2 Add VS Code connection info display: show `kubectl port-forward` command and devcontainer attach instructions
- [x] 12.3 Update `hasPod` logic: check Deployment replicas instead of just podName + active status
- [x] 12.4 Remove `retainPodMinutes` from create form (replaced by archive retention)
- [x] 12.5 Add Traces tab integration per section 11

## 13. Web Type Updates

- [x] 13.1 Remove `retainPodMinutes` from web `AgentRunSpec` type
- [x] 13.2 Add `deploymentName`, `debugActive` to web `AgentRunStatus` type
- [x] 13.3 Update `mapRun()` for new status fields
- [x] 13.4 Add `TraceSpan` and `SpanDiff` types
- [x] 13.5 Update `AgentRunForm` to remove Retain Pod field

## 14. Controller Updates

- [x] 14.1 Update controller to set `deploymentName` on CRD status during sync
- [x] 14.2 Add archive cleanup reconciliation: for runs with `completedAt` older than 7 days and Deployment still exists, trigger `ArchiveAndCleanup`
- [x] 14.3 Update RBAC: add Deployment create/update/delete, PVC create/delete permissions to controller service account
- [x] 14.4 Update RBAC: add pods/exec permission to API server service account

## 15. Documentation

- [x] 15.1 Update `README.md`: document new architecture (Deployment + PVC per run, three-layer observability)
- [x] 15.2 Update `AGENTS.md`: document workspace layout (`/workspace/.aot/`, `.devcontainer/`, trace files)
- [x] 15.3 Add architecture diagram to docs
- [x] 15.4 Document VS Code attachment workflow for developers

## 16. E2E Tests: Go API

- [x] 16.1 Add `TestE2E_DeploymentLifecycle`: create run → verify Deployment + PVC created → wait for completion → verify replicas=0, PVC exists
- [x] 16.2 Add `TestE2E_PersistentFiles`: after completion, GET files endpoint → verify workspace files readable from disk
- [x] 16.3 Add `TestE2E_PersistentLogs`: after completion, GET logs endpoint → verify agent.log readable from disk
- [x] 16.4 Add `TestE2E_DebugPod`: POST /debug → verify Deployment scales to 1 → exec shell works → DELETE /debug → verify scales to 0
- [x] 16.5 Add `TestE2E_Traces`: create run → wait for completion → GET /traces → verify spans exist with tool calls
- [x] 16.6 Add `TestE2E_TraceDiff`: GET trace span diff → verify file changes are captured
- [x] 16.7 Add `TestE2E_DevcontainerJson`: after hydration, verify `.devcontainer/devcontainer.json` exists in workspace

## 17. E2E Tests: Playwright

- [x] 17.1 Add test: completed run → Logs tab shows content from disk (no pod needed)
- [x] 17.2 Add test: completed run → Files tab shows tree from disk
- [x] 17.3 Add test: completed run → Shell tab shows "Debug Run" button → click → terminal appears
- [x] 17.4 Add test: Traces tab → timeline renders spans → click span → diff view appears
- [x] 17.5 Add test: running run → all tabs work (live mode)

## 18. Verification

- [x] 18.1 Run `go build ./...` — all Go code compiles
- [x] 18.2 Run `npx tsc --noEmit -p web/tsconfig.json` — web compiles
- [x] 18.3 Run `go test ./internal/... ./test/...` — all existing tests pass
- [x] 18.4 Deploy to aot-local: rebuild images, import to k0s, restart services
- [x] 18.5 Create agent run → verify Deployment + PVC in cluster
- [x] 18.6 Watch logs stream in Logs tab → verify real-time output
- [x] 18.7 Browse files in Files tab → verify workspace visible
- [x] 18.8 After completion → verify Logs/Files still work (from disk)
- [x] 18.9 Click "Debug Run" → verify shell access → "Stop Debug"
- [x] 18.10 Open Traces tab → verify timeline with spans and diffs
- [x] 18.11 Validate with user against aot-local cluster
