package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

// ProjectHandler handles Project REST endpoints.
type ProjectHandler struct {
	K8sClient client.Client
	Namespace string
	SoftServe *softserve.Client
}

// RegisterProjectHandlers registers Project REST routes.
func (h *ProjectHandler) RegisterProjectHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects", h.handleListProjects)
	mux.HandleFunc("POST /api/v1/projects", h.handleCreateProject)
	mux.HandleFunc("GET /api/v1/projects/{name}", h.handleGetProject)
	mux.HandleFunc("DELETE /api/v1/projects/{name}", h.handleDeleteProject)
	mux.HandleFunc("GET /api/v1/projects/{name}/files", h.handleListFiles)
	mux.HandleFunc("GET /api/v1/projects/{name}/files/{path...}", h.handleReadFile)
	mux.HandleFunc("PUT /api/v1/projects/{name}/files/{path...}", h.handleWriteFile)
}

type projectResponse struct {
	Name            string                       `json:"name"`
	DisplayName     string                       `json:"displayName"`
	Description     string                       `json:"description"`
	Repos           []aotv1alpha1.Repository     `json:"repos"`
	Devbox          *aotv1alpha1.DevboxConfig    `json:"devbox,omitempty"`
	Defaults        *aotv1alpha1.ProjectDefaults `json:"defaults,omitempty"`
	ConfigRepoReady bool                         `json:"configRepoReady"`
	ConfigRepoURL   string                       `json:"configRepoURL"`
	RunCount        int32                        `json:"runCount"`
	LastRunID       string                       `json:"lastRunId"`
	TotalCost       string                       `json:"totalCost"`
	CreatedAt       string                       `json:"createdAt"`
}

func projectToResponse(p *aotv1alpha1.Project) projectResponse {
	return projectResponse{
		Name:            p.Name,
		DisplayName:     p.Spec.DisplayName,
		Description:     p.Spec.Description,
		Repos:           p.Spec.Repos,
		Devbox:          p.Spec.Devbox,
		Defaults:        p.Spec.Defaults,
		ConfigRepoReady: p.Status.ConfigRepoReady,
		ConfigRepoURL:   p.Status.ConfigRepoURL,
		RunCount:        p.Status.RunCount,
		LastRunID:       p.Status.LastRunID,
		TotalCost:       p.Status.TotalCost,
		CreatedAt:       p.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *ProjectHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.ProjectList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})

	var resp []projectResponse
	for i := range list.Items {
		resp = append(resp, projectToResponse(&list.Items[i]))
	}
	if resp == nil {
		resp = []projectResponse{}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string                       `json:"name"`
		DisplayName string                       `json:"displayName"`
		Description string                       `json:"description"`
		Repos       []aotv1alpha1.Repository     `json:"repos"`
		Devbox      *aotv1alpha1.DevboxConfig    `json:"devbox"`
		Defaults    *aotv1alpha1.ProjectDefaults `json:"defaults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}

	project := &aotv1alpha1.Project{}
	project.Name = body.Name
	project.Namespace = h.Namespace
	project.Spec = aotv1alpha1.ProjectSpec{
		DisplayName: body.DisplayName,
		Description: body.Description,
		Repos:       body.Repos,
		Devbox:      body.Devbox,
		Defaults:    body.Defaults,
	}

	if err := h.K8sClient.Create(r.Context(), project); err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{Error: fmt.Sprintf("create failed: %v", err)})
		return
	}

	writeJSON(w, http.StatusCreated, projectToResponse(project))
}

func (h *ProjectHandler) handleGetProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	project := &aotv1alpha1.Project{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{
		Namespace: h.Namespace, Name: name,
	}, project); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("project not found: %v", err)})
		return
	}
	writeJSON(w, http.StatusOK, projectToResponse(project))
}

func (h *ProjectHandler) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	project := &aotv1alpha1.Project{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{
		Namespace: h.Namespace, Name: name,
	}, project); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "project not found"})
		return
	}
	if err := h.K8sClient.Delete(r.Context(), project); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (h *ProjectHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if h.SoftServe == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "soft-serve not configured"})
		return
	}
	files, err := h.SoftServe.ListFiles(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("list files: %v", err)})
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *ProjectHandler) handleReadFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := r.PathValue("path")
	if h.SoftServe == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "soft-serve not configured"})
		return
	}
	content, err := h.SoftServe.ReadFile(name, path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("read file: %v", err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path, "content": content})
}

func (h *ProjectHandler) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := r.PathValue("path")
	if h.SoftServe == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "soft-serve not configured"})
		return
	}

	var body struct {
		Content   string `json:"content"`
		CommitMsg string `json:"commitMessage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if body.CommitMsg == "" {
		body.CommitMsg = fmt.Sprintf("update %s", path)
	}

	if err := h.SoftServe.WriteFile(name, path, body.Content, body.CommitMsg); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("write file: %v", err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path, "status": "committed"})
}
