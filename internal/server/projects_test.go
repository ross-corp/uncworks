package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

func projectScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func TestProjectHandler_ListProjects_Empty(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var projects []projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestProjectHandler_CreateAndGet(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	// Create
	body := `{"name":"test-proj","displayName":"Test Project","repos":[{"url":"https://github.com/org/repo","branch":"main"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201. body: %s", rec.Code, rec.Body.String())
	}

	var created projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if created.Name != "test-proj" {
		t.Errorf("name = %q, want test-proj", created.Name)
	}
	if created.DisplayName != "Test Project" {
		t.Errorf("displayName = %q, want Test Project", created.DisplayName)
	}

	// Get
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/test-proj", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rec.Code)
	}

	var got projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if got.Name != "test-proj" {
		t.Errorf("get name = %q, want test-proj", got.Name)
	}
}

func TestProjectHandler_CreateMissingName(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	body := `{"displayName":"No Name"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestProjectHandler_GetNotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestProjectHandler_Delete(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "del-proj", Namespace: "default"},
		Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Delete Me"},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/del-proj", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", rec.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/del-proj", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("after delete, get status = %d, want 404", rec.Code)
	}
}

// mockRepoManager implements softserve.RepoManager for testing file operations.
type mockRepoManager struct {
	files map[string]map[string]string // repo -> path -> content
}

func (m *mockRepoManager) CreateRepo(name string) error                      { return nil }
func (m *mockRepoManager) DeleteRepo(name string) error                      { return nil }
func (m *mockRepoManager) RepoExists(name string) (bool, error)              { return true, nil }
func (m *mockRepoManager) CloneURL(name string) string                       { return "ssh://test/" + name }
func (m *mockRepoManager) ScaffoldAndPush(_ softserve.ScaffoldProject) error { return nil }

func (m *mockRepoManager) ReadFile(repo, path string) (string, error) {
	if r, ok := m.files[repo]; ok {
		if c, ok := r[path]; ok {
			return c, nil
		}
	}
	return "", fmt.Errorf("not found")
}

func (m *mockRepoManager) WriteFile(repo, path, content, _ string) error {
	if m.files[repo] == nil {
		m.files[repo] = map[string]string{}
	}
	m.files[repo][path] = content
	return nil
}

func (m *mockRepoManager) ListFiles(repo string) ([]string, error) {
	var files []string
	for p := range m.files[repo] {
		files = append(files, p)
	}
	return files, nil
}

func TestProjectHandler_ListFiles(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "file-proj", Namespace: "default"},
		Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Files"},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	mock := &mockRepoManager{files: map[string]map[string]string{
		"file-proj": {
			"devbox.json":                 `{"packages":[]}`,
			"openspec/specs/auth/spec.md": "## Requirements",
		},
	}}
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default", SoftServe: mock}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/file-proj/files", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var files []string
	if err := json.Unmarshal(rec.Body.Bytes(), &files); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestProjectHandler_ReadFile(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "read-proj", Namespace: "default"},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	mock := &mockRepoManager{files: map[string]map[string]string{
		"read-proj": {"devbox.json": `{"packages":["go@1.22"]}`},
	}}
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default", SoftServe: mock}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/read-proj/files/devbox.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["content"] != `{"packages":["go@1.22"]}` {
		t.Errorf("content = %q", result["content"])
	}
}

func TestProjectHandler_WriteFile(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "write-proj", Namespace: "default"},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	mock := &mockRepoManager{files: map[string]map[string]string{
		"write-proj": {},
	}}
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default", SoftServe: mock}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	body := `{"content":"## New Spec\nThe system SHALL...","commitMessage":"add auth spec"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/write-proj/files/openspec/specs/auth/spec.md", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	// Verify the file was written to the mock
	if mock.files["write-proj"]["openspec/specs/auth/spec.md"] != "## New Spec\nThe system SHALL..." {
		t.Errorf("file not written correctly")
	}
}

func TestProjectHandler_NoSoftServe(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "no-ss", Namespace: "default"},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default", SoftServe: nil}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/no-ss/files", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestProjectHandler_ListWithProjects(t *testing.T) {
	projects := []aotv1alpha1.Project{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "proj-a", Namespace: "default"},
			Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Project A"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "proj-b", Namespace: "default"},
			Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Project B"},
		},
	}
	objs := make([]runtime.Object, len(projects))
	for i := range projects {
		objs[i] = &projects[i]
	}

	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithRuntimeObjects(objs...).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var list []projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 projects, got %d", len(list))
	}
}

func TestProjectHandler_UpdateProject(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "upd-proj", Namespace: "default"},
		Spec: aotv1alpha1.ProjectSpec{
			DisplayName: "Old Name",
			Devbox:      nil,
			Defaults:    nil,
		},
	}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(project).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	newDisplay := "New Name"
	body := map[string]interface{}{
		"displayName": newDisplay,
		"devbox": map[string]interface{}{
			"packages": []string{"go@1.22", "nodejs@20"},
		},
		"defaults": map[string]interface{}{
			"modelTier": "premium",
		},
	}
	encoded, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/upd-proj", bytes.NewBuffer(encoded))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var got projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DisplayName != newDisplay {
		t.Errorf("displayName = %q, want %q", got.DisplayName, newDisplay)
	}
	if got.Devbox == nil || len(got.Devbox.Packages) != 2 {
		t.Errorf("expected 2 devbox packages, got devbox: %v", got.Devbox)
	}
	if got.Defaults == nil || got.Defaults.ModelTier != "premium" {
		t.Errorf("expected defaults.modelTier=premium, got: %v", got.Defaults)
	}
}

func TestProjectHandler_UpdateProject_NotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()
	h := &ProjectHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterProjectHandlers(mux)

	body := `{"displayName":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestIsValidRepoPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"devbox.json", true},
		{"openspec/specs/auth/spec.md", true},
		{".devcontainer/devcontainer.json", true},
		{"", false},
		{"/etc/passwd", false},
		{"../../../etc/passwd", false},
		{"foo/../../bar", false},
		{".git/config", false},
		{".env", false},
		{".ssh/id_rsa", false},
		{"normal/path/file.txt", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isValidRepoPath(tt.path); got != tt.want {
				t.Errorf("isValidRepoPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
