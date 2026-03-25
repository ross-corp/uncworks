package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// ChainHandler handles RunTemplate, Chain, ChainRun, and Schedule REST endpoints.
type ChainHandler struct {
	K8sClient client.Client
	Namespace string
}

// RegisterChainHandlers registers all chain-related REST routes.
func (h *ChainHandler) RegisterChainHandlers(mux *http.ServeMux) {
	// RunTemplates
	mux.HandleFunc("GET /api/v1/templates", h.handleListTemplates)
	mux.HandleFunc("POST /api/v1/templates", h.handleCreateTemplate)
	mux.HandleFunc("GET /api/v1/templates/{name}", h.handleGetTemplate)
	mux.HandleFunc("DELETE /api/v1/templates/{name}", h.handleDeleteTemplate)
	mux.HandleFunc("POST /api/v1/templates/{name}/trigger", h.handleTriggerTemplate)

	// Chains
	mux.HandleFunc("GET /api/v1/chains", h.handleListChains)
	mux.HandleFunc("POST /api/v1/chains", h.handleCreateChain)
	mux.HandleFunc("GET /api/v1/chains/{name}", h.handleGetChain)
	mux.HandleFunc("DELETE /api/v1/chains/{name}", h.handleDeleteChain)
	mux.HandleFunc("POST /api/v1/chains/{name}/trigger", h.handleTriggerChain)

	// ChainRuns
	mux.HandleFunc("GET /api/v1/chainruns", h.handleListChainRuns)
	mux.HandleFunc("GET /api/v1/chainruns/{name}", h.handleGetChainRun)

	// Schedules
	mux.HandleFunc("GET /api/v1/schedules", h.handleListSchedules)
	mux.HandleFunc("POST /api/v1/schedules", h.handleCreateSchedule)
	mux.HandleFunc("GET /api/v1/schedules/{name}", h.handleGetSchedule)
	mux.HandleFunc("DELETE /api/v1/schedules/{name}", h.handleDeleteSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{name}/suspend", h.handleSuspendSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{name}/resume", h.handleResumeSchedule)
}

// ── RunTemplate handlers ──

func (h *ChainHandler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.RunTemplateList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})
	writeJSON(w, http.StatusOK, list.Items)
}

func (h *ChainHandler) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		aotv1alpha1.RunTemplateSpec
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name required"})
		return
	}
	tmpl := &aotv1alpha1.RunTemplate{}
	tmpl.Name = body.Name
	tmpl.Namespace = h.Namespace
	tmpl.Spec = body.RunTemplateSpec
	if body.DisplayName != "" {
		tmpl.Spec.DisplayName = body.DisplayName
	}
	if err := h.K8sClient.Create(r.Context(), tmpl); err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

func (h *ChainHandler) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tmpl := &aotv1alpha1.RunTemplate{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, tmpl); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "template not found"})
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

func (h *ChainHandler) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tmpl := &aotv1alpha1.RunTemplate{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, tmpl); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "template not found"})
		return
	}
	// 409 check: are any Chains referencing this template?
	var chainList aotv1alpha1.ChainList
	if err := h.K8sClient.List(r.Context(), &chainList, client.InNamespace(h.Namespace)); err == nil {
		for _, c := range chainList.Items {
			for _, step := range c.Spec.Steps {
				if step.TemplateRef == name {
					writeJSON(w, http.StatusConflict, errorResponse{
						Error: fmt.Sprintf("template %q is referenced by chain %q (step %q)", name, c.Name, step.Name),
					})
					return
				}
			}
		}
	}
	if err := h.K8sClient.Delete(r.Context(), tmpl); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (h *ChainHandler) handleTriggerTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tmpl := &aotv1alpha1.RunTemplate{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, tmpl); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "template not found"})
		return
	}
	// Create an AgentRun from the template
	run := &aotv1alpha1.AgentRun{}
	run.GenerateName = "ar-"
	run.Namespace = h.Namespace
	run.Spec = aotv1alpha1.AgentRunSpec{
		Backend:            aotv1alpha1.BackendPod,
		Repos:              tmpl.Spec.Repos,
		Prompt:             tmpl.Spec.Prompt,
		ModelTier:          tmpl.Spec.ModelTier,
		ManageModelTier:    tmpl.Spec.ManageModelTier,
		ImplementModelTier: tmpl.Spec.ImplementModelTier,
		OrchestrationMode:  tmpl.Spec.OrchestrationMode,
		TTLSeconds:         tmpl.Spec.TTLSeconds,
		AutoPush:           tmpl.Spec.AutoPush,
		AutoPR:             tmpl.Spec.AutoPR,
		PRBaseBranch:       tmpl.Spec.PRBaseBranch,
		ProjectRef:         tmpl.Spec.ProjectRef,
		SpecRef:            tmpl.Spec.SpecRef,
	}
	run.Labels = map[string]string{
		"aot.uncworks.io/template": tmpl.Name,
	}
	if err := h.K8sClient.Create(r.Context(), run); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("create run: %v", err)})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"runId": run.Name, "template": tmpl.Name})
}

