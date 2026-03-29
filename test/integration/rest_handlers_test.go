package integration

// REST handler integration tests exercise the full HTTP handler stack
// (server.CountsHandler, server.ArchiveHandler, server.ProjectHandler,
// server.ChainHandler) using a fake k8s client from outside the internal/server
// package. These tests complement the unit tests in internal/server/*_test.go
// by verifying cross-handler consistency (e.g. counts reflecting created runs).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/test/testutil"
)

// buildMux registers all REST handlers onto a single mux backed by a shared
// fake k8s client so handlers see the same state.
func buildMux(t *testing.T) *http.ServeMux {
	t.Helper()
	k8s := testutil.NewFakeK8sClient(t, &aotv1alpha1.AgentRun{})

	mux := http.NewServeMux()

	countsH := &server.CountsHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	countsH.RegisterCountsHandlers(mux)

	archiveH := &server.ArchiveHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	archiveH.RegisterArchiveHandlers(mux)

	projectH := &server.ProjectHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	projectH.RegisterProjectHandlers(mux)

	chainH := &server.ChainHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	chainH.RegisterChainHandlers(mux)

	return mux
}

// ── GET /api/v1/counts ──

func TestREST_Counts_Empty(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("counts: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var resp server.CountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal counts: %v", err)
	}
	if resp.Runs != 0 {
		t.Errorf("expected 0 runs, got %d", resp.Runs)
	}
	if resp.ActiveRuns != 0 {
		t.Errorf("expected 0 active runs, got %d", resp.ActiveRuns)
	}
	if resp.Projects != 0 {
		t.Errorf("expected 0 projects, got %d", resp.Projects)
	}
	if resp.Templates != 0 {
		t.Errorf("expected 0 templates, got %d", resp.Templates)
	}
}

func TestREST_Counts_ReflectsCreatedProject(t *testing.T) {
	mux := buildMux(t)

	// Create a project
	body := `{"name":"count-test-proj","displayName":"Count Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: status %d, body: %s", rec.Code, rec.Body.String())
	}

	// Create a template
	tmplBody := `{"name":"count-tmpl","displayName":"Count Template","prompt":"do something"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(tmplBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: status %d, body: %s", rec.Code, rec.Body.String())
	}

	// Check counts
	req = httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("counts: status %d", rec.Code)
	}

	var resp server.CountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Projects != 1 {
		t.Errorf("expected 1 project, got %d", resp.Projects)
	}
	if resp.Templates != 1 {
		t.Errorf("expected 1 template, got %d", resp.Templates)
	}
}

// ── POST /api/v1/runs/{id}/archive ──

