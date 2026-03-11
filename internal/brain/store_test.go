package brain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("aot_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("Skipping: cannot start PostgreSQL container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestStore_Migrate(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Running migrate again should be idempotent
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Second Migrate failed: %v", err)
	}
}

func TestStore_SaveAndGetState(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	state := &AgentState{
		ID:         "state-1",
		AgentRunID: "ar-1",
		Phase:      "Pending",
		Message:    "Queued",
		Prompt:     "Fix the tests",
		RepoURL:    "https://github.com/example/repo.git",
		Branch:     "main",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.SaveState(ctx, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	got, err := store.GetState(ctx, "ar-1")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if got.AgentRunID != "ar-1" {
		t.Errorf("got AgentRunID %q, want %q", got.AgentRunID, "ar-1")
	}
	if got.Phase != "Pending" {
		t.Errorf("got Phase %q, want %q", got.Phase, "Pending")
	}
	if got.Prompt != "Fix the tests" {
		t.Errorf("got Prompt %q, want %q", got.Prompt, "Fix the tests")
	}
}

func TestStore_SaveState_Upsert(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	state := &AgentState{
		ID: "state-1", AgentRunID: "ar-1", Phase: "Pending",
		Prompt: "p", RepoURL: "r", CreatedAt: now, UpdatedAt: now,
	}
	if err := store.SaveState(ctx, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Upsert with new phase
	state.Phase = "Running"
	state.UpdatedAt = now.Add(time.Second)
	if err := store.SaveState(ctx, state); err != nil {
		t.Fatalf("SaveState upsert: %v", err)
	}

	got, _ := store.GetState(ctx, "ar-1")
	if got.Phase != "Running" {
		t.Errorf("expected upserted phase Running, got %q", got.Phase)
	}
}

func TestStore_GetState_NotFound(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	_, err := store.GetState(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent state")
	}
}

func TestStore_UpdatePhase(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := store.SaveState(ctx, &AgentState{
		ID: "state-2", AgentRunID: "ar-2", Phase: "Pending",
		Prompt: "p", RepoURL: "r", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	if err := store.UpdatePhase(ctx, "ar-2", "Running", "Pod started"); err != nil {
		t.Fatalf("UpdatePhase failed: %v", err)
	}

	got, _ := store.GetState(ctx, "ar-2")
	if got.Phase != "Running" {
		t.Errorf("got Phase %q, want %q", got.Phase, "Running")
	}
	if got.Message != "Pod started" {
		t.Errorf("got Message %q, want %q", got.Message, "Pod started")
	}
}

func TestStore_UpdatePhase_NotFound(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	err := store.UpdatePhase(ctx, "nonexistent", "Running", "msg")
	if err == nil {
		t.Fatal("expected error for nonexistent agent run")
	}
}

func TestStore_ListByPhase(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < 3; i++ {
		if err := store.SaveState(ctx, &AgentState{
			ID: fmt.Sprintf("state-%d", i), AgentRunID: fmt.Sprintf("ar-%d", i),
			Phase: "Pending", Prompt: "p", RepoURL: "r",
			CreatedAt: now.Add(time.Duration(i) * time.Second), UpdatedAt: now,
		}); err != nil {
			t.Fatalf("SaveState: %v", err)
		}
	}
	if err := store.SaveState(ctx, &AgentState{
		ID: "state-running", AgentRunID: "ar-running",
		Phase: "Running", Prompt: "p", RepoURL: "r",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	pending, err := store.ListByPhase(ctx, "Pending")
	if err != nil {
		t.Fatalf("ListByPhase failed: %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("got %d pending states, want 3", len(pending))
	}

	running, err := store.ListByPhase(ctx, "Running")
	if err != nil {
		t.Fatalf("ListByPhase Running: %v", err)
	}
	if len(running) != 1 {
		t.Errorf("got %d running states, want 1", len(running))
	}
}

// Queue functionality (Enqueue, Dequeue, QueueLength) removed — replaced by Temporal task queues.
