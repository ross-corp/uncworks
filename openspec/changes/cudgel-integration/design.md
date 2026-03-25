## Context

Uncworks agent runs currently have no structural awareness of the codebase they operate on. The `internal/hydration/` package provisions the workspace (git clone, devbox, spec injection) but writes no code-level context. Agents must manually grep or read files to orient themselves, repeating discovery work every run.

Cudgel (https://github.com/roshbhatia/cudgel) is a Rust CLI for semantic code search: TreeSitter symbol extraction, all-MiniLM-L6-v2 ONNX embeddings at 384 dims, pgvector backend, and a `cudgel graph <symbol>` call-graph traversal command. It outputs structured JSON. The cluster already has a postgres deployment with pgvector enabled (`deploy/postgres/migrations/001_knowledge_schema.sql`). Cudgel can share this instance.

Relevant entry points:
- `internal/hydration/hydrator.go` — `Hydrator.Run()` is the init-container entrypoint; new context seeding hooks in here
- `internal/sidecar/gateway.go` — sidecar server that brokers agent tool calls; new `semantic_search` tool registered here
- `deploy/helm/aot/` — service deployment templates (apiserver, bff, soft-serve, worker patterns to follow)

## Goals / Non-Goals

**Goals:**
- Deploy cudgel as a k8s Deployment with a thin HTTP shim exposing search and graph endpoints
- Build `internal/cudgel/` Go client with `SemanticSearch` and `GraphTraversal` methods
- Seed codebase context at run start via a new `SeedCodebaseContext` step in `Hydrator.Run()`
- Expose `semantic_search` as a callable agent tool in the sidecar gateway
- Reuse the existing postgres+pgvector instance (no new database deployment)

**Non-Goals:**
- Replace or modify the existing past-work embedding pipeline (`vector-embedding-pipeline`) or `SearchPastWork` gRPC endpoint
- Build a full RAG pipeline — cudgel owns the embedding/indexing, we only integrate
- Index automation (post-commit hooks, CronJob reindex) — Phase 2
- MCP server integration, Claude Code tooling, or developer-facing tooling

## Decisions

### Decision: HTTP shim as a separate container, not a sidecar per agent pod

**Rationale**: Cudgel's index is per-repo and shared across many runs. Co-locating it as a sidecar would require a full ONNX model load and index build per agent pod — expensive and unnecessary. A single Deployment per cluster (or one per tenant) allows the index to stay warm, amortizes the ONNX model load, and gives a stable `cudgel.aot.svc.cluster.local` endpoint.

**Alternative considered**: Sidecar per pod. Rejected because index initialization cost (~2-5s) on every run start is unacceptable, and the index data would need to be re-built from scratch each pod launch unless backed by a PVC.

**Shim design**: Thin Go HTTP server (`cmd/cudgel-shim/`) that shells out to the `cudgel` binary. Three endpoints:
- `POST /search` → `cudgel query <text> --limit <n> --json`
- `POST /graph` → `cudgel graph <symbol> --depth <n> --json`
- `POST /index` → `cudgel index <repo_path>` (used by CronJob in Phase 2)

### Decision: `internal/cudgel/` wraps the HTTP shim, not the CLI directly

**Rationale**: The Go application code runs in agent pods, not alongside the cudgel binary. HTTP is the right boundary. This also makes `internal/cudgel/` easy to mock in tests — just swap the endpoint.

**Client interface**:
```go
type Client interface {
    SemanticSearch(ctx context.Context, query string, limit int) ([]Symbol, error)
    GraphTraversal(ctx context.Context, symbol string, depth int) ([]Edge, error)
}
```

`Symbol` carries: `name`, `kind` (function/struct/etc), `file`, `line`, `snippet`, `score float64`.
`Edge` carries: `from`, `to`, `kind` (calls/imports/implements).

### Decision: Context seeding in `Hydrator.Run()`, not as a Temporal activity

**Rationale**: The hydrator is already the init-container entrypoint. Adding a `SeedCodebaseContext` step there keeps the architecture simple — no new Temporal activity type, no workflow topology change. The hydrator already has a 5-second-ish timeout-or-continue pattern for devbox (logs warning, proceeds). Same pattern applies: call cudgel, write `.aot/context/codebase.md` if results come back, proceed regardless of failure.

**Context file location**: `.aot/context/codebase.md` (parallel to `past-work.md` from existing context-hydration spec). Agent system prompt already references `.aot/context/` as a context source.

**Query**: Run's prompt text is used as the search query. Top-K = 10 for regular agents, 20 for senior agents (consistent with existing past-work hydration K values).

### Decision: `semantic_search` tool added to sidecar gateway, not brain/search

**Rationale**: The sidecar gateway (`internal/sidecar/gateway.go`) is where all agent-callable tools are registered. Adding a new RPC method to the `AgentSidecarService` proto and a handler in gateway.go is the established pattern. The gateway already has access to env vars that can carry `CUDGEL_ENDPOINT`.

**Tool signature** (proto):
```protobuf
rpc SemanticSearch(SemanticSearchRequest) returns (SemanticSearchResponse);
message SemanticSearchRequest { string query = 1; int32 limit = 2; }
message SemanticSearchResponse { repeated CodeChunk chunks = 1; }
message CodeChunk { string name = 1; string kind = 2; string file = 3; int32 line = 4; string snippet = 5; float score = 6; }
```

### Decision: Reuse existing postgres, enable cudgel's own database

**Rationale**: The cluster already runs postgres with pgvector enabled. Cudgel uses pgvector for its index. We create a `cudgel` database in the existing postgres instance and pass the DSN to the cudgel-shim Deployment via k8s Secret. No new postgres StatefulSet needed.

**Migration**: A new SQL migration (`deploy/postgres/migrations/002_cudgel_db.sql`) runs `CREATE DATABASE IF NOT EXISTS cudgel` and grants the cudgel user access.

## Risks / Trade-offs

- **Cudgel is a CLI tool, not a server** → Shim shells out per-request. Under high concurrency, this means multiple cudgel processes. Acceptable for Phase 1 (agents query at start + sporadically during run). Mitigated by request timeout (10s) and simple connection pooling at the shim level.
- **Index staleness** → Without Phase 2 reindex automation, the cudgel index reflects the codebase at last manual `cudgel index` run. Agents may get outdated symbol data. Mitigation: document that the index is a best-effort snapshot; Phase 2 adds CronJob.
- **Cudgel endpoint unavailable at hydration** → If cudgel pod is down, `SeedCodebaseContext` must degrade gracefully (log + skip). This is already the established pattern for context hydration failures.
- **Proto changes require regeneration** → Adding `SemanticSearch` to the agent sidecar proto requires running `buf generate`. Tracked in tasks.

## Migration Plan

1. Enable pgvector on existing postgres (if not already — migration 001 already does this)
2. Run `deploy/postgres/migrations/002_cudgel_db.sql` to create cudgel database
3. Build and push `cudgel-shim` container image
4. Apply helm chart changes (`deploy/helm/aot/templates/cudgel.yaml`)
5. Run initial `POST /index` against the target repo to populate the index
6. Deploy updated agent image (with new sidecar gateway proto + cudgel tool)
7. Deploy updated hydration init-container image

**Rollback**: Set `cudgel.enabled: false` in helm values. `SeedCodebaseContext` is a no-op when `CUDGEL_ENDPOINT` env var is unset. `SemanticSearch` tool returns empty results when endpoint is unavailable. Zero risk to existing run behavior.

## Open Questions

- Should `cudgel index` be scoped per-repo or index all repos together? (Leaning: per-repo with repo URL as namespace key in cudgel's own metadata)
- What is the correct resource sizing for the cudgel-shim Deployment? (Leaning: 1 replica, 256Mi/0.25 CPU as starting point, same as soft-serve)
- Should `semantic_search` tool results be appended to the trace spans for observability? (Leaning: yes, as a lightweight span with query + result count)
