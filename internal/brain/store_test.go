package brain

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("AOT_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://localhost:5432/aot_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("Skipping: cannot connect to Postgres: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping: cannot ping Postgres: %v", err)
	}

	return pool
}

func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS agent_states, agent_queue")
}

func TestStore_Migrate(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	defer cleanupTables(t, pool)

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
	defer pool.Close()
	defer cleanupTables(t, pool)

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

func TestStore_UpdatePhase(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	defer cleanupTables(t, pool)

	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := store.SaveState(ctx, &AgentState{
		ID: "state-2", AgentRunID: "ar-2", Phase: "Pending", Prompt: "p", RepoURL: "r",
		CreatedAt: now, UpdatedAt: now,
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
	defer pool.Close()
	defer cleanupTables(t, pool)

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
	defer pool.Close()
	defer cleanupTables(t, pool)

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
		ID: "state-running", AgentRunID: "ar-running", Phase: "Running", Prompt: "p", RepoURL: "r",
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
}

func TestStore_Queue(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	defer cleanupTables(t, pool)

	store := NewStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Enqueue two entries with different priorities
	if err := store.Enqueue(ctx, &QueueEntry{
		ID: "q-1", AgentRunID: "ar-low", Priority: 0, UserID: "user1", CreatedAt: now,
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := store.Enqueue(ctx, &QueueEntry{
		ID: "q-2", AgentRunID: "ar-high", Priority: 10, UserID: "user2", CreatedAt: now,
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	length, err := store.QueueLength(ctx)
	if err != nil {
		t.Fatalf("QueueLength: %v", err)
	}
	if length != 2 {
		t.Errorf("got queue length %d, want 2", length)
	}

	// Dequeue should return highest priority first
	entry, err := store.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if entry.AgentRunID != "ar-high" {
		t.Errorf("got AgentRunID %q, want %q", entry.AgentRunID, "ar-high")
	}

	length, err = store.QueueLength(ctx)
	if err != nil {
		t.Fatalf("QueueLength: %v", err)
	}
	if length != 1 {
		t.Errorf("got queue length %d, want 1", length)
	}

	// Dequeue empty queue
	if _, err := store.Dequeue(ctx); err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	entry, err = store.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue on empty failed: %v", err)
	}
	if entry != nil {
		t.Error("expected nil on empty queue")
	}
}
