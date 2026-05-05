package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func chainScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func newChainHandler(objs ...runtime.Object) (*ChainHandler, *http.ServeMux) {
	k8s := fake.NewClientBuilder().WithScheme(chainScheme()).WithRuntimeObjects(objs...).Build()
	h := &ChainHandler{K8sClient: k8s, Namespace: "default"}
	mux := http.NewServeMux()
	h.RegisterChainHandlers(mux)
	return h, mux
}

// ── Template tests ──

func TestChainHandler_CreateTemplate_Valid(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name":"my-tmpl","displayName":"My Template","prompt":"do something"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var out aotv1alpha1.RunTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "my-tmpl" {
		t.Errorf("name = %q, want my-tmpl", out.Name)
	}
}

func TestChainHandler_CreateTemplate_MissingName(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"displayName":"No Name"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestChainHandler_DeleteTemplate_Success(t *testing.T) {
	tmpl := &aotv1alpha1.RunTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "del-tmpl", Namespace: "default"},
	}
	_, mux := newChainHandler(tmpl)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/del-tmpl", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body: %s", rec.Code, rec.Body.String())
	}

	// Verify gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/templates/del-tmpl", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("after delete, get status = %d, want 404", rec.Code)
	}
}

func TestChainHandler_DeleteTemplate_ConflictWhenChainReferences(t *testing.T) {
	tmpl := &aotv1alpha1.RunTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "tmpl-ref", Namespace: "default"},
	}
	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "chain-using-tmpl", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "step1", TemplateRef: "tmpl-ref"},
			},
		},
	}
	_, mux := newChainHandler(tmpl, chain)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/tmpl-ref", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 conflict", rec.Code)
	}
}

func TestChainHandler_DeleteTemplate_NotFound(t *testing.T) {
	_, mux := newChainHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// ── Chain tests ──

func TestChainHandler_CreateChain_Valid(t *testing.T) {
	tmpl := &aotv1alpha1.RunTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "t", Namespace: "default"},
	}
	_, mux := newChainHandler(tmpl)

	body := `{
		"name": "my-chain",
		"steps": [
			{"name": "A", "templateRef": "t"},
			{"name": "B", "templateRef": "t", "dependsOn": ["A"]}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create chain: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var out aotv1alpha1.Chain
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "my-chain" {
		t.Errorf("name = %q, want my-chain", out.Name)
	}
}

func TestChainHandler_CreateChain_InvalidDAG_Cycle(t *testing.T) {
	_, mux := newChainHandler()

	// A depends on B, B depends on A → cycle
	body := `{
		"name": "cycle-chain",
		"steps": [
			{"name": "A", "templateRef": "t", "dependsOn": ["B"]},
			{"name": "B", "templateRef": "t", "dependsOn": ["A"]}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for cycle DAG", rec.Code)
	}
}

func TestChainHandler_CreateChain_MissingName(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"steps": [{"name": "A", "templateRef": "t"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing name", rec.Code)
	}
}

func TestChainHandler_CreateChain_MissingSteps(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name": "no-steps"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing steps", rec.Code)
	}
}

func TestChainHandler_DeleteChain_Success(t *testing.T) {
	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "del-chain", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{{Name: "A", TemplateRef: "t"}},
		},
	}
	_, mux := newChainHandler(chain)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/chains/del-chain", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestChainHandler_DeleteChain_ConflictWhenScheduleReferences(t *testing.T) {
	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-chain", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{{Name: "A", TemplateRef: "t"}},
		},
	}
	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "weekly", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:     "0 9 * * 1",
			ChainRef: "sched-chain",
		},
	}
	_, mux := newChainHandler(chain, sched)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/chains/sched-chain", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 conflict", rec.Code)
	}
}

