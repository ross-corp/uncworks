-- 001_knowledge_schema.sql
-- Persistent Knowledge System: run artifacts, vector embeddings, and semantic search.
-- Depends on existing agent_states table from internal/brain/store.go Migrate().

-- Enable pgvector extension for embedding storage and similarity search.
CREATE EXTENSION IF NOT EXISTS vector;

-- Raw run artifacts --

CREATE TABLE IF NOT EXISTS run_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS run_diffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    span_id TEXT,
    file_path TEXT NOT NULL,
    patch TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS run_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    parent_id UUID,
    name TEXT NOT NULL,
    type TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vector-indexed chunks --

CREATE TABLE IF NOT EXISTS code_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    diff_id UUID REFERENCES run_diffs(id),
    chunk_text TEXT NOT NULL,
    file_path TEXT,
    language TEXT,
    node_type TEXT,
    repo_url TEXT,
    boost REAL NOT NULL DEFAULT 1.0,
    embedding vector(384) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trace_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id TEXT NOT NULL REFERENCES agent_states(agent_run_id),
    span_id UUID REFERENCES run_spans(id),
    chunk_text TEXT NOT NULL,
    chunk_type TEXT,
    severity TEXT,
    repo_url TEXT,
    embedding vector(384) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW indexes for fast approximate nearest-neighbor search (cosine distance) --

CREATE INDEX IF NOT EXISTS idx_code_chunks_embedding ON code_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

CREATE INDEX IF NOT EXISTS idx_trace_chunks_embedding ON trace_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- Supporting indexes --

CREATE INDEX IF NOT EXISTS idx_run_logs_run_id ON run_logs(agent_run_id);
CREATE INDEX IF NOT EXISTS idx_run_diffs_run_id ON run_diffs(agent_run_id);
CREATE INDEX IF NOT EXISTS idx_run_spans_run_id ON run_spans(agent_run_id);
CREATE INDEX IF NOT EXISTS idx_code_chunks_run_id ON code_chunks(agent_run_id);
CREATE INDEX IF NOT EXISTS idx_code_chunks_repo ON code_chunks(repo_url);
CREATE INDEX IF NOT EXISTS idx_trace_chunks_run_id ON trace_chunks(agent_run_id);
