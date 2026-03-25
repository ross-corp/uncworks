package server

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// CountsHandler handles the lightweight counts endpoint used by GlobalNav.
type CountsHandler struct {
	K8sClient client.Client
	Namespace string
}

// RegisterCountsHandlers registers the counts REST endpoint on the given mux.
func (h *CountsHandler) RegisterCountsHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/counts", h.handleCounts)
}

// CountsResponse is the JSON shape returned by GET /api/v1/counts.
type CountsResponse struct {
	Runs       int `json:"runs"`
	ActiveRuns int `json:"activeRuns"`
	Projects   int `json:"projects"`
	Templates  int `json:"templates"`
	Chains     int `json:"chains"`
	ChainRuns  int `json:"chainruns"`
	Schedules  int `json:"schedules"`
}

func (h *CountsHandler) handleCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var runs aotv1alpha1.AgentRunList
	if err := h.K8sClient.List(ctx, &runs, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	activeRuns := 0
	for _, run := range runs.Items {
		p := run.Status.Phase
		if p == aotv1alpha1.AgentRunPhaseRunning ||
			p == aotv1alpha1.AgentRunPhasePending ||
			p == aotv1alpha1.AgentRunPhaseWaitingForInput {
			activeRuns++
		}
	}

	var projects aotv1alpha1.ProjectList
	if err := h.K8sClient.List(ctx, &projects, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var templates aotv1alpha1.RunTemplateList
	if err := h.K8sClient.List(ctx, &templates, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var chains aotv1alpha1.ChainList
	if err := h.K8sClient.List(ctx, &chains, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var chainRuns aotv1alpha1.ChainRunList
	if err := h.K8sClient.List(ctx, &chainRuns, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var schedules aotv1alpha1.ScheduleList
	if err := h.K8sClient.List(ctx, &schedules, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, CountsResponse{
		Runs:       len(runs.Items),
		ActiveRuns: activeRuns,
		Projects:   len(projects.Items),
		Templates:  len(templates.Items),
		Chains:     len(chains.Items),
		ChainRuns:  len(chainRuns.Items),
		Schedules:  len(schedules.Items),
	})
}
