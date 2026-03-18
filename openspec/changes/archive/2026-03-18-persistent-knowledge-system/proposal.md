## Why

Agent runs produce logs, diffs, traces, and file changes. Currently this data lives on PVCs with a 7-day TTL and is lost after cleanup. There is no way for the system to learn from past work -- each run starts from zero context. A bug the agent fixed last week is invisible to today's run. The senior agent cannot learn that junior runs on a particular repo tend to fail on test setup. All institutional knowledge evaporates.

Three open-source tools demonstrate how to do semantic code search over repositories:
- **cudgel** (by this project's author): TreeSitter AST extraction, sentence-transformers/all-MiniLM-L6-v2 via ONNX, pgvector in PostgreSQL, HNSW indexing
- **VectorCode**: tree-sitter chunking, Chroma vector DB, Neovim integration
- **osgrep**: tree-sitter chunking, local ONNX embeddings, dual-index (code vs docs), ColBERT reranking, structural boosting

The k0s cluster already has PostgreSQL available (used by LiteLLM). We can add pgvector to it. The existing `internal/brain/store.go` already provides a PostgreSQL-backed `Store` with an `agent_states` table -- this change extends that foundation with run artifacts, embeddings, and retrieval.

## What Changes

Three layers are added to turn ephemeral run data into persistent, searchable organizational memory:

### Layer 1: Persistent Storage (PostgreSQL)
- Extend the existing brain store PostgreSQL schema with tables for run logs, diffs (per tool call), and trace spans
- Data survives PVC cleanup and is queryable permanently
- New tables: `run_logs`, `run_diffs`, `run_spans` with foreign keys to `agent_states`

### Layer 2: Embedding & Indexing
- After each run completes, chunk diffs and significant log segments using tree-sitter for code-aware boundaries
- Embed chunks using a local ONNX model (all-MiniLM-L6-v2, same as cudgel) -- no external API dependency
- Store 384-dimensional embeddings in pgvector with HNSW index
- Dual index strategy (like osgrep): separate `code_chunks` from `trace_chunks` tables
- Structural boosting: weight function/class-level changes higher than whitespace or formatting

### Layer 3: Context Hydration
- When a new run starts, query pgvector: "what past work is relevant to this prompt + these repos?"
- Retrieve top-K relevant diffs, traces, and learnings
- Inject as context into the agent's workspace (a context file the agent reads at startup)
- Senior agents get richer context for decomposition planning
- The system learns: each completed run enriches the knowledge base

### Semantic Search API
- REST/gRPC endpoint for searching all past work by natural language query
- Powers both programmatic access and future UI integration (command palette search)

## Capabilities

### New Capabilities
- `persistent-run-storage`: PostgreSQL schema for runs, logs, diffs, and trace spans with proper indexes and foreign keys to existing `agent_states` table
- `vector-embedding-pipeline`: tree-sitter chunking, ONNX embedding with all-MiniLM-L6-v2, pgvector storage with HNSW indexing and dual-index strategy
- `context-hydration`: Temporal activity that queries relevant past work from pgvector, formats context, and writes it to the agent workspace before the agent starts
- `semantic-search-api`: gRPC endpoint for searching past work by natural language, returning ranked results with source metadata

### Modified Capabilities
- `brain-store`: Extend `internal/brain/store.go` with new tables, embedding storage methods, and vector search queries

## Impact

- `internal/brain/store.go` -- add migration for new tables (run_logs, run_diffs, run_spans, code_chunks, trace_chunks), add CRUD methods
- `internal/brain/embedder.go` -- new: tree-sitter chunking, ONNX model loading, embedding generation
- `internal/brain/hydrator.go` -- new: context hydration logic (query pgvector, format results, write context file)
- `internal/brain/search.go` -- new: semantic search query builder and result ranking
- `internal/temporal/activities.go` -- add PersistRunData and HydrateContext activities
- `internal/temporal/workflow.go` -- call HydrateContext before agent start, call PersistRunData after completion
- `internal/server/grpc.go` -- add SearchPastWork RPC handler
- `proto/aot/api/v1/api.proto` -- add SearchPastWork RPC and messages
- `deploy/helm/aot/templates/` -- PostgreSQL pgvector extension init, ONNX model sidecar or init container
- `e2e/` -- new knowledge system E2E tests