// ── Chain handlers ──

func (h *ChainHandler) handleListChains(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.ChainList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list.Items)
}

func (h *ChainHandler) handleCreateChain(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		aotv1alpha1.ChainSpec
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if body.Name == "" || len(body.Steps) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and steps required"})
		return
	}
	if err := aotv1alpha1.ValidateChainDAG(body.Steps); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("invalid chain DAG: %v", err)})
		return
	}
	chain := &aotv1alpha1.Chain{}
	chain.Name = body.Name
	chain.Namespace = h.Namespace
	chain.Spec = body.ChainSpec
	if err := h.K8sClient.Create(r.Context(), chain); err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, chain)
}

func (h *ChainHandler) handleGetChain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	chain := &aotv1alpha1.Chain{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, chain); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "chain not found"})
		return
	}
	writeJSON(w, http.StatusOK, chain)
}

func (h *ChainHandler) handleDeleteChain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	chain := &aotv1alpha1.Chain{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, chain); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "chain not found"})
		return
	}
	// 409 check: are any Schedules referencing this chain?
	var schedList aotv1alpha1.ScheduleList
	if err := h.K8sClient.List(r.Context(), &schedList, client.InNamespace(h.Namespace)); err == nil {
		for _, s := range schedList.Items {
			if s.Spec.ChainRef == name {
				writeJSON(w, http.StatusConflict, errorResponse{
					Error: fmt.Sprintf("chain %q is referenced by schedule %q", name, s.Name),
				})
				return
			}
		}
	}
	if err := h.K8sClient.Delete(r.Context(), chain); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (h *ChainHandler) handleTriggerChain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	chain := &aotv1alpha1.Chain{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, chain); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "chain not found"})
		return
	}
	// Create a ChainRun
	cr := &aotv1alpha1.ChainRun{}
	cr.GenerateName = "cr-"
	cr.Namespace = h.Namespace
	cr.Spec = aotv1alpha1.ChainRunSpec{
		ChainRef:    name,
		TriggeredBy: "manual",
	}
	if err := h.K8sClient.Create(r.Context(), cr); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("create chain run: %v", err)})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"chainRunId": cr.Name, "chain": name})
}

// ── ChainRun handlers ──

func (h *ChainHandler) handleListChainRuns(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.ChainRunList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})
	writeJSON(w, http.StatusOK, list.Items)
}

func (h *ChainHandler) handleGetChainRun(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cr := &aotv1alpha1.ChainRun{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, cr); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "chain run not found"})
		return
	}
	writeJSON(w, http.StatusOK, cr)
}

// ── Schedule handlers ──

func (h *ChainHandler) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.ScheduleList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list.Items)
}

func (h *ChainHandler) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		aotv1alpha1.ScheduleSpec
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if body.Name == "" || body.Cron == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and cron required"})
		return
	}
	if body.ChainRef == "" && body.TemplateRef == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chainRef or templateRef required"})
		return
	}
	if body.ChainRef != "" && body.TemplateRef != "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chainRef and templateRef are mutually exclusive"})
		return
	}
	sched := &aotv1alpha1.Schedule{}
	sched.Name = body.Name
	sched.Namespace = h.Namespace
	sched.Spec = body.ScheduleSpec
	if err := h.K8sClient.Create(r.Context(), sched); err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sched)
}

func (h *ChainHandler) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sched := &aotv1alpha1.Schedule{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, sched); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "schedule not found"})
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

func (h *ChainHandler) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sched := &aotv1alpha1.Schedule{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, sched); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "schedule not found"})
		return
	}
	if err := h.K8sClient.Delete(r.Context(), sched); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (h *ChainHandler) handleSuspendSchedule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sched := &aotv1alpha1.Schedule{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, sched); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "schedule not found"})
		return
	}
	sched.Spec.Suspend = true
	if err := h.K8sClient.Update(r.Context(), sched); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"name": name, "suspended": true})
}

func (h *ChainHandler) handleResumeSchedule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sched := &aotv1alpha1.Schedule{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{Namespace: h.Namespace, Name: name}, sched); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "schedule not found"})
		return
	}
	sched.Spec.Suspend = false
	if err := h.K8sClient.Update(r.Context(), sched); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"name": name, "suspended": false})
}
