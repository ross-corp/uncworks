## Audit Scope

The audit covers every layer of the platform. Each section lists what exists and what to check.

### Layer 1: Go Backend (6 binaries, 12 internal packages)

**Binaries (cmd/)**
| Binary | Purpose | Check |
|--------|---------|-------|
| `apiserver` | ConnectRPC API server | Endpoint correctness, auth, CORS, error handling |
| `controller` | K8s CRD reconciler | Reconcile logic, PipelineConfig passthrough, status updates |
| `temporal-worker` | Temporal activity host | Activity registration, LiteLLM client init, image env vars |
| `sidecar` | Agent pod RPC gateway | ExecCommand workdir, StartAgent, resolveWorkDir, SendInput |
| `hydration` | Init container | Worktree paths, devbox compose, manifest |
| `aot` | CLI tool | Commands, open functionality |

**Internal packages**
| Package | Purpose | Check |
|---------|---------|-------|
| `brain` | Knowledge/embedding store | Is this used? PostgreSQL dependency |
| `cli` | CLI helpers | Worktree finder |
| `controller` | CRD controller | AgentRun reconciliation, workflow triggering |
| `embeddings` | Embedding provider | Is this used? |
| `eventbus` | SSE event bus | Event routing |
| `hydration` | Workspace setup | Clone, worktree, devbox, manifest |
| `litellm` | LiteLLM client | Key provisioning, revocation |
| `server` | API + file handlers | gRPC, REST, structured logs, files, traces, thinking |
| `sidecar` | RPC gateway | Agent lifecycle, ExecCommand, NotifyEvent, traces |
| `temporal` | Workflow + activities | Spec-driven pipeline, single-agent, orchestration |
| `testutil` | Test helpers | Shared test utilities |

**Key questions:**
- Is `internal/brain` dead code? It requires PostgreSQL which isn't deployed.
- Is `internal/embeddings` dead code?
- Are all 12 e2e tests runnable? Do they reference stale APIs?
- Do contract tests cover the current API surface?
- Are there Go packages with 0% test coverage?

### Layer 2: Proto/API (2 proto files)

| File | Service | Check |
|------|---------|-------|
| `proto/aot/api/v1/api.proto` | AOTService | All RPCs match server implementation |
| `proto/aot/agent/v1/agent.proto` | AgentSidecarService | All RPCs match sidecar implementation |

**Key questions:**
- Do proto messages match the Go CRD types?
- Are there deprecated fields still in the proto?
- Is the generated code (`gen/go/`) up to date with the proto?

### Layer 3: Web UI (20 components, 4 views, 8 hooks)

**Components**
| Component | Check |
|-----------|-------|
| ActivityFeed | Label consistency (manage/impl), markdown rendering, tool pairing |
| CommandPaletteNew | Cancel action works, cmdk styling |
| ErrorBoundary | Used? Wraps the app? |
| FileExplorer | Two-column layout, file selection |
| FilePreview | Monaco editor, syntax detection |
| FileTree | Auto-refresh, hydration retry, auto-expand |
| RunStatusBadge | Badge variants match phases |
| ShellTerminal/Inner | WebSocket connection, xterm.js |
| Skeleton | Used by sidebar only |
| SpecEditor | **Unused?** Not imported anywhere. Dead code candidate. |
| StageProgress | Badge variants, progress calculation |
| Toast | CSS variable usage, auto-dismiss |
| TraceTimeline | Flame graph, diff viewer, span types |
| VerificationPanel | Gate display, result rendering |

**Views**
| View | Check |
|------|-------|
| Layout | Theme init, command palette, route outlet |
| NewRunView | Form fields, clone support, orchestration modes |
| RunDetailView | Tabs, Sheet, cancel/retry, HITL overlay |
| RunListView | Keyboard nav, filtering, polling |

**Hooks**
| Hook | Check |
|------|-------|
| apiFetch | Auth header injection, base URL |
| useClient | AOTClient methods, type mappings |
| useFiles | listDir, readFile endpoints |
| useThemeNew | Light/dark only (themes removed) |
| useTraces | Polling interval, error handling |
| useTraceSpans | SSE endpoint — does server support it? |
| use-mobile | Is this used? |
| use-toast | shadcn toast — used? |

