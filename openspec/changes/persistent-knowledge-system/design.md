## Context

The AOT system runs agent workflows that produce rich artifacts -- git diffs, execution logs, trace spans, tool call records -- but all of this data is ephemeral. PVCs have a 7-day TTL. Once cleaned up, the system has no memory of what happened. Every new run starts from scratch, even when past runs have solved similar problems or modified the same repositories. The existing `internal/brain/store.go` provides a PostgreSQL-backed store with an `agent_states` table that tracks run metadata (phase, prompt, repo URL, branch, trace ID). This change extends that foundation into a full knowledge system.

## Goals / Non-Goals

**Goals:**
- All run artifacts (logs, diffs, trace spans) are persisted permanently in PostgreSQL
- Code diffs and log segments are chunked, embedded, and indexed for semantic search
- New runs receive relevant context from past work before the agent starts
- A gRPC endpoint allows searching all past work by natural language
- Embedding runs locally via ONNX with no external API dependency
- The embedding pipeline does not block the main workflow -- it runs asynchronously after completion

**Non-Goals:**
- Real-time streaming of embeddings during a run (batch after completion is sufficient)
- Fine-tuning or training custom embedding models
- Replacing the existing agent_states table (we extend it with foreign-keyed tables)
- Building a full RAG pipeline with LLM-generated summaries (future work)
- UI for browsing knowledge (future work; only the search API is in scope)

## Decisions

### 1. Extend existing PostgreSQL + pgvector (same as cudgel)

The LiteLLM PostgreSQL instance already runs in the k0s cluster. We add the `pgvector` extension to it and create new tables. This avoids deploying a separate vector database (Chroma, Milvus, etc.) and keeps the operational surface small. The `internal/brain/store.go` already uses `pgx/v5` and `pgxpool` -- we extend the same `Store` struct.

**Alternative considered:** Deploy Chroma (like VectorCode uses). Rejected -- adds another stateful service to manage, and pgvector is proven in cudgel with the same embedding model.

### 2. all-MiniLM-L6-v2 via ONNX for embeddings (384-dim vectors)

The same model cudgel uses. It produces 384-dimensional vectors, runs locally via ONNX Runtime, and requires no API calls. The model file is ~80MB and can be bundled into the controlplane image or loaded from a PVC at startup.

**Alternative considered:** Use an external embedding API (OpenAI, Cohere). Rejected -- adds latency, cost, and an external dependency for an offline-capable system.

### 3. Tree-sitter for code-aware chunking

Diffs are code. Naive line-based or token-count chunking breaks semantic boundaries. Tree-sitter parses code into AST nodes, and we chunk at function/class/method boundaries. This is the same approach used by cudgel, VectorCode, and osgrep. We use the Go tree-sitter bindings (`github.com/smacker/go-tree-sitter`).

For log/trace content (non-code), we fall back to paragraph-level chunking with overlap.

### 4. Dual-index strategy (osgrep-inspired)

Two separate pgvector tables with HNSW indexes:
- `code_chunks`: embedded code diffs and file changes, with metadata (file path, language, repo URL, run ID, function name)
- `trace_chunks`: embedded log segments and trace spans, with metadata (severity, activity name, run ID, timestamp range)

Searching code and traces separately allows different ranking strategies and avoids polluting code search results with log noise. Queries can target one or both indexes.

### 5. HNSW index with cosine distance

HNSW (Hierarchical Navigable Small Worlds) provides fast approximate nearest-neighbor search. pgvector supports it natively. Cosine distance is standard for sentence-transformer models. Index parameters: `m=16, ef_construction=200` (same as cudgel defaults).

### 6. Structural boosting for code chunks

When embedding code chunks, we assign a `boost` score based on AST node type:
- Function/method definitions: boost 1.0
- Class/struct definitions: boost 0.9
- Import/require statements: boost 0.3
- Whitespace-only changes: boost 0.1

The boost is stored alongside the embedding and used as a multiplier during search ranking. This prevents trivial changes from dominating results.

### 7. Context hydration as a Temporal activity

A new `HydrateContext` activity runs before `StartAgent` in the workflow. It:
1. Takes the run's prompt and repo URL as input
2. Embeds the prompt using the same ONNX model
3. Queries both `code_chunks` and `trace_chunks` tables for top-K similar results (K=10 per index)
4. Formats results into a markdown context file
5. Writes the file to the agent's workspace at `.aot/context/past-work.md`

