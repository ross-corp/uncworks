// Package embeddings tests cover ChunkText, ChunkCodeSimple, BoostForNodeType,
// and the Embedder's input validation and HTTP error handling.
package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- NewEmbedder defaults ---

func TestNewEmbedder_Defaults(t *testing.T) {
	e := NewEmbedder("", "", nil)
	if e.baseURL != DefaultOllamaURL {
		t.Errorf("baseURL: got %q, want %q", e.baseURL, DefaultOllamaURL)
	}
	if e.model != DefaultModel {
		t.Errorf("model: got %q, want %q", e.model, DefaultModel)
	}
	if e.httpClient == nil {
		t.Fatal("httpClient must not be nil")
	}
	if e.httpClient.Timeout != DefaultHTTPTimeout {
		t.Errorf("httpClient.Timeout: got %v, want %v", e.httpClient.Timeout, DefaultHTTPTimeout)
	}
}

func TestNewEmbedder_TrailingSlash(t *testing.T) {
	e := NewEmbedder("http://localhost:11434/", "", nil)
	if strings.HasSuffix(e.baseURL, "/") {
		t.Errorf("baseURL should not have trailing slash, got %q", e.baseURL)
	}
}

func TestNewEmbedder_CustomClientPreserved(t *testing.T) {
	custom := &http.Client{Timeout: 5}
	e := NewEmbedder("", "", custom)
	if e.httpClient != custom {
		t.Error("custom httpClient should be preserved as-is")
	}
}

// --- Embed input validation ---

func TestEmbed_EmptyText(t *testing.T) {
	e := NewEmbedder("", "", nil)
	_, err := e.Embed(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestEmbed_WhitespaceText(t *testing.T) {
	e := NewEmbedder("", "", nil)
	_, err := e.Embed(context.Background(), "   \t\n  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only text")
	}
}

// --- Embed HTTP behaviour ---

func TestEmbed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "test-model", srv.Client())
	_, err := e.Embed(context.Background(), "hello world")
	if err == nil {
		t.Fatal("expected error for non-200 HTTP response")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected 503 in error, got %v", err)
	}
}

func TestEmbed_EmptyEmbeddingsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"embeddings":[]}`)
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "test-model", srv.Client())
	_, err := e.Embed(context.Background(), "hello world")
	if err == nil {
		t.Fatal("expected error when embeddings array is empty")
	}
}

func TestEmbed_WrongDimension(t *testing.T) {
	// Return a vector of wrong dimension (e.g. 3 instead of 384)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := embedResponse{Embeddings: [][]float64{{0.1, 0.2, 0.3}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "test-model", srv.Client())
	_, err := e.Embed(context.Background(), "hello world")
	if err == nil {
		t.Fatal("expected error for wrong embedding dimension")
	}
	if !strings.Contains(err.Error(), "unexpected embedding dimension") {
		t.Errorf("expected dimension error message, got %v", err)
	}
}

func TestEmbed_Success(t *testing.T) {
	// Return a correctly-sized 384-dim vector
	raw := make([]float64, EmbeddingDim)
	for i := range raw {
		raw[i] = float64(i) * 0.001
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := embedResponse{Embeddings: [][]float64{raw}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	e := NewEmbedder(srv.URL, "test-model", srv.Client())
	vec, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != EmbeddingDim {
		t.Errorf("got %d dims, want %d", len(vec), EmbeddingDim)
	}
}

func TestEmbed_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is cancelled
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	e := NewEmbedder(srv.URL, "test-model", srv.Client())
	_, err := e.Embed(ctx, "hello world")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- ChunkText ---

func TestChunkText_Empty(t *testing.T) {
	chunks := ChunkText("", 512, 64)
	if len(chunks) != 0 {
		t.Errorf("got %d chunks for empty input, want 0", len(chunks))
	}
}

func TestChunkText_SingleParagraph(t *testing.T) {
	chunks := ChunkText("hello world", 512, 64)
	if len(chunks) != 1 {
		t.Errorf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0] != "hello world" {
		t.Errorf("got %q, want %q", chunks[0], "hello world")
	}
}

func TestChunkText_MultiParagraph_Splits(t *testing.T) {
	// Build a content that exceeds 10 estimated tokens to force a split
	words := strings.Repeat("word ", 20) // ~20 words >> 10 tokens
	para1 := words
	para2 := strings.Repeat("other ", 20)
	content := strings.TrimSpace(para1) + "\n\n" + strings.TrimSpace(para2)

	chunks := ChunkText(content, 10, 2)
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks for large content, got %d", len(chunks))
	}
}

func TestChunkText_DefaultsApplied(t *testing.T) {
	// maxTokens=0 and overlapTokens=0 should use defaults and not panic
	chunks := ChunkText("paragraph one\n\nparagraph two", 0, 0)
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestChunkText_WhitespaceOnlyParagraphs(t *testing.T) {
	chunks := ChunkText("   \n\n   \n\n   ", 512, 64)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for whitespace-only content, got %d", len(chunks))
	}
}

// --- ChunkCodeSimple ---

func TestChunkCodeSimple_Empty(t *testing.T) {
	chunks := ChunkCodeSimple("", "go")
	if len(chunks) != 0 {
		t.Errorf("got %d chunks for empty input, want 0", len(chunks))
	}
}

func TestChunkCodeSimple_GoFunction(t *testing.T) {
	code := `func Hello() string {
	return "hello"
}`
	chunks := ChunkCodeSimple(code, "go")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "function" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "function")
	}
}

func TestChunkCodeSimple_GoStruct(t *testing.T) {
	code := `type Foo struct {
	X int
}`
	chunks := ChunkCodeSimple(code, "go")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "struct" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "struct")
	}
}

func TestChunkCodeSimple_PythonDef(t *testing.T) {
	code := `def my_func(x):
    return x * 2`
	chunks := ChunkCodeSimple(code, "python")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "function" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "function")
	}
}

func TestChunkCodeSimple_PythonClass(t *testing.T) {
	code := `class MyClass:
    pass`
	chunks := ChunkCodeSimple(code, "python")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "class" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "class")
	}
}

func TestChunkCodeSimple_Import(t *testing.T) {
	code := `import "fmt"`
	chunks := ChunkCodeSimple(code, "go")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "import" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "import")
	}
}

func TestChunkCodeSimple_JSFunction(t *testing.T) {
	code := `function greet(name) { return "hi " + name; }`
	chunks := ChunkCodeSimple(code, "javascript")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].NodeType != "function" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "function")
	}
}

func TestChunkCodeSimple_DefaultLanguage(t *testing.T) {
	code := `def something(): pass`
	chunks := ChunkCodeSimple(code, "ruby")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	// Falls into default case: detects "def " via lowercase match
	if chunks[0].NodeType != "function" {
		t.Errorf("got NodeType %q, want %q", chunks[0].NodeType, "function")
	}
}

// --- BoostForNodeType ---

func TestBoostForNodeType(t *testing.T) {
	tests := []struct {
		nodeType string
		want     float32
	}{
		{"function", 1.0},
		{"method", 1.0},
		{"class", 0.9},
		{"struct", 0.9},
		{"import", 0.3},
		{"whitespace", 0.1},
		{"block", 0.7},
		{"unknown", 0.7},
	}
	for _, tc := range tests {
		t.Run(tc.nodeType, func(t *testing.T) {
			got := BoostForNodeType(tc.nodeType)
			if got != tc.want {
				t.Errorf("BoostForNodeType(%q) = %v, want %v", tc.nodeType, got, tc.want)
			}
		})
	}
}