**Key questions:**
- Is SpecEditor dead code?
- Does useTraceSpans SSE endpoint exist on the server?
- Is use-mobile used anywhere?
- Are there components with no story coverage that should have it?

### Layer 4: Kubernetes/Helm (5 templates, 5 Docker images)

**Helm templates**
| Template | Check |
|----------|-------|
| apiserver.yaml | Env vars, service selector, probe paths |
| controller.yaml | RBAC, service account, image |
| worker.yaml | LITELLM_BASE_URL, LITELLM_MASTER_KEY, pipeline env vars |
| web.yaml | nginx configmap, /api/ proxy, NodePort |
| rbac.yaml | ClusterRole permissions |

**Docker images**
| Image | Check |
|-------|-------|
| Dockerfile.agent-base | Base image currency, devbox version |
| Dockerfile.sidecar | pi version, openspec version, extension copy |
| Dockerfile.hydration | Go build, binary name |
| Dockerfile.controlplane | Multi-binary build |
| Dockerfile.web | nginx config copy, SPA fallback |

**Key questions:**
- Are Helm values.yaml defaults accurate?
- Does the CRD YAML match the Go types?
- Are Docker base images up to date?

### Layer 5: Extensions (1 file)

| File | Check |
|------|-------|
| aot-determinism.ts | Loop detection, role policies, ask_user, delegate_task, turn limit |

**Key questions:**
- Does the extension compile standalone?
- Are the Typebox schemas correct?
- Does the file-based HITL actually work end-to-end?

### Layer 6: CI/CD (5 workflows)

| Workflow | Check |
|----------|-------|
| ci.yml | Dagger pipeline, parallel execution, release-please dependency |
| doc-staleness.yml | Script exists, runs correctly |
| wiki-sync.yml | Token secret, flatten logic |
| release-chart.yaml | Chart publishing |
| release-images.yaml | Image publishing to GHCR |

**Key questions:**
- Does the Dagger pipeline actually pass in CI?
- Do release-chart and release-images work?
- Is the doc-staleness script too noisy?

### Layer 7: Tests (19 test files + 12 e2e + 3 contract)

**Unit tests by package**
| Package | Test Files | Check |
|---------|-----------|-------|
| controller | agentrun_controller_test.go, orchestration_test.go, multi_agent_test.go, webhook_test.go | Cover PipelineConfig passthrough? |
| hydration | hydrator_test.go, devbox_test.go | Cover new workspace layout? |
| server | grpc_test.go, security_test.go, thinking_test.go | Cover structured logs dedup? |
| sidecar | gateway_test.go | Cover ExecCommand workdir fix? Cover loop detection? |
| temporal | workflow_spec_driven_test.go, openspec_parsers_test.go | Cover manage/implement roles? |
| brain | integration_test.go, store_test.go | Require PostgreSQL? Dead tests? |
| litellm | client_test.go | Cover key provisioning? |
| eventbus | eventbus_test.go | Cover SSE? |
| cli | open_test.go, github_test.go | Cover worktree finder? |

**Key questions:**
- Which packages have 0% coverage for recent changes?
- Are brain/embeddings tests runnable without PostgreSQL?
- Do e2e tests reference stale APIs or old workspace layout?

### Layer 8: OpenSpec Specs (18 specs)

Verify each spec still accurately describes the system:

| Spec | Check |
|------|-------|
| agent-role-separation | Matches PI_ROLE implementation? |
| cluster-management | Still relevant? |
| container-images | Matches current Dockerfiles? |
| context-hydration | Matches new workspace layout? |
| deterministic-policy | Matches extension policies? |
| docs-rewrite | Matches current doc structure? |
| doc-staleness | Script works? |
| helm-chart | Matches current templates? |
| install-docs | Accurate? |
| persistent-run-storage | PVC layout correct? |
| pipeline-config | Matches PipelineConfig types? |
| reliable-ci | Matches Dagger pipeline? |
| run-pipeline | Matches workflow code? |
| run-verification | Matches verify gates? |
| semantic-search-api | Dead spec? Brain not deployed. |
| sidecar-exec | Matches ExecCommand? |
| spec-generation | Matches plan prompt? |
| subagent-visibility | Matches delegate_task? |

## Output Format

The audit produces:
1. A findings report in `openspec/changes/full-platform-audit/findings.md`
2. Follow-up proposals created via `/opsx:propose` for each significant finding
