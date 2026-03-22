package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
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
