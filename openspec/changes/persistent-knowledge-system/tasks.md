## 1. PostgreSQL Deployment and Schema

- [ ] 1.1 Enable pgvector extension on the existing LiteLLM PostgreSQL instance (`CREATE EXTENSION IF NOT EXISTS vector`)
- [ ] 1.2 Add pgvector extension init to Helm chart (init container or SQL ConfigMap that runs on pod startup)
- [ ] 1.3 Extend `Store.Migrate()` in `internal/brain/store.go` with `CREATE TABLE IF NOT EXISTS` for `run_logs`, `run_diffs`, `run_spans`
- [ ] 1.4 Extend `Store.Migrate()` with `CREATE TABLE IF NOT EXISTS` for `code_chunks` and `trace_chunks` (with `vector(384)` columns)
- [ ] 1.5 Add HNSW indexes on `code_chunks.embedding` and `trace_chunks.embedding` columns (cosine distance, m=16, ef_construction=200)
- [ ] 1.6 Add supporting indexes on foreign keys and filter columns (`agent_run_id`, `repo_url`)
- [ ] 1.7 Verify migration is idempotent -- running `Migrate()` twice does not error or lose data
- [ ] 1.8 Add unit tests for migration (use testcontainers or pgx mock)

## 2. Run Data Persistence

- [ ] 2.1 Add `SaveRunLogs(ctx, agentRunID, logs []RunLog) error` method to brain Store
- [ ] 2.2 Add `SaveRunDiffs(ctx, agentRunID, diffs []RunDiff) error` method to brain Store
- [ ] 2.3 Add `SaveRunSpans(ctx, agentRunID, spans []RunSpan) error` method to brain Store
- [ ] 2.4 Add `GetRunLogs(ctx, agentRunID) ([]RunLog, error)` method to brain Store
- [ ] 2.5 Add `GetRunDiffs(ctx, agentRunID) ([]RunDiff, error)` method to brain Store
- [ ] 2.6 Add `GetRunSpans(ctx, agentRunID) ([]RunSpan, error)` method to brain Store
- [ ] 2.7 Define Go structs: `RunLog`, `RunDiff`, `RunSpan` with appropriate fields and JSON/JSONB support
- [ ] 2.8 Add `PersistRunData` Temporal activity in `internal/temporal/activities.go` that reads workspace artifacts and calls brain Store save methods
- [ ] 2.9 Wire `PersistRunData` activity into the workflow cleanup/defer block in `internal/temporal/workflow.go` (runs after agent completion)
- [ ] 2.10 Add unit tests for each brain Store CRUD method

## 3. Embedding Pipeline

- [ ] 3.1 Create `internal/brain/embedder.go` with `Embedder` struct that loads all-MiniLM-L6-v2 ONNX model
- [ ] 3.2 Implement `Embedder.Embed(text string) ([]float32, error)` using ONNX Runtime Go bindings (or shelling out to a Python sidecar)
- [ ] 3.3 Implement tree-sitter chunking: `ChunkCode(content string, language string) ([]CodeChunk, error)` using `go-tree-sitter`
- [ ] 3.4 Implement paragraph chunking for non-code content: `ChunkText(content string, maxTokens int, overlap int) ([]TextChunk, error)`
- [ ] 3.5 Implement structural boost assignment based on AST node type (function=1.0, class=0.9, import=0.3, whitespace=0.1, other=0.7)
- [ ] 3.6 Add `SaveCodeChunks(ctx, chunks []CodeChunkRecord) error` method to brain Store (batch insert with embeddings)
- [ ] 3.7 Add `SaveTraceChunks(ctx, chunks []TraceChunkRecord) error` method to brain Store (batch insert with embeddings)
- [ ] 3.8 Create `EmbedAndStore` orchestration function that: reads run_diffs and run_logs, chunks them, embeds chunks, saves to pgvector tables
- [ ] 3.9 Add `EmbedRunData` Temporal activity that calls `EmbedAndStore` for a completed run
- [ ] 3.10 Wire `EmbedRunData` activity after `PersistRunData` in the workflow cleanup block
- [ ] 3.11 Bundle all-MiniLM-L6-v2 ONNX model file (~80MB) into the controlplane Docker image (or mount from PVC)
- [ ] 3.12 Add unit tests for tree-sitter chunking (Go, Python, TypeScript test cases)
- [ ] 3.13 Add unit tests for paragraph chunking (overlap, max tokens)
- [ ] 3.14 Add integration test for embed-and-store pipeline (requires PostgreSQL with pgvector)

## 4. Context Hydration Activity