func TestREST_Archive_NotFound(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/nonexistent/archive",
		bytes.NewBufferString(`{"archived":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp struct{ Error string }
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// ── POST /api/v1/runs/bulk-archive ──

func TestREST_BulkArchive_EmptyRunIDs_Returns200(t *testing.T) {
	// handleBulkArchive does not reject an empty runIds list; it simply
	// iterates over zero IDs and returns archived=0 with no errors.
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/bulk-archive",
		bytes.NewBufferString(`{"runIds":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for empty runIds, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v, _ := resp["archived"].(float64); v != 0 {
		t.Errorf("expected archived=0, got %v", v)
	}
}

func TestREST_BulkArchive_InvalidJSON(t *testing.T) {
	// Malformed JSON must return 400.
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/bulk-archive",
		bytes.NewBufferString(`not-valid-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ── GET /api/v1/projects ──

func TestREST_Projects_CRUD_HappyPath(t *testing.T) {
	mux := buildMux(t)

	// Create
	createBody := `{"name":"int-proj","displayName":"Integration Project","repos":[{"url":"https://github.com/org/repo","branch":"main"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: status %d, body: %s", rec.Code, rec.Body.String())
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list projects: status %d", rec.Code)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 project, got %d", len(list))
	}

	// Get
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/int-proj", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get project: status %d, body: %s", rec.Code, rec.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/int-proj", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete project: status %d, body: %s", rec.Code, rec.Body.String())
	}

	// Get after delete — expect 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/int-proj", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestREST_Project_Create_MissingName(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects",
		bytes.NewBufferString(`{"displayName":"no name"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Project_Get_NotFound(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Project_Update_NotFound(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/does-not-exist",
		bytes.NewBufferString(`{"displayName":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for update of non-existent project, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ── Chain + Schedule REST endpoints ──

func TestREST_Chain_CRUD_HappyPath(t *testing.T) {
	mux := buildMux(t)

	// Create template first (needed as step reference).
	tmplBody := `{"name":"t1","prompt":"step 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(tmplBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d %s", rec.Code, rec.Body.String())
	}

	// Create chain
	chainBody := `{"name":"my-chain","steps":[{"name":"s1","templateRef":"t1"}]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/chains", bytes.NewBufferString(chainBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create chain: %d %s", rec.Code, rec.Body.String())
	}

	// Get chain
	req = httptest.NewRequest(http.MethodGet, "/api/v1/chains/my-chain", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get chain: %d %s", rec.Code, rec.Body.String())
	}

	// Delete chain
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/chains/my-chain", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete chain: %d %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Chain_Create_MissingSteps(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chains",
		bytes.NewBufferString(`{"name":"bad-chain","steps":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty steps, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Schedule_Create_MissingCron(t *testing.T) {
	mux := buildMux(t)

	body := `{"name":"bad-sched","chainRef":"some-chain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing cron, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Schedule_Create_BothChainAndTemplateRef(t *testing.T) {
	mux := buildMux(t)

	body := `{"name":"conflict-sched","cron":"0 * * * *","chainRef":"c","templateRef":"t"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for mutually exclusive chainRef+templateRef, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Schedule_Create_NeitherRef(t *testing.T) {
	mux := buildMux(t)

	body := `{"name":"no-ref-sched","cron":"0 * * * *"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when neither chainRef nor templateRef is set, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestREST_Schedule_SuspendResume(t *testing.T) {
	mux := buildMux(t)

	// Create schedule
	body := `{"name":"toggle-sched","cron":"0 12 * * *","chainRef":"some-chain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create schedule: %d %s", rec.Code, rec.Body.String())
	}

	// Suspend
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedules/toggle-sched/suspend", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend: %d %s", rec.Code, rec.Body.String())
	}
	var suspResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &suspResp); err != nil {
		t.Fatalf("unmarshal suspend: %v", err)
	}
	if v, _ := suspResp["suspended"].(bool); !v {
		t.Errorf("expected suspended=true, got %v", suspResp["suspended"])
	}

	// Resume
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedules/toggle-sched/resume", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resume: %d %s", rec.Code, rec.Body.String())
	}
	var resumeResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resumeResp); err != nil {
		t.Fatalf("unmarshal resume: %v", err)
	}
	if v, _ := resumeResp["suspended"].(bool); v {
		t.Errorf("expected suspended=false after resume, got %v", resumeResp["suspended"])
	}
}

func TestREST_Schedule_Suspend_NotFound(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/does-not-exist/suspend", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ── isValidRepoPath (project file-path safety) ──

func TestREST_ProjectFiles_NoSoftServe_ServiceUnavailable(t *testing.T) {
	// Without SoftServe configured, file operations must return 503.
	mux := buildMux(t)

	for _, path := range []string{
		"/api/v1/projects/myproj/files",
		"/api/v1/projects/myproj/files/README.md",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s: expected 503 without SoftServe, got %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestREST_ProjectFiles_WriteFile_NoSoftServe(t *testing.T) {
	mux := buildMux(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/myproj/files/README.md",
		bytes.NewBufferString(`{"content":"hello","commitMessage":"init"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 without SoftServe, got %d: %s", rec.Code, rec.Body.String())
	}
}
