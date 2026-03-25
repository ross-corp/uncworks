// Package brain implements the PostgreSQL-backed shared state store for AOT.
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AgentState represents the persistent state of an agent run in the shared brain.
type AgentState struct {
	ID          string
	AgentRunID  string
	Phase       string
	Message     string
	Prompt      string
	RepoURL     string
	Branch      string
	TraceID     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	Metadata    map[string]string
}

// RunDiff represents a file diff produced by an agent run.
type RunDiff struct {
	ID         string
	AgentRunID string
	SpanID     string
	FilePath   string
	Patch      string
	CreatedAt  time.Time
}

// TraceSpan represents a trace span from an agent run.
type TraceSpan struct {
	ID         string
	AgentRunID string
	ParentID   string
	Name       string
	Type       string
	StartTime  time.Time
	EndTime    *time.Time
	Metadata   map[string]interface{}
	CreatedAt  time.Time
}

// CodeChunkRecord represents an embedded code chunk for pgvector storage.
type CodeChunkRecord struct {
	AgentRunID string
	DiffID     string
	ChunkText  string
	FilePath   string
	Language   string
	NodeType   string
	RepoURL    string
	Boost      float32
	Embedding  []float32
}

// TraceChunkRecord represents an embedded trace chunk for pgvector storage.
type TraceChunkRecord struct {
	AgentRunID string
	SpanID     string
	ChunkText  string
	ChunkType  string
	Severity   string
	RepoURL    string
	Embedding  []float32
}

// CodeChunkResult represents a code chunk search result from pgvector.
type CodeChunkResult struct {
	ID         string
	AgentRunID string
	ChunkText  string
	FilePath   string
	Language   string
	NodeType   string
	RepoURL    string
	Boost      float32
	Similarity float64
	CreatedAt  time.Time
}

// TraceChunkResult represents a trace chunk search result from pgvector.
type TraceChunkResult struct {
	ID         string
	AgentRunID string
	ChunkText  string
	ChunkType  string
	Severity   string
	RepoURL    string
	Similarity float64
	CreatedAt  time.Time
}

// Store provides access to the shared brain database.
type Store struct {
	pool          *pgxpool.Pool
	pgvectorReady bool
}

// NewStore creates a new Store with the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Migrate creates the required database tables.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS agent_states (
			id TEXT PRIMARY KEY,
			agent_run_id TEXT UNIQUE NOT NULL,
			phase TEXT NOT NULL DEFAULT 'Pending',
			message TEXT,
			prompt TEXT NOT NULL,
			repo_url TEXT NOT NULL,
			branch TEXT,
			trace_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_agent_states_phase ON agent_states(phase);
	`)
	if err != nil {
		return err
	}

	// Knowledge system tables: run artifacts
	_, err = s.pool.Exec(ctx, `
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

		CREATE INDEX IF NOT EXISTS idx_run_logs_run_id ON run_logs(agent_run_id);
		CREATE INDEX IF NOT EXISTS idx_run_diffs_run_id ON run_diffs(agent_run_id);
		CREATE INDEX IF NOT EXISTS idx_run_spans_run_id ON run_spans(agent_run_id);
	`)
	if err != nil {
		return err
	}

	// Knowledge system tables: vector embeddings (requires pgvector extension)
	if err := s.migrateVectorTables(ctx); err != nil {
		log.Printf("WARNING: pgvector tables not created (pgvector extension may not be available): %v", err)
		// Graceful degradation: run artifacts are stored, but embedding/search is unavailable.
	}

	return nil
}

// migrateVectorTables creates pgvector-dependent tables. Fails gracefully if pgvector is not installed.
func (s *Store) migrateVectorTables(ctx context.Context) error {
	// Try to enable the extension
	_, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return fmt.Errorf("enable pgvector: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
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

		CREATE INDEX IF NOT EXISTS idx_code_chunks_run_id ON code_chunks(agent_run_id);
		CREATE INDEX IF NOT EXISTS idx_code_chunks_repo ON code_chunks(repo_url);
		CREATE INDEX IF NOT EXISTS idx_trace_chunks_run_id ON trace_chunks(agent_run_id);
	`)
	if err != nil {
		return fmt.Errorf("create vector tables: %w", err)
	}

	// HNSW indexes for fast approximate nearest-neighbor search
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_code_chunks_embedding ON code_chunks
			USING hnsw (embedding vector_cosine_ops)
			WITH (m = 16, ef_construction = 200);

		CREATE INDEX IF NOT EXISTS idx_trace_chunks_embedding ON trace_chunks
			USING hnsw (embedding vector_cosine_ops)
			WITH (m = 16, ef_construction = 200);
	`)
	if err != nil {
		return fmt.Errorf("create HNSW indexes: %w", err)
	}

	s.pgvectorReady = true
	return nil
}

// PgvectorReady returns whether the pgvector extension and tables are available.
func (s *Store) PgvectorReady() bool {
	return s.pgvectorReady
}

// SaveState upserts an agent state.
func (s *Store) SaveState(ctx context.Context, state *AgentState) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO agent_states (id, agent_run_id, phase, message, prompt, repo_url, branch, trace_id, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (agent_run_id)
		DO UPDATE SET phase = $3, message = $4, trace_id = $8, updated_at = $10, completed_at = $11
	`, state.ID, state.AgentRunID, state.Phase, state.Message, state.Prompt, state.RepoURL, state.Branch, state.TraceID, state.CreatedAt, state.UpdatedAt, state.CompletedAt)
	return err
}

