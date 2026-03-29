package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// maxArchiveBodyBytes caps request bodies for archive mutations at 64 KB.
const maxArchiveBodyBytes = 64 << 10

// ArchiveHandler handles archive/unarchive REST endpoints.
type ArchiveHandler struct {
	K8sClient client.Client
	Namespace string
}

// RegisterArchiveHandlers registers the archive REST routes.
func (h *ArchiveHandler) RegisterArchiveHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/runs/{id}/archive", h.handleArchive)
	mux.HandleFunc("POST /api/v1/runs/bulk-archive", h.handleBulkArchive)
}

func (h *ArchiveHandler) handleArchive(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing run ID"})
		return
	}

	var body struct {
		Archived bool `json:"archived"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxArchiveBodyBytes)).Decode(&body); err != nil {
		body.Archived = true // default to archive
	}

	crd := &aotv1alpha1.AgentRun{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{
		Namespace: h.Namespace,
		Name:      runID,
	}, crd); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("run not found: %s", runID)})
		} else {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("get run: %v", err)})
		}
		return
	}

	crd.Status.Archived = body.Archived
	if err := h.K8sClient.Status().Update(r.Context(), crd); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("update failed: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"archived": body.Archived})
}

func (h *ArchiveHandler) handleBulkArchive(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RunIDs   []string `json:"runIds"`
		Archived *bool    `json:"archived"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxArchiveBodyBytes)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	archived := true // default to archive when omitted
	if body.Archived != nil {
		archived = *body.Archived
	}

	var errors []string
	for _, runID := range body.RunIDs {
		crd := &aotv1alpha1.AgentRun{}
		if err := h.K8sClient.Get(r.Context(), client.ObjectKey{
			Namespace: h.Namespace,
			Name:      runID,
		}, crd); err != nil {
			if apierrors.IsNotFound(err) {
				errors = append(errors, fmt.Sprintf("%s: not found", runID))
			} else {
				errors = append(errors, fmt.Sprintf("%s: get failed", runID))
			}
			continue
		}
		crd.Status.Archived = archived
		if err := h.K8sClient.Status().Update(r.Context(), crd); err != nil {
			errors = append(errors, fmt.Sprintf("%s: update failed", runID))
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"archived": len(body.RunIDs) - len(errors),
		"errors":   errors,
	})
}
