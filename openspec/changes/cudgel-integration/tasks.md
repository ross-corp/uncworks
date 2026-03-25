## 1. Database Setup

- [ ] 1.1 Write `deploy/postgres/migrations/002_cudgel_db.sql` — creates `cudgel` database, grants cudgel user permissions
- [ ] 1.2 Verify pgvector extension is enabled in existing postgres (check migration 001 is applied in cluster)
- [ ] 1.3 Create k8s Secret `cudgel-db-credentials` in aot namespace with `CUDGEL_DATABASE_URL`

## 2. Cudgel HTTP Shim

- [ ] 2.1 Create `cmd/cudgel-shim/main.go` — Go HTTP server with three handlers: `/search`, `/graph`, `/index`
- [ ] 2.2 Implement `/search` handler — shells out to `cudgel query <text> --limit <n> --json`, returns JSON array of symbols
- [ ] 2.3 Implement `/graph` handler — shells out to `cudgel graph <symbol> --depth <n> --json`, returns JSON array of edges
- [ ] 2.4 Implement `/index` handler — validates `repo_path` prefix allowlist, runs `cudgel index <path>` asynchronously, returns 202
- [ ] 2.5 Add error handling: binary-not-found → 503, bad input → 400, process failure → 503
- [ ] 2.6 Write unit tests for each handler in `cmd/cudgel-shim/main_test.go`
- [ ] 2.7 Write `Dockerfile.cudgel-shim` — multi-stage build: compile Go shim, fetch cudgel Rust binary, produce final image
- [ ] 2.8 Add `cudgel-shim` image to CI build pipeline (`.github/workflows/` or equivalent)

## 3. Go Client Package

- [ ] 3.1 Create `internal/cudgel/client.go` — defines `Client` interface, `Symbol` and `Edge` structs
- [ ] 3.2 Implement `HTTPClient` struct with `NewHTTPClient(endpoint string) *HTTPClient`
- [ ] 3.3 Implement `SemanticSearch(ctx, query string, limit int) ([]Symbol, error)` — POSTs to `/search`, parses response
- [ ] 3.4 Implement `GraphTraversal(ctx, symbol string, depth int) ([]Edge, error)` — POSTs to `/graph`, parses response
- [ ] 3.5 Implement no-op `NopClient` that returns empty results (used when `CUDGEL_ENDPOINT` is unset)
- [ ] 3.6 Write unit tests in `internal/cudgel/client_test.go` using an `httptest.Server`

## 4. Proto and Gateway Changes

- [ ] 4.1 Add `SemanticSearch` RPC, `SemanticSearchRequest`, `SemanticSearchResponse`, and `CodeChunk` message to the `AgentSidecarService` proto file
- [ ] 4.2 Run `buf generate` to regenerate Go bindings
- [ ] 4.3 Implement `SemanticSearch` handler in `internal/sidecar/gateway.go` — reads `CUDGEL_ENDPOINT` env var, instantiates client, delegates to `internal/cudgel`
- [ ] 4.4 Add limit clamping: default 10 if unset, max 50
- [ ] 4.5 Return empty response (no error) when `CUDGEL_ENDPOINT` is unset or cudgel returns error
- [ ] 4.6 Write gateway tests for `SemanticSearch` in `internal/sidecar/gateway_test.go`

## 5. Hydration Context Seeding

- [ ] 5.1 Add `SeedCodebaseContext(ctx context.Context, prompt, agentType string) error` method to `Hydrator` in `internal/hydration/hydrator.go`
- [ ] 5.2 Implement: read `CUDGEL_ENDPOINT` env var; if unset, return nil immediately
- [ ] 5.3 Implement: call `cudgel.SemanticSearch` with the prompt, K=10 (or K=20 for senior/orchestrator agent type), 5-second context timeout
- [ ] 5.4 Implement: on success, write `.aot/context/codebase.md` with formatted markdown output (header + symbol entries, max 4,000 tokens)
- [ ] 5.5 Implement: on error or timeout, log warning and return nil (graceful degradation)
- [ ] 5.6 Call `SeedCodebaseContext` in `Hydrator.Run()` after `.aot` directory structure creation and before devbox setup
- [ ] 5.7 Create `.aot/context/` directory in `Hydrator.Run()` alongside existing `.aot/traces/` and `.aot/logs/`
- [ ] 5.8 Write unit tests for `SeedCodebaseContext` in `internal/hydration/hydrator_test.go` (mock cudgel client)

## 6. SearchPastWork Extension

- [ ] 6.1 Add `SOURCE_CODE = 3` value to `SourceFilter` enum in the search proto
- [ ] 6.2 Run `buf generate` to regenerate
- [ ] 6.3 Add `SOURCE_CODE` branch in the `SearchPastWork` RPC handler — instantiates `internal/cudgel.HTTPClient` from env, forwards query, maps results to existing `SearchResult` message format
- [ ] 6.4 Map cudgel `kind` → `node_type`, cudgel `snippet` → `chunk_text`, cudgel `score` → `similarity_score`
- [ ] 6.5 Return empty results (no error) when cudgel is unreachable
- [ ] 6.6 Write tests for the `SOURCE_CODE` path

## 7. Helm Chart

- [ ] 7.1 Create `deploy/helm/aot/templates/cudgel.yaml` with Deployment and Service (pattern: match soft-serve.yaml structure)
- [ ] 7.2 Add `cudgel.enabled`, `cudgel.image`, `cudgel.endpoint`, `cudgel.resources` fields to `deploy/helm/aot/values.yaml`
- [ ] 7.3 Add `cudgel.enabled: true` with appropriate defaults to `deploy/helm/aot/dev-values.yaml`
- [ ] 7.4 Set `CUDGEL_DATABASE_URL` env var in cudgel Deployment from the `cudgel-db-credentials` Secret
- [ ] 7.5 Pass `CUDGEL_ENDPOINT` env var to worker and controller Deployments (points to cudgel Service)
- [ ] 7.6 Pass `CUDGEL_ENDPOINT` env var to the hydration init-container spec

## 8. Integration and Smoke Test

- [ ] 8.1 Run `POST /index` against the uncworks repo in the local k0s cluster to populate the cudgel index
- [ ] 8.2 Verify `POST /search` returns expected symbols for a sample query
- [ ] 8.3 Start a test agent run and confirm `.aot/context/codebase.md` is written in the workspace
- [ ] 8.4 Verify `SemanticSearch` RPC is callable from an agent and returns results
- [ ] 8.5 Test graceful degradation: stop cudgel pod, verify agent run still starts and completes without errors
