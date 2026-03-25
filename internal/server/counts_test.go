package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func countsScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func TestCountsHandler_EmptySystem(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(countsScheme()).Build()
	h := &CountsHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterCountsHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var result CountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// All counts should be zero on an empty system
	if result.Runs != 0 {
		t.Errorf("runs = %d, want 0", result.Runs)
	}
	if result.ActiveRuns != 0 {
		t.Errorf("activeRuns = %d, want 0", result.ActiveRuns)
	}
	if result.Projects != 0 {
		t.Errorf("projects = %d, want 0", result.Projects)
	}
	if result.Templates != 0 {
		t.Errorf("templates = %d, want 0", result.Templates)
	}
	if result.Chains != 0 {
		t.Errorf("chains = %d, want 0", result.Chains)
	}
	if result.ChainRuns != 0 {
		t.Errorf("chainruns = %d, want 0", result.ChainRuns)
	}
	if result.Schedules != 0 {
		t.Errorf("schedules = %d, want 0", result.Schedules)
	}
}

func TestCountsHandler_ActiveRunsFiltering(t *testing.T) {
	runs := []aotv1alpha1.AgentRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run-running", Namespace: "default"},
			Status:     aotv1alpha1.AgentRunStatus{Phase: aotv1alpha1.AgentRunPhaseRunning},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run-pending", Namespace: "default"},
			Status:     aotv1alpha1.AgentRunStatus{Phase: aotv1alpha1.AgentRunPhasePending},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run-waiting", Namespace: "default"},
			Status:     aotv1alpha1.AgentRunStatus{Phase: aotv1alpha1.AgentRunPhaseWaitingForInput},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run-succeeded", Namespace: "default"},
			Status:     aotv1alpha1.AgentRunStatus{Phase: aotv1alpha1.AgentRunPhaseSucceeded},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run-failed", Namespace: "default"},
			Status:     aotv1alpha1.AgentRunStatus{Phase: aotv1alpha1.AgentRunPhaseFailed},
		},
	}

	objs := make([]runtime.Object, len(runs))
	for i := range runs {
		objs[i] = &runs[i]
	}

	k8s := fake.NewClientBuilder().
		WithScheme(countsScheme()).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&aotv1alpha1.AgentRun{}).
		Build()

	h := &CountsHandler{K8sClient: k8s, Namespace: "default"}

	mux := http.NewServeMux()
	h.RegisterCountsHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var result CountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Runs != 5 {
		t.Errorf("runs = %d, want 5", result.Runs)
	}
	if result.ActiveRuns != 3 {
		t.Errorf("activeRuns = %d, want 3 (running+pending+waiting)", result.ActiveRuns)
	}
}