func TestChainHandler_DeleteChain_NotFound(t *testing.T) {
	_, mux := newChainHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/chains/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestChainHandler_ListChains_Empty(t *testing.T) {
	_, mux := newChainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chains", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

// ── Schedule tests ──

func TestChainHandler_CreateSchedule_WithChainRef(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name":"weekly","cron":"0 9 * * 1","chainRef":"my-chain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create schedule: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var out aotv1alpha1.Schedule
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "weekly" {
		t.Errorf("name = %q, want weekly", out.Name)
	}
	if out.Spec.ChainRef != "my-chain" {
		t.Errorf("chainRef = %q, want my-chain", out.Spec.ChainRef)
	}
}

func TestChainHandler_CreateSchedule_WithTemplateRef(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name":"daily","cron":"0 8 * * *","templateRef":"my-tmpl"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create schedule: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var out aotv1alpha1.Schedule
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Spec.TemplateRef != "my-tmpl" {
		t.Errorf("templateRef = %q, want my-tmpl", out.Spec.TemplateRef)
	}
}

func TestChainHandler_CreateSchedule_MissingCron(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name":"no-cron","chainRef":"my-chain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing cron", rec.Code)
	}
}

func TestChainHandler_CreateSchedule_MissingRef(t *testing.T) {
	_, mux := newChainHandler()

	body := `{"name":"no-ref","cron":"0 9 * * 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing chainRef/templateRef", rec.Code)
	}
}

func TestChainHandler_DeleteSchedule_Success(t *testing.T) {
	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "del-sched", Namespace: "default"},
		Spec:       aotv1alpha1.ScheduleSpec{Cron: "0 9 * * 1", ChainRef: "c"},
	}
	_, mux := newChainHandler(sched)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/del-sched", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestChainHandler_DeleteSchedule_NotFound(t *testing.T) {
	_, mux := newChainHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/ghost", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestChainHandler_SuspendResumeSchedule(t *testing.T) {
	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sr-sched", Namespace: "default"},
		Spec:       aotv1alpha1.ScheduleSpec{Cron: "0 9 * * 1", ChainRef: "c"},
	}
	_, mux := newChainHandler(sched)

	// Suspend
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sr-sched/suspend", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend status = %d, body: %s", rec.Code, rec.Body.String())
	}

	// Resume
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sr-sched/resume", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resume status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestCapList_NilInput_ReturnsEmptySlice(t *testing.T) {
	var items []aotv1alpha1.RunTemplate
	result := capList(items, maxListItems)
	if result == nil {
		t.Error("capList(nil, ...) returned nil, want empty slice")
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("JSON = %q, want []", string(b))
	}
}

func TestChainHandler_ListChainRuns_EmptyReturnsArray(t *testing.T) {
	_, mux := newChainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chainruns", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	body := strings.TrimSpace(rec.Body.String())
	if body == "null" {
		t.Error("GET /api/v1/chainruns returned JSON null, want []")
	}
	var out []aotv1alpha1.ChainRun
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out == nil {
		t.Error("deserialized response is nil slice")
	}
}

func TestCapList_TruncatesAt500(t *testing.T) {
	items := make([]aotv1alpha1.RunTemplate, 600)
	result := capList(items, maxListItems)
	if len(result) != 500 {
		t.Errorf("capList(600, 500) = %d items, want 500", len(result))
	}
}

func TestCapList_PassesThroughWhenUnder(t *testing.T) {
	items := make([]aotv1alpha1.RunTemplate, 10)
	result := capList(items, maxListItems)
	if len(result) != 10 {
		t.Errorf("capList(10, 500) = %d items, want 10", len(result))
	}
}

func TestChainHandler_ListTemplates_CapAt500(t *testing.T) {
	var objs []runtime.Object
	for i := 0; i < 600; i++ {
		tmpl := &aotv1alpha1.RunTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("tmpl-%d", i),
				Namespace: "default",
			},
		}
		objs = append(objs, tmpl)
	}
	_, mux := newChainHandler(objs...)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var out []aotv1alpha1.RunTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 500 {
		t.Errorf("list returned %d items, want 500", len(out))
	}
}