// GetState retrieves an agent state by agent run ID.
func (s *Store) GetState(ctx context.Context, agentRunID string) (*AgentState, error) {
	state := &AgentState{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, agent_run_id, phase, message, prompt, repo_url, branch, trace_id, created_at, updated_at, completed_at
		FROM agent_states WHERE agent_run_id = $1
	`, agentRunID).Scan(
		&state.ID, &state.AgentRunID, &state.Phase, &state.Message,
		&state.Prompt, &state.RepoURL, &state.Branch, &state.TraceID,
		&state.CreatedAt, &state.UpdatedAt, &state.CompletedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("agent state not found: %s", agentRunID)
	}
	return state, err
}

// UpdatePhase updates the phase and message for an agent run.
func (s *Store) UpdatePhase(ctx context.Context, agentRunID, phase, message string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_states SET phase = $2, message = $3, updated_at = NOW() WHERE agent_run_id = $1
	`, agentRunID, phase, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent state not found: %s", agentRunID)
	}
	return nil
}

// ListByPhase returns all agent states matching the given phase.
func (s *Store) ListByPhase(ctx context.Context, phase string) ([]*AgentState, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_run_id, phase, message, prompt, repo_url, branch, trace_id, created_at, updated_at, completed_at
		FROM agent_states WHERE phase = $1 ORDER BY created_at
	`, phase)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*AgentState
	for rows.Next() {
		s := &AgentState{}
		if err := rows.Scan(
			&s.ID, &s.AgentRunID, &s.Phase, &s.Message,
			&s.Prompt, &s.RepoURL, &s.Branch, &s.TraceID,
			&s.CreatedAt, &s.UpdatedAt, &s.CompletedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

// --- Run Data Persistence (Knowledge System) ---

// SaveRunLog persists a run's log content to PostgreSQL.
func (s *Store) SaveRunLog(ctx context.Context, agentRunID, content string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO run_logs (agent_run_id, content) VALUES ($1, $2)
	`, agentRunID, content)
	return err
}

// SaveRunDiff persists a file diff from an agent run.
func (s *Store) SaveRunDiff(ctx context.Context, agentRunID, spanID, filePath, patch string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO run_diffs (agent_run_id, span_id, file_path, patch) VALUES ($1, $2, $3, $4)
	`, agentRunID, spanID, filePath, patch)
	return err
}

// SaveRunSpan persists a trace span from an agent run.
func (s *Store) SaveRunSpan(ctx context.Context, agentRunID string, span TraceSpan) error {
	metadataJSON, err := json.Marshal(span.Metadata)
	if err != nil {
		return fmt.Errorf("marshal span metadata: %w", err)
	}

	var parentID *string
	if span.ParentID != "" {
		parentID = &span.ParentID
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO run_spans (agent_run_id, parent_id, name, type, start_time, end_time, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, agentRunID, parentID, span.Name, span.Type, span.StartTime, span.EndTime, metadataJSON)
	return err
}

// GetRunLogs retrieves concatenated log content for an agent run.
func (s *Store) GetRunLogs(ctx context.Context, agentRunID string) (string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT content FROM run_logs WHERE agent_run_id = $1 ORDER BY created_at
	`, agentRunID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return "", err
		}
		if result != "" {
			result += "\n"
		}
		result += content
	}
	return result, rows.Err()
}

// GetRunDiffs retrieves all diffs for an agent run.
func (s *Store) GetRunDiffs(ctx context.Context, agentRunID string) ([]RunDiff, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_run_id, span_id, file_path, patch, created_at
		FROM run_diffs WHERE agent_run_id = $1 ORDER BY created_at
	`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diffs []RunDiff
	for rows.Next() {
		var d RunDiff
		var spanID *string
		if err := rows.Scan(&d.ID, &d.AgentRunID, &spanID, &d.FilePath, &d.Patch, &d.CreatedAt); err != nil {
			return nil, err
		}
		if spanID != nil {
			d.SpanID = *spanID
		}
		diffs = append(diffs, d)
	}
	return diffs, rows.Err()
}

// GetRunSpans retrieves all trace spans for an agent run.
func (s *Store) GetRunSpans(ctx context.Context, agentRunID string) ([]TraceSpan, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_run_id, parent_id, name, type, start_time, end_time, metadata, created_at
		FROM run_spans WHERE agent_run_id = $1 ORDER BY start_time
	`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []TraceSpan
	for rows.Next() {
		var sp TraceSpan
		var parentID *string
		var spanType *string
		var metadataJSON []byte
		if err := rows.Scan(&sp.ID, &sp.AgentRunID, &parentID, &sp.Name, &spanType, &sp.StartTime, &sp.EndTime, &metadataJSON, &sp.CreatedAt); err != nil {
			return nil, err
		}
		if parentID != nil {
			sp.ParentID = *parentID
		}
		if spanType != nil {
			sp.Type = *spanType
		}
		if metadataJSON != nil {
			sp.Metadata = make(map[string]interface{})
			if err := json.Unmarshal(metadataJSON, &sp.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal span %s metadata: %w", sp.ID, err)
			}
		}
		spans = append(spans, sp)
	}
	return spans, rows.Err()
}

