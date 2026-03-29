package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

// maxProjectBodyBytes caps request bodies for project mutations at 256 KB.
const maxProjectBodyBytes = 256 << 10

// ProjectHandler handles Project REST endpoints.
type ProjectHandler struct {
	K8sClient client.Client
	Namespace string
	SoftServe softserve.RepoManager
}

// RegisterProjectHandlers registers Project REST routes.
func (h *ProjectHandler) RegisterProjectHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects", h.handleListProjects)
	mux.HandleFunc("POST /api/v1/projects", h.handleCreateProject)
	mux.HandleFunc("GET /api/v1/projects/{name}", h.handleGetProject)
	mux.HandleFunc("PUT /api/v1/projects/{name}", h.handleUpdateProject)
	mux.HandleFunc("DELETE /api/v1/projects/{name}", h.handleDeleteProject)
	mux.HandleFunc("GET /api/v1/projects/{name}/files", h.handleListFiles)
	mux.HandleFunc("GET /api/v1/projects/{name}/files/{path...}", h.handleReadFile)
	mux.HandleFunc("PUT /api/v1/projects/{name}/files/{path...}", h.handleWriteFile)
}

type projectResponse struct {
	Name                 string                       `json:"name"`
	DisplayName          string                       `json:"displayName"`
	Description          string                       `json:"description"`
	Repos                []aotv1alpha1.Repository     `json:"repos"`
	Devbox               *aotv1alpha1.DevboxConfig    `json:"devbox,omitempty"`
	Defaults             *aotv1alpha1.ProjectDefaults `json:"defaults,omitempty"`
	ConfigRepoReady      bool                         `json:"configRepoReady"`
	ConfigRepoURL        string                       `json:"configRepoURL"`
	// ConfigRepoMessage is set when ConfigRepoReady is false and the controller
	// has recorded a reason — e.g. "Failed to reach soft-serve: connection refused".
	ConfigRepoMessage    string                       `json:"configRepoMessage,omitempty"`
	RunCount             int32                        `json:"runCount"`
	LastRunID            string                       `json:"lastRunId"`
	TotalCost            string                       `json:"totalCost"`
	CreatedAt            string                       `json:"createdAt"`
}

func projectToResponse(p *aotv1alpha1.Project) projectResponse {
	r := projectResponse{
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
	// Surface the condition message so the UI can show why provisioning is stuck.
	if !p.Status.ConfigRepoReady {
		for _, c := range p.Status.Conditions {
			if c.Type == "ConfigRepoReady" && c.Status == "False" && c.Message != "" {
				r.ConfigRepoMessage = c.Message
				break
			}
		}
	}
	return r
}

func (h *ProjectHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	var list aotv1alpha1.ProjectList
	if err := h.K8sClient.List(r.Context(), &list, client.InNamespace(h.Namespace)); err != nil {
		slog.Error("listing projects failed", slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list projects"})
		return
	}

	// Count runs per project dynamically (status.RunCount is not updated by controller).
	var runList aotv1alpha1.AgentRunList
	runCounts := map[string]int32{}
	if err := h.K8sClient.List(r.Context(), &runList, client.InNamespace(h.Namespace)); err == nil {
		for _, run := range runList.Items {
			if run.Spec.ProjectRef != "" {
				runCounts[run.Spec.ProjectRef]++
			}
		}
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})

	capped := capList(list.Items, maxListItems)
	var resp []projectResponse
	for i := range capped {
		r := projectToResponse(&capped[i])
		if n, ok := runCounts[capped[i].Name]; ok {
			r.RunCount = n
		}
		resp = append(resp, r)
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
	if err := json.NewDecoder(io.LimitReader(r.Body, maxProjectBodyBytes)).Decode(&body); err != nil {
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
		slog.Error("creating project failed", "name", project.Name, slog.Any("error", err))
		writeJSON(w, http.StatusConflict, errorResponse{Error: "project already exists or could not be created"})
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
		if apierrors.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "project not found"})
		} else {
			slog.Error("getting project failed", "name", name, slog.Any("error", err))
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get project"})
		}
		return
	}
	writeJSON(w, http.StatusOK, projectToResponse(project))
}

func (h *ProjectHandler) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	project := &aotv1alpha1.Project{}
	if err := h.K8sClient.Get(r.Context(), client.ObjectKey{
		Namespace: h.Namespace, Name: name,
	}, project); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "project not found"})
		} else {
			slog.Error("getting project for update failed", "name", name, slog.Any("error", err))
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get project"})
		}
		return
	}

	var body struct {
		DisplayName *string                      `json:"displayName,omitempty"`
		Description *string                      `json:"description,omitempty"`
		Repos       []aotv1alpha1.Repository     `json:"repos,omitempty"`
		Devbox      *aotv1alpha1.DevboxConfig    `json:"devbox,omitempty"`
		Defaults    *aotv1alpha1.ProjectDefaults `json:"defaults,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxProjectBodyBytes)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	if body.DisplayName != nil {
		project.Spec.DisplayName = *body.DisplayName
	}
	if body.Description != nil {
		project.Spec.Description = *body.Description
	}
	if body.Repos != nil {
		project.Spec.Repos = body.Repos
	}
	if body.Devbox != nil {
		project.Spec.Devbox = body.Devbox
	}
	if body.Defaults != nil {
		project.Spec.Defaults = body.Defaults
	}

	if err := h.K8sClient.Update(r.Context(), project); err != nil {
		slog.Error("updating project failed", "name", name, slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update project"})
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
		slog.Error("deleting project failed", "name", name, slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete project"})
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
	if !isValidRepoPath(path) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid file path"})
		return
	}
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
	if !isValidRepoPath(path) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid file path"})
		return
	}
	if h.SoftServe == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "soft-serve not configured"})
		return
	}

	var body struct {
		Content   string `json:"content"`
		CommitMsg string `json:"commitMessage"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxProjectBodyBytes)).Decode(&body); err != nil {
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

// isValidRepoPath validates a file path within a repo is safe (no traversal).
func isValidRepoPath(p string) bool {
	if p == "" {
		return false
	}
	// Reject absolute paths
	if strings.HasPrefix(p, "/") {
		return false
	}
	// Clean the path and reject traversal
	cleaned := filepath.Clean(p)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
		return false
	}
	// Reject hidden files at root level (except .devcontainer)
	parts := strings.Split(cleaned, "/")
	if len(parts) > 0 && strings.HasPrefix(parts[0], ".") && parts[0] != ".devcontainer" {
		return false
	}
	return true
}
