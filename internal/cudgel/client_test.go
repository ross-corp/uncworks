package cudgel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uncworks/aot/internal/cudgel"
)

func TestHTTPClient_SemanticSearch_Success(t *testing.T) {
	want := []cudgel.Symbol{
		{Name: "Foo", Kind: "function", File: "foo.go", Line: 10, Snippet: "func Foo()", Score: 0.9},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.Method != http.MethodPost {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := cudgel.NewHTTPClient(srv.URL)
	got, err := c.SemanticSearch(context.Background(), "auth middleware", 5)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Foo" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestHTTPClient_SemanticSearch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cudgel unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := cudgel.NewHTTPClient(srv.URL)
	_, err := c.SemanticSearch(context.Background(), "auth", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHTTPClient_SemanticSearch_Unreachable(t *testing.T) {
	c := cudgel.NewHTTPClient("http://127.0.0.1:19999")                   // nothing listening
	ctx, cancel := context.WithTimeout(context.Background(), 100_000_000) // 100ms
	defer cancel()
	_, err := c.SemanticSearch(ctx, "auth", 5)
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestHTTPClient_GraphTraversal_Success(t *testing.T) {
	want := []cudgel.Edge{
		{From: "A", To: "B", Kind: "calls"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graph" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := cudgel.NewHTTPClient(srv.URL)
	got, err := c.GraphTraversal(context.Background(), "A", 2)
	if err != nil {
		t.Fatalf("GraphTraversal: %v", err)
	}
	if len(got) != 1 || got[0].From != "A" {
		t.Errorf("unexpected edges: %+v", got)
	}
}

func TestHTTPClient_GraphTraversal_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := cudgel.NewHTTPClient(srv.URL)
	got, err := c.GraphTraversal(context.Background(), "unknown.Symbol", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty edges, got %+v", got)
	}
}

func TestNopClient_ReturnsEmpty(t *testing.T) {
	var c cudgel.Client = &cudgel.NopClient{}

	symbols, err := c.SemanticSearch(context.Background(), "anything", 10)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected empty symbols, got %+v", symbols)
	}

	edges, err := c.GraphTraversal(context.Background(), "anything", 2)
	if err != nil {
		t.Fatalf("GraphTraversal: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected empty edges, got %+v", edges)
	}
}