// --- Vector Embedding Storage (Knowledge System) ---

// SaveCodeChunks batch-inserts embedded code chunks into pgvector atomically.
func (s *Store) SaveCodeChunks(ctx context.Context, chunks []CodeChunkRecord) error {
	if !s.pgvectorReady {
		return fmt.Errorf("pgvector not available")
	}
	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, c := range chunks {
		embeddingStr := float32SliceToVectorLiteral(c.Embedding)
		_, err := tx.Exec(ctx, `
			INSERT INTO code_chunks (agent_run_id, diff_id, chunk_text, file_path, language, node_type, repo_url, boost, embedding)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, c.AgentRunID, nilIfEmpty(c.DiffID), c.ChunkText, c.FilePath, c.Language, c.NodeType, c.RepoURL, c.Boost, embeddingStr)
		if err != nil {
			return fmt.Errorf("insert code chunk: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// SaveTraceChunks batch-inserts embedded trace chunks into pgvector atomically.
func (s *Store) SaveTraceChunks(ctx context.Context, chunks []TraceChunkRecord) error {
	if !s.pgvectorReady {
		return fmt.Errorf("pgvector not available")
	}
	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, c := range chunks {
		embeddingStr := float32SliceToVectorLiteral(c.Embedding)
		_, err := tx.Exec(ctx, `
			INSERT INTO trace_chunks (agent_run_id, span_id, chunk_text, chunk_type, severity, repo_url, embedding)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, c.AgentRunID, nilIfEmpty(c.SpanID), c.ChunkText, c.ChunkType, c.Severity, c.RepoURL, embeddingStr)
		if err != nil {
			return fmt.Errorf("insert trace chunk: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// --- Vector Search (Knowledge System) ---

// SearchCodeChunks performs a cosine similarity search on code_chunks.
func (s *Store) SearchCodeChunks(ctx context.Context, queryVec []float32, repoURL string, limit int, createdAfter, createdBefore *time.Time) ([]CodeChunkResult, error) {
	if !s.pgvectorReady {
		return nil, fmt.Errorf("pgvector not available")
	}
	if limit <= 0 {
		limit = 10
	}

	vecStr := float32SliceToVectorLiteral(queryVec)

	query := `
		SELECT id, agent_run_id, chunk_text, file_path, language, node_type, repo_url, boost,
			1 - (embedding <=> $1::vector) AS similarity, created_at
		FROM code_chunks
		WHERE 1=1
	`
	args := []interface{}{vecStr}
	argIdx := 2

	if repoURL != "" {
		query += fmt.Sprintf(" AND repo_url = $%d", argIdx)
		args = append(args, repoURL)
		argIdx++
	}
	if createdAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *createdAfter)
		argIdx++
	}
	if createdBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *createdBefore)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY embedding <=> $1::vector LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CodeChunkResult
	for rows.Next() {
		var r CodeChunkResult
		if err := rows.Scan(&r.ID, &r.AgentRunID, &r.ChunkText, &r.FilePath, &r.Language, &r.NodeType, &r.RepoURL, &r.Boost, &r.Similarity, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchTraceChunks performs a cosine similarity search on trace_chunks.
func (s *Store) SearchTraceChunks(ctx context.Context, queryVec []float32, repoURL string, limit int, createdAfter, createdBefore *time.Time) ([]TraceChunkResult, error) {
	if !s.pgvectorReady {
		return nil, fmt.Errorf("pgvector not available")
	}
	if limit <= 0 {
		limit = 10
	}

	vecStr := float32SliceToVectorLiteral(queryVec)

	query := `
		SELECT id, agent_run_id, chunk_text, chunk_type, severity, repo_url,
			1 - (embedding <=> $1::vector) AS similarity, created_at
		FROM trace_chunks
		WHERE 1=1
	`
	args := []interface{}{vecStr}
	argIdx := 2

	if repoURL != "" {
		query += fmt.Sprintf(" AND repo_url = $%d", argIdx)
		args = append(args, repoURL)
		argIdx++
	}
	if createdAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *createdAfter)
		argIdx++
	}
	if createdBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *createdBefore)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY embedding <=> $1::vector LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TraceChunkResult
	for rows.Next() {
		var r TraceChunkResult
		if err := rows.Scan(&r.ID, &r.AgentRunID, &r.ChunkText, &r.ChunkType, &r.Severity, &r.RepoURL, &r.Similarity, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// --- Helpers ---

// float32SliceToVectorLiteral converts a float32 slice to a pgvector literal string like "[0.1,0.2,...]".
func float32SliceToVectorLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%g", f)
	}
	s += "]"
	return s
}

// nilIfEmpty returns nil for empty strings, useful for nullable FK columns.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
