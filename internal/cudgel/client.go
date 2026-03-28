// Package cudgel provides a Go client for the cudgel HTTP shim service.
// The client communicates with the cudgel-shim Deployment via HTTP and exposes
// SemanticSearch and GraphTraversal methods.
package cudgel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Symbol is a code symbol returned by a semantic search.
type Symbol struct {
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// Edge represents a directed relationship in a call graph.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

// Client is the interface for interacting with the cudgel service.
type Client interface {
	// SemanticSearch performs a semantic code search and returns ranked symbols.
	SemanticSearch(ctx context.Context, query string, limit int) ([]Symbol, error)
	// GraphTraversal returns call-graph edges for the given symbol.
	GraphTraversal(ctx context.Context, symbol string, depth int) ([]Edge, error)
}

// HTTPClient is a Client backed by the cudgel HTTP shim.
type HTTPClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTPClient pointed at the given endpoint.
// endpoint should be the base URL of the cudgel-shim service, e.g.
// "http://cudgel:8080".
func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SemanticSearch sends a POST /search request and returns the ranked symbols.
func (c *HTTPClient) SemanticSearch(ctx context.Context, query string, limit int) ([]Symbol, error) {
	body, err := json.Marshal(map[string]any{
		"query": query,
		"limit": limit,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cudgel /search returned %d", resp.StatusCode)
	}

	var symbols []Symbol
	if err := json.NewDecoder(resp.Body).Decode(&symbols); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return symbols, nil
}

// GraphTraversal sends a POST /graph request and returns the call-graph edges.
func (c *HTTPClient) GraphTraversal(ctx context.Context, symbol string, depth int) ([]Edge, error) {
	body, err := json.Marshal(map[string]any{
		"symbol": symbol,
		"depth":  depth,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/graph", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cudgel /graph returned %d", resp.StatusCode)
	}

	var edges []Edge
	if err := json.NewDecoder(resp.Body).Decode(&edges); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if edges == nil {
		edges = []Edge{}
	}
	return edges, nil
}

// NopClient is a Client that always returns empty results without making any
// HTTP calls. Used when CUDGEL_ENDPOINT is unset.
type NopClient struct{}

// SemanticSearch returns an empty slice with no error.
func (n *NopClient) SemanticSearch(_ context.Context, _ string, _ int) ([]Symbol, error) {
	return []Symbol{}, nil
}

// GraphTraversal returns an empty slice with no error.
func (n *NopClient) GraphTraversal(_ context.Context, _ string, _ int) ([]Edge, error) {
	return []Edge{}, nil
}
