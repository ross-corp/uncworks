package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TraceSpan represents a single trace span from an agent run.
type TraceSpan struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"traceId,omitempty"`
	ParentID  string                 `json:"parentId,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`             // llm, tool, thought, input, stage
	StartTime string                 `json:"startTime"`        // RFC3339 timestamp (written by sidecar as time.Time)
	EndTime   string                 `json:"endTime"`          // RFC3339 timestamp (written by sidecar as time.Time)
	Status    string                 `json:"status,omitempty"` // "ok", "error", "unset"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	HasDiff   bool                   `json:"hasDiff"`
	Diff      *SpanDiff              `json:"diff,omitempty"`
}

// SpanDiff contains the file diffs associated with a trace span.
type SpanDiff struct {
	Files []FileDiff `json:"files"`
}

// FileDiff represents a single file's diff within a span.
type FileDiff struct {
	Path  string `json:"path"`
	Patch string `json:"patch"`
}

// TraceHandler serves trace-related API endpoints.
type TraceHandler struct {
	k8sClient  runtimeclient.Client
	restConfig *rest.Config
	namespace  string
}

// NewTraceHandler creates a new TraceHandler.
func NewTraceHandler(k8sClient runtimeclient.Client, restConfig *rest.Config, namespace string) *TraceHandler {
	return &TraceHandler{
		k8sClient:  k8sClient,
		restConfig: restConfig,
		namespace:  namespace,
	}
}

// RegisterTraceHandlers registers the trace REST endpoints on the given mux.
func (t *TraceHandler) RegisterTraceHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/runs/{id}/traces", t.handleListTraces)
	mux.HandleFunc("GET /api/v1/runs/{id}/traces/{spanId}/diff", t.handleSpanDiff)
}

// handleListTraces returns all trace spans for an agent run as a JSON array.
func (t *TraceHandler) handleListTraces(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	hostPath, err := t.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v (run may be archived or deleted)", runID, err)})
		return
	}

	spansPath := filepath.Join(hostPath, ".aot", "traces", "spans.jsonl")
	spans, err := readSpansFile(spansPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No traces file yet — return empty array.
			writeJSON(w, http.StatusOK, []TraceSpan{})
			return
		}
		slog.Error("failed to read spans file", "file", spansPath, slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read traces"})
		return
	}

	// Strip diff data from the list response (diffs are fetched individually).
	for i := range spans {
		spans[i].Diff = nil
	}

	writeJSON(w, http.StatusOK, spans)
}

// handleSpanDiff returns the diff data for a specific trace span.
func (t *TraceHandler) handleSpanDiff(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	spanID := r.PathValue("spanId")

	hostPath, err := t.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v (run may be archived or deleted)", runID, err)})
		return
	}

	spansPath := filepath.Join(hostPath, ".aot", "traces", "spans.jsonl")
	spans, err := readSpansFile(spansPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "no traces found for this run"})
			return
		}
		slog.Error("failed to read spans file", "file", spansPath, slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read traces"})
		return
	}

	for _, span := range spans {
		if span.ID == spanID {
			if span.Diff == nil {
				writeJSON(w, http.StatusOK, SpanDiff{Files: []FileDiff{}})
				return
			}
			writeJSON(w, http.StatusOK, span.Diff)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("span %q not found", spanID)})
}

// getPVCHostPath delegates to a FileHandler-style PVC host path lookup.
// This is a standalone version for the TraceHandler to avoid circular dependencies.
func (t *TraceHandler) getPVCHostPath(ctx context.Context, runID string) (string, error) {
	fh := &FileHandler{
		k8sClient:  t.k8sClient,
		restConfig: t.restConfig,
		namespace:  t.namespace,
	}
	return fh.getPVCHostPath(ctx, runID)
}

// readSpansFile reads a JSONL file of trace spans and returns them as a slice.
// When the same span ID appears multiple times (e.g., open then close), the
// later entry wins — this deduplicates stage spans that are written twice
// (once at start without endTime, once at close with endTime).
func readSpansFile(path string) ([]TraceSpan, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Use a map to deduplicate by ID — later entries override earlier ones
	byID := make(map[string]int) // span ID → index in spans slice
	var spans []TraceSpan
	scanner := bufio.NewScanner(file)
	// Allow up to 1MB per line for spans with large diffs.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var span TraceSpan
		if err := json.Unmarshal(line, &span); err != nil {
			slog.Debug("skipping malformed span line", slog.Any("error", err))
			continue
		}
		if idx, exists := byID[span.ID]; exists {
			// Replace earlier version with later (closed) version
			spans[idx] = span
		} else {
			byID[span.ID] = len(spans)
			spans = append(spans, span)
		}
	}

	if err := scanner.Err(); err != nil {
		return spans, fmt.Errorf("scan spans file: %w", err)
	}

	return spans, nil
}
