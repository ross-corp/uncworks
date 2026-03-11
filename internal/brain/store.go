// Package brain implements the PostgreSQL-backed shared state store for AOT.
package brain

import (
	"context"
	"fmt"
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

// Store provides access to the shared brain database.
type Store struct {
	pool *pgxpool.Pool
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
	return err
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