The agent's system prompt already reads files from the workspace. This file provides relevant history without modifying the agent harness.

**Alternative considered:** Inject into the system prompt directly. Rejected -- the context can be large and the system prompt has token limits. A file in the workspace lets the agent decide how much to use.

### 8. Asynchronous embedding after run completion

The `PersistRunData` activity runs after the agent completes (in the workflow's cleanup/defer block). It:
1. Reads logs, diffs, and spans from the workspace/PVC
2. Writes raw data to `run_logs`, `run_diffs`, `run_spans` tables
3. Triggers the embedding pipeline (chunk, embed, insert into pgvector)

This does NOT block the workflow completion signal. The run is marked complete, and embedding happens afterward. If embedding fails, the raw data is still persisted and can be re-embedded later.

### 9. Semantic search via gRPC

A new `SearchPastWork` RPC on the existing AOT API service. Request takes a natural language query string, optional repo URL filter, optional time range, and result limit. Response returns ranked results with: chunk text, source (code vs trace), run ID, file path, similarity score, and metadata.

The search handler embeds the query, runs parallel searches against both indexes, merges and re-ranks results, and returns the top-N.

## Schema

### New tables (added to brain store migration)

```sql
-- Raw run artifacts
CREATE TABLE run_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    log_type TEXT NOT NULL,  -- 'stdout', 'stderr', 'system'
    content TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

CREATE TABLE run_diffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    file_path TEXT NOT NULL,
    diff_content TEXT NOT NULL,
    language TEXT,
    tool_call_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE run_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    span_name TEXT NOT NULL,
    trace_id TEXT,
    parent_span_id UUID,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    attributes JSONB,
    status TEXT
);

-- Vector-indexed chunks
CREATE TABLE code_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    diff_id UUID REFERENCES run_diffs(id),
    chunk_text TEXT NOT NULL,
    file_path TEXT,
    language TEXT,
    node_type TEXT,        -- 'function', 'class', 'method', 'block'
    repo_url TEXT,
    boost REAL NOT NULL DEFAULT 1.0,
    embedding vector(384) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trace_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    span_id UUID REFERENCES run_spans(id),
    chunk_text TEXT NOT NULL,
    chunk_type TEXT,       -- 'log', 'error', 'activity', 'decision'
    severity TEXT,
    repo_url TEXT,
    embedding vector(384) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW indexes for fast approximate nearest-neighbor search
CREATE INDEX idx_code_chunks_embedding ON code_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX idx_trace_chunks_embedding ON trace_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- Supporting indexes
CREATE INDEX idx_run_logs_run_id ON run_logs(agent_run_id);
CREATE INDEX idx_run_diffs_run_id ON run_diffs(agent_run_id);
CREATE INDEX idx_run_spans_run_id ON run_spans(agent_run_id);
CREATE INDEX idx_code_chunks_run_id ON code_chunks(agent_run_id);
CREATE INDEX idx_code_chunks_repo ON code_chunks(repo_url);
CREATE INDEX idx_trace_chunks_run_id ON trace_chunks(agent_run_id);
```

## Risks / Trade-offs

- **[Risk] pgvector HNSW index build time grows with data** -- For the expected volume (hundreds to low thousands of runs), this is not a concern. If it becomes one, we can partition by time or switch to IVFFlat for faster builds at the cost of recall.
- **[Risk] ONNX model adds ~80MB to the controlplane image** -- Acceptable. Alternatively, load from a PVC or init container. We start with bundling for simplicity.
- **[Risk] Tree-sitter Go bindings (CGO) complicate the build** -- The `go-tree-sitter` library requires CGO. The existing build already uses CGO for other dependencies. If this becomes a problem, we can shell out to a tree-sitter CLI instead.
- **[Risk] Context hydration adds latency to run startup** -- The pgvector HNSW query is sub-100ms for our data sizes. The activity has a 5-second timeout. If it fails or times out, the run proceeds without context (graceful degradation).
- **[Risk] Embedding quality for diffs** -- Diffs (unified diff format) are not natural language. The embedding model may not produce ideal vectors. Mitigation: we embed the actual code content (post-change), not the diff markers. For significant quality issues, we can add a re-ranking step later.
- **[Trade-off] Dual index vs single index** -- Two tables add schema complexity but provide cleaner search results and allow independent tuning. Worth the complexity for the separation of concerns.
