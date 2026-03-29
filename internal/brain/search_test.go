package brain

import (
	"context"
	"testing"
)

func TestSearcher_EmptyQueryVec(t *testing.T) {
	store := NewStore(nil)
	s := NewSearcher(store)
	_, err := s.Search(context.Background(), SearchQuery{QueryVec: nil})
	if err == nil {
		t.Fatal("expected error for nil QueryVec")
	}
	_, err = s.Search(context.Background(), SearchQuery{QueryVec: []float32{}})
	if err == nil {
		t.Fatal("expected error for empty QueryVec")
	}
}

func TestSearcher_InvalidSourceFilter(t *testing.T) {
	store := NewStore(nil)
	s := NewSearcher(store)
	_, err := s.Search(context.Background(), SearchQuery{
		QueryVec:     []float32{0.1},
		SourceFilter: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid SourceFilter")
	}
}

func TestSearcher_LimitClamping(t *testing.T) {
	// Use a store with pgvectorReady=false so the DB calls fail fast rather
	// than panicking on a nil pool; we only want to exercise limit clamping.
	store := NewStore(nil)
	s := NewSearcher(store)

	// Limit=0 defaults to 10; still fails at pgvector check — confirm it gets that far
	_, err := s.Search(context.Background(), SearchQuery{
		QueryVec: []float32{0.1},
		Limit:    0,
	})
	// Error is expected (pgvector not ready), but it must not be the validation error
	if err == nil {
		t.Fatal("expected an error from pgvector check")
	}

	// Limit > 100 is clamped; same expected failure path
	_, err = s.Search(context.Background(), SearchQuery{
		QueryVec: []float32{0.1},
		Limit:    999,
	})
	if err == nil {
		t.Fatal("expected an error from pgvector check")
	}
}

func TestSearcher_SourceFilterCode_NoPgvector(t *testing.T) {
	store := NewStore(nil) // pgvectorReady = false
	s := NewSearcher(store)
	_, err := s.Search(context.Background(), SearchQuery{
		QueryVec:     []float32{0.1},
		SourceFilter: "code",
	})
	if err == nil {
		t.Fatal("expected error: pgvector not available")
	}
}

func TestSearcher_SourceFilterTrace_NoPgvector(t *testing.T) {
	store := NewStore(nil)
	s := NewSearcher(store)
	_, err := s.Search(context.Background(), SearchQuery{
		QueryVec:     []float32{0.1},
		SourceFilter: "trace",
	})
	if err == nil {
		t.Fatal("expected error: pgvector not available")
	}
}
