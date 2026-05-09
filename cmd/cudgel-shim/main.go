// Package main implements the cudgel HTTP shim — a thin Go HTTP server that
// bridges the cudgel CLI binary to HTTP endpoints consumed by the aot cluster.
//
// Endpoints:
//
//	POST /search  — semantic code search via `cudgel query`
//	POST /graph   — call-graph traversal via `cudgel graph`
//	POST /index   — trigger background index via `cudgel index`
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Symbol is a single code symbol returned by a search.
type Symbol struct {
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// Edge is a single relationship in a call graph.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

// searchRequest is the JSON body for POST /search.
type searchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// graphRequest is the JSON body for POST /graph.
type graphRequest struct {
	Symbol string `json:"symbol"`
	Depth  int    `json:"depth"`
}

// indexRequest is the JSON body for POST /index.
type indexRequest struct {
	RepoPath string `json:"repo_path"`
}

// allowedRepoPrefixes is the list of path prefixes that are permitted for indexing.
// Requests with repo_path outside these prefixes are rejected with 400.
var allowedRepoPrefixes = []string{"/workspace", "/repos"}

// server holds shared server state.
type server struct {
	cudgelBin string // path to the cudgel binary (default: "cudgel" on PATH)
}

func newServer() *server {
	bin := os.Getenv("CUDGEL_BIN")
	if bin == "" {
		bin = "cudgel"
	}
	return &server{cudgelBin: bin}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// checkBinary returns an error if the cudgel binary cannot be found.
func (s *server) checkBinary() error {
	_, err := exec.LookPath(s.cudgelBin)
	if err != nil {
		return fmt.Errorf("cudgel binary not found: %w", err)
	}
	return nil
}

// handleSearch handles POST /search.
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "query must not be empty")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	if err := s.checkBinary(); err != nil {
		slog.Error("cudgel binary unavailable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "cudgel binary not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := runCudgel(ctx, s.cudgelBin, "query", req.Query,
		"--limit", fmt.Sprintf("%d", limit), "--json")
	if err != nil {
		slog.Error("cudgel query failed", "err", err, "query", req.Query)
		writeError(w, http.StatusServiceUnavailable, "cudgel query failed: "+err.Error())
		return
	}

	var symbols []Symbol
	if err := json.Unmarshal([]byte(out), &symbols); err != nil {
		slog.Error("failed to parse cudgel output", "err", err, "out", out)
		writeError(w, http.StatusServiceUnavailable, "failed to parse cudgel output")
		return
	}

	writeJSON(w, http.StatusOK, symbols)
}

// handleGraph handles POST /graph.
func (s *server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req graphRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Symbol) == "" {
		writeError(w, http.StatusBadRequest, "symbol must not be empty")
		return
	}

	depth := req.Depth
	if depth <= 0 {
		depth = 2
	}
	if depth > 5 {
		depth = 5
	}

	if err := s.checkBinary(); err != nil {
		slog.Error("cudgel binary unavailable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "cudgel binary not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := runCudgel(ctx, s.cudgelBin, "graph", req.Symbol,
		"--depth", fmt.Sprintf("%d", depth), "--json")
	if err != nil {
		slog.Error("cudgel graph failed", "err", err, "symbol", req.Symbol)
		writeError(w, http.StatusServiceUnavailable, "cudgel graph failed: "+err.Error())
		return
	}

	var edges []Edge
	if err := json.Unmarshal([]byte(out), &edges); err != nil {
		// Unknown symbol may produce empty/null — treat as empty graph
		slog.Warn("failed to parse cudgel graph output, returning empty", "err", err)
		writeJSON(w, http.StatusOK, []Edge{})
		return
	}
	if edges == nil {
		edges = []Edge{}
	}

	writeJSON(w, http.StatusOK, edges)
}

// handleIndex handles POST /index.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req indexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.RepoPath) == "" {
		writeError(w, http.StatusBadRequest, "repo_path must not be empty")
		return
	}

	if !isAllowedPath(req.RepoPath) {
		writeError(w, http.StatusBadRequest, "repo_path is not in an allowed directory")
		return
	}

	if err := s.checkBinary(); err != nil {
		slog.Error("cudgel binary unavailable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "cudgel binary not available")
		return
	}

	// Run indexing asynchronously; respond 202 immediately.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		out, err := runCudgel(ctx, s.cudgelBin, "index", req.RepoPath)
		if err != nil {
			slog.Error("cudgel index failed", "repo_path", req.RepoPath, "err", err, "output", out)
			return
		}
		slog.Info("cudgel index complete", "repo_path", req.RepoPath, "output", out)
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

// isAllowedPath returns true if repoPath has one of the allowed prefixes.
func isAllowedPath(repoPath string) bool {
	clean := filepath.Clean(repoPath)
	for _, prefix := range allowedRepoPrefixes {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}

// runCudgel executes the cudgel binary with the given arguments and returns
// combined stdout output. Returns an error if the process fails or the binary
// is not found.
func runCudgel(ctx context.Context, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return strings.TrimSpace(string(out)), fmt.Errorf("exit %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	addr := os.Getenv("CUDGEL_SHIM_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	s := newServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/graph", s.handleGraph)
	mux.HandleFunc("/index", s.handleIndex)

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		slog.Info("cudgel-shim starting", "addr", addr, "binary", s.cudgelBin)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server exited", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	
	<-ctx.Done()
	slog.Info("shutting down cudgel-shim server...")
	
	// Give in-flight requests up to 30 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("cudgel-shim shutdown error", "err", err)
	} else {
		slog.Info("cudgel-shim shutdown complete")
	}
}
