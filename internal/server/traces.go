package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TraceSpan represents a single trace span from an agent run.
type TraceSpan struct {
	ID        string                 `json:"id"`
	ParentID  string                 `json:"parentId,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // llm, tool, thought, input
	StartTime string                 `json:"startTime"`
	EndTime   string                 `json:"endTime"`
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
	mux.HandleFunc("GET /api/v1/runs/{id}/traces/{span-id}/diff", t.handleSpanDiff)
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
		log.Printf("failed to read spans file %s: %v", spansPath, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read traces: " + err.Error()})
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
	spanID := r.PathValue("span-id")

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
		log.Printf("failed to read spans file %s: %v", spansPath, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read traces: " + err.Error()})
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
func readSpansFile(path string) ([]TraceSpan, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

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
			log.Printf("skipping malformed span line: %v", err)
			continue
		}
		spans = append(spans, span)
	}

	if err := scanner.Err(); err != nil {
		return spans, fmt.Errorf("scan spans file: %w", err)
	}

	return spans, nil
}