- [ ] 4.1 Create `internal/brain/hydrator.go` with `Hydrator` struct
- [ ] 4.2 Implement `Hydrator.QueryRelevantContext(ctx, prompt string, repoURL string, agentType string) ([]SearchResult, error)` -- embeds prompt, queries both pgvector tables
- [ ] 4.3 Implement `Hydrator.FormatContextFile(results []SearchResult) (string, error)` -- produces markdown with headers, source metadata, and chunk content (capped at 8,000 tokens)
- [ ] 4.4 Implement `Hydrator.WriteContextFile(workspacePath string, content string) error` -- writes to `.aot/context/past-work.md`
- [ ] 4.5 Add senior agent logic: increase top-K from 10 to 25 when `agentType` is "senior" or "orchestrator"
- [ ] 4.6 Add `HydrateContext` Temporal activity in `internal/temporal/activities.go` with 5-second timeout
- [ ] 4.7 Wire `HydrateContext` activity into the workflow between workspace provisioning and agent startup (before `StartAgent`)
- [ ] 4.8 Ensure graceful degradation: if HydrateContext fails or times out, log warning and proceed without context
- [ ] 4.9 Add unit tests for context formatting (markdown structure, token limit)
- [ ] 4.10 Add unit tests for hydration query (mock pgvector responses)

## 5. Semantic Search API Endpoint

- [ ] 5.1 Add `SearchPastWork` RPC to `proto/aot/api/v1/api.proto` with request/response messages (query, repo_url, source_filter, created_after, created_before, limit)
- [ ] 5.2 Add `SearchResult` message to proto with fields: chunk_text, source_type, similarity_score, run_id, file_path, language, node_type, chunk_type, severity, repo_url, created_at
- [ ] 5.3 Regenerate Go proto types (`buf generate`)
- [ ] 5.4 Regenerate TypeScript proto types (`buf generate`)
- [ ] 5.5 Create `internal/brain/search.go` with `Searcher` struct
- [ ] 5.6 Implement `Searcher.Search(ctx, query SearchQuery) ([]SearchResult, error)` -- embeds query, runs parallel pgvector searches, merges and re-ranks by boosted similarity
- [ ] 5.7 Add `SearchPastWork` handler to `internal/server/grpc.go` that validates input, calls Searcher, and returns results
- [ ] 5.8 Clamp limit to max 100, default to 10 if not specified
- [ ] 5.9 Add unit tests for search result merging and ranking
- [ ] 5.10 Add unit tests for gRPC handler (mock searcher)

## 6. Brain Store Integration

- [ ] 6.1 Add `Embedder` field to brain `Store` struct (or pass as dependency)
- [ ] 6.2 Add vector search query methods: `SearchCodeChunks(ctx, queryVec []float32, repoURL string, limit int) ([]CodeChunkResult, error)`
- [ ] 6.3 Add vector search query methods: `SearchTraceChunks(ctx, queryVec []float32, repoURL string, limit int) ([]TraceChunkResult, error)`
- [ ] 6.4 Add time-range filter support to vector search methods (created_after, created_before)
- [ ] 6.5 Ensure Store initializes pgvector extension check on startup (fail fast if pgvector is not available)
- [ ] 6.6 Register new Temporal activities (`PersistRunData`, `EmbedRunData`, `HydrateContext`) on the worker
- [ ] 6.7 Wire brain Store with embedder into the Temporal worker and activity constructors in `cmd/controlplane/main.go` (or equivalent entrypoint)

## 7. UI: Search Past Work from Command Palette

- [ ] 7.1 Add search input to web dashboard command palette (or a dedicated search page)
- [ ] 7.2 Wire search input to `SearchPastWork` gRPC endpoint via ConnectRPC client
- [ ] 7.3 Display search results with chunk text, source type badge (code/trace), similarity score, and run ID link
- [ ] 7.4 Add repo URL and time range filters to search UI
- [ ] 7.5 Add loading and empty state handling

## 8. End-to-End Tests

- [ ] 8.1 E2E test: create a run via API, verify run data is persisted to `run_logs` and `run_diffs` tables after completion
- [ ] 8.2 E2E test: verify embeddings are generated and stored in `code_chunks` and `trace_chunks` after a run completes
- [ ] 8.3 E2E test: create two runs, verify the second run's HydrateContext activity finds relevant context from the first
- [ ] 8.4 E2E test: call `SearchPastWork` API with a query related to a completed run, verify relevant results are returned
- [ ] 8.5 E2E test: verify search with repo_url filter only returns results from that repo
- [ ] 8.6 Add `test:e2e:knowledge` Taskfile target

## 9. Verification

- [ ] 9.1 Run `task test:go` -- all unit tests pass (including new brain store, embedder, hydrator, search tests)
- [ ] 9.2 Run `task test:contract` -- contract tests pass
- [ ] 9.3 Run `task test:temporal` -- workflow tests pass with PersistRunData, EmbedRunData, and HydrateContext activities
- [ ] 9.4 Run `task test:e2e:knowledge` -- knowledge system E2E tests pass
- [ ] 9.5 Verify pgvector HNSW index is created and functional (`\di+` in psql)
- [ ] 9.6 Verify context hydration file is written to agent workspace on run startup (inspect `.aot/context/past-work.md`)
- [ ] 9.7 Verify embedding pipeline does not block workflow completion (run completes before embeddings finish)
- [ ] 9.8 Deploy to dev cluster and manually test: create run, wait for completion, search for related content via API
