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

func archiveScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func TestArchiveHandler_ArchiveRun(t *testing.T) {
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ar-test1", Namespace: "default"},
		Spec:       aotv1alpha1.AgentRunSpec{Prompt: "test"},
	}
	k8s := fake.NewClientBuilder().WithScheme(archiveScheme()).WithObjects(run).WithStatusSubresource(run).Build()
	h := &ArchiveHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterArchiveHandlers(mux)

	body := `{"archived":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/ar-test1/archive", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !result["archived"] {
		t.Error("expected archived=true in response")
	}
}

func TestArchiveHandler_ArchiveNotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(archiveScheme()).Build()
	h := &ArchiveHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterArchiveHandlers(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/nonexistent/archive", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestArchiveHandler_BulkArchive(t *testing.T) {
	runs := []aotv1alpha1.AgentRun{
		{ObjectMeta: metav1.ObjectMeta{Name: "ar-1", Namespace: "default"}, Spec: aotv1alpha1.AgentRunSpec{Prompt: "a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "ar-2", Namespace: "default"}, Spec: aotv1alpha1.AgentRunSpec{Prompt: "b"}},
	}
	objs := make([]runtime.Object, len(runs))
	for i := range runs {
		objs[i] = &runs[i]
	}
	k8s := fake.NewClientBuilder().WithScheme(archiveScheme()).WithRuntimeObjects(objs...).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	h := &ArchiveHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterArchiveHandlers(mux)

	body := `{"runIds":["ar-1","ar-2","ar-nonexistent"],"archived":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/bulk-archive", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	archived, ok := result["archived"].(float64)
	if !ok || archived != 2 {
		t.Errorf("expected 2 archived, got %v", result["archived"])
	}
	errors, ok := result["errors"].([]interface{})
	if !ok || len(errors) != 1 {
		t.Errorf("expected 1 error (nonexistent), got %v", result["errors"])
	}
}
