package brain

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// SearchQuery represents a semantic search request against the knowledge base.
type SearchQuery struct {
	QueryVec      []float32
	RepoURL       string
	SourceFilter  string // "code", "trace", or "" (both)
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Limit         int
}

// SearchResult represents a unified search result from code or trace chunks.
type SearchResult struct {
	ChunkText    string
	SourceType   string // "code" or "trace"
	Similarity   float64
	BoostedScore float64
	AgentRunID   string
	FilePath     string
	Language     string
	NodeType     string
	ChunkType    string
	Severity     string
	RepoURL      string
	CreatedAt    time.Time
}

// Searcher performs semantic search across the knowledge base.
type Searcher struct {
	store *Store
}

// NewSearcher creates a Searcher backed by the given Store.
func NewSearcher(store *Store) *Searcher {
	return &Searcher{store: store}
}

// Search runs a semantic search query against code and/or trace chunks,
// merging and re-ranking results by boosted similarity score.
func (s *Searcher) Search(ctx context.Context, q SearchQuery) ([]SearchResult, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.SourceFilter != "" && q.SourceFilter != "code" && q.SourceFilter != "trace" {
		return nil, fmt.Errorf("invalid SourceFilter %q: must be empty, \"code\", or \"trace\"", q.SourceFilter)
	}

	var results []SearchResult

	// Search code chunks
	if q.SourceFilter == "" || q.SourceFilter == "code" {
		codeResults, err := s.store.SearchCodeChunks(ctx, q.QueryVec, q.RepoURL, q.Limit, q.CreatedAfter, q.CreatedBefore)
		if err != nil {
			return nil, err
		}
		for _, r := range codeResults {
			results = append(results, SearchResult{
				ChunkText:    r.ChunkText,
				SourceType:   "code",
				Similarity:   r.Similarity,
				BoostedScore: r.Similarity * float64(r.Boost),
				AgentRunID:   r.AgentRunID,
				FilePath:     r.FilePath,
				Language:     r.Language,
				NodeType:     r.NodeType,
				RepoURL:      r.RepoURL,
				CreatedAt:    r.CreatedAt,
			})
		}
	}

	// Search trace chunks
	if q.SourceFilter == "" || q.SourceFilter == "trace" {
		traceResults, err := s.store.SearchTraceChunks(ctx, q.QueryVec, q.RepoURL, q.Limit, q.CreatedAfter, q.CreatedBefore)
		if err != nil {
			return nil, err
		}
		for _, r := range traceResults {
			results = append(results, SearchResult{
				ChunkText:    r.ChunkText,
				SourceType:   "trace",
				Similarity:   r.Similarity,
				BoostedScore: r.Similarity, // trace chunks have no boost
				AgentRunID:   r.AgentRunID,
				ChunkType:    r.ChunkType,
				Severity:     r.Severity,
				RepoURL:      r.RepoURL,
				CreatedAt:    r.CreatedAt,
			})
		}
	}

	// Sort by boosted score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].BoostedScore > results[j].BoostedScore
	})

	// Clamp to limit
	if len(results) > q.Limit {
		results = results[:q.Limit]
	}

	return results, nil
}
