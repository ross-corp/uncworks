package controller

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	temporalmocks "go.temporal.io/sdk/mocks"

	"github.com/stretchr/testify/mock"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/internal/eventbus"
	aottemporal "github.com/uncworks/aot/internal/temporal"
	"github.com/uncworks/aot/internal/testutil"
)

// setupReconciler creates an envtest environment and returns a configured
// AgentRunReconciler along with a raw client and cleanup function.
func setupReconciler(t *testing.T) (*AgentRunReconciler, client.Client, func()) {
	t.Helper()
	testutil.EnsureEnvtestAssets()

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deploy", "crds")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}

	if err := aotv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		_ = testEnv.Stop()
		t.Fatalf("add scheme: %v", err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		_ = testEnv.Stop()
		t.Fatalf("create client: %v", err)
	}

	reconciler := &AgentRunReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
		// TemporalClient is nil — tests that need it must set it explicitly
	}

	return reconciler, k8sClient, func() { _ = testEnv.Stop() }
}

// newAgentRun is a helper that creates a minimal AgentRun with sensible defaults.
func newAgentRun(name string, opts ...func(*aotv1alpha1.AgentRun)) *aotv1alpha1.AgentRun {
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:     "do the thing",
			TTLSeconds: 3600,
		},
	}
	for _, fn := range opts {
		fn(ar)
	}
	return ar
}

// ---------------------------------------------------------------------------
// 1. BuildAgentPod (shared function in temporal package)
// ---------------------------------------------------------------------------

func TestBuildAgentPod_DefaultImage(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:         "agentrun-test-build",
		Namespace:    "default",
		AgentRunName: "test-build",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "do the thing",
	}
	pod := aottemporal.BuildAgentPod(input)

	if pod.Name != "agentrun-test-build" {
		t.Errorf("expected pod name agentrun-test-build, got %s", pod.Name)
	}
	if pod.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", pod.Namespace)
	}
	if pod.Labels["app.kubernetes.io/name"] != "aot-agent" {
		t.Error("missing app name label")
	}
	if pod.Labels["aot.uncworks.io/agentrun"] != "test-build" {
		t.Error("missing agentrun label")
	}

	if len(pod.Spec.InitContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(pod.Spec.InitContainers))
	}
	if pod.Spec.InitContainers[0].Name != "hydration" {
		t.Errorf("expected init container name hydration, got %s", pod.Spec.InitContainers[0].Name)
	}

	if len(pod.Spec.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(pod.Spec.Containers))
	}
	if pod.Spec.Containers[0].Name != "agent" {
		t.Errorf("expected agent container, got %s", pod.Spec.Containers[0].Name)
	}
	if pod.Spec.Containers[1].Name != "rpc-gateway" {
		t.Errorf("expected rpc-gateway container, got %s", pod.Spec.Containers[1].Name)
	}
	if len(pod.Spec.Containers[1].Ports) != 1 || pod.Spec.Containers[1].Ports[0].ContainerPort != 50052 {
		t.Error("expected sidecar grpc port 50052")
	}

	if len(pod.Spec.Volumes) != 1 || pod.Spec.Volumes[0].Name != "workspace" {
		t.Error("expected workspace volume")
	}
}

func TestBuildAgentPod_CustomImage(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:         "agentrun-custom",
		Namespace:    "default",
		AgentRunName: "custom",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "do the thing",
		Image:        "my-registry.io/agent:v2",
	}
	pod := aottemporal.BuildAgentPod(input)

	if pod.Spec.Containers[0].Image != "my-registry.io/agent:v2" {
		t.Errorf("expected custom image, got %s", pod.Spec.Containers[0].Image)
	}
}

func TestBuildAgentPod_EnvVars(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:         "agentrun-env",
		Namespace:    "default",
		AgentRunName: "env-test",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "do the thing",
		EnvVars:      map[string]string{"CUSTOM_KEY": "CUSTOM_VAL"},
	}
	pod := aottemporal.BuildAgentPod(input)

	agentEnv := pod.Spec.Containers[0].Env
	envMap := make(map[string]string, len(agentEnv))
	for _, e := range agentEnv {
		envMap[e.Name] = e.Value
	}

	for _, key := range []string{"AOT_AGENT_RUN_ID", "AOT_REPOS", "AOT_PROMPT"} {
		if _, ok := envMap[key]; !ok {
			t.Errorf("missing expected env var %s", key)
		}
	}
	if envMap["AOT_AGENT_RUN_ID"] != "env-test" {
		t.Errorf("expected AOT_AGENT_RUN_ID=env-test, got %s", envMap["AOT_AGENT_RUN_ID"])
	}
	if envMap["CUSTOM_KEY"] != "CUSTOM_VAL" {
		t.Errorf("expected CUSTOM_KEY=CUSTOM_VAL, got %s", envMap["CUSTOM_KEY"])
	}

	sidecarEnv := pod.Spec.Containers[1].Env
	sidecarEnvMap := make(map[string]string, len(sidecarEnv))
	for _, e := range sidecarEnv {
		sidecarEnvMap[e.Name] = e.Value
	}
	if sidecarEnvMap["AOT_AGENT_RUN_ID"] != "env-test" {
		t.Errorf("sidecar should have AOT_AGENT_RUN_ID=env-test, got %s", sidecarEnvMap["AOT_AGENT_RUN_ID"])
	}
}

func TestBuildAgentPod_LLMEnvVars(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:           "agentrun-llm",
		Namespace:      "default",
		AgentRunName:   "llm-test",
		Repos:          []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:         "do the thing",
		LLMKey:         "sk-test-key-123",
		LiteLLMBaseURL: "http://litellm:4000",
	}
	pod := aottemporal.BuildAgentPod(input)

	// Agent container should have OPENAI_BASE_URL and OPENAI_API_KEY
	agentEnv := pod.Spec.Containers[0].Env
	envMap := make(map[string]string, len(agentEnv))
	for _, e := range agentEnv {
		envMap[e.Name] = e.Value
	}

	if envMap["OPENAI_BASE_URL"] != "http://litellm:4000/v1" {
		t.Errorf("expected OPENAI_BASE_URL=http://litellm:4000/v1, got %s", envMap["OPENAI_BASE_URL"])
	}
	if envMap["OPENAI_API_KEY"] != "sk-test-key-123" {
		t.Errorf("expected OPENAI_API_KEY=sk-test-key-123, got %s", envMap["OPENAI_API_KEY"])
	}

	// Init container should NOT have LLM env vars
	initEnv := pod.Spec.InitContainers[0].Env
	for _, e := range initEnv {
		if e.Name == "OPENAI_BASE_URL" || e.Name == "OPENAI_API_KEY" {
			t.Errorf("init container should not have %s", e.Name)
		}
	}

	// Sidecar SHOULD have LLM env vars (pi-coding-agent runs in the sidecar)
	sidecarEnv := pod.Spec.Containers[1].Env
	sidecarEnvMap := make(map[string]string, len(sidecarEnv))
	for _, e := range sidecarEnv {
		sidecarEnvMap[e.Name] = e.Value
	}
	if sidecarEnvMap["OPENAI_BASE_URL"] != "http://litellm:4000/v1" {
		t.Errorf("sidecar should have OPENAI_BASE_URL, got %s", sidecarEnvMap["OPENAI_BASE_URL"])
	}
	if sidecarEnvMap["OPENAI_API_KEY"] != "sk-test-key-123" {
		t.Errorf("sidecar should have OPENAI_API_KEY, got %s", sidecarEnvMap["OPENAI_API_KEY"])
	}
}

func TestBuildAgentPod_NoLLMEnvVars(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:         "agentrun-nollm",
		Namespace:    "default",
		AgentRunName: "nollm-test",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "do the thing",
	}
	pod := aottemporal.BuildAgentPod(input)

	agentEnv := pod.Spec.Containers[0].Env
	for _, e := range agentEnv {
		if e.Name == "OPENAI_BASE_URL" || e.Name == "OPENAI_API_KEY" {
			t.Errorf("agent container should not have %s when LLM not configured", e.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. mapPhase
// ---------------------------------------------------------------------------

func TestMapPhase(t *testing.T) {
	tests := []struct {
		input    string
		expected aotv1alpha1.AgentRunPhase
	}{
		{"Pending", aotv1alpha1.AgentRunPhasePending},
		{"Creating", aotv1alpha1.AgentRunPhasePending},
		{"Hydrating", aotv1alpha1.AgentRunPhasePending},
		{"Running", aotv1alpha1.AgentRunPhaseRunning},
		{"WaitingForInput", aotv1alpha1.AgentRunPhaseWaitingForInput},
		{"Succeeded", aotv1alpha1.AgentRunPhaseSucceeded},
		{"Failed", aotv1alpha1.AgentRunPhaseFailed},
		{"Cancelled", aotv1alpha1.AgentRunPhaseCancelled},
		{"Cancelling", aotv1alpha1.AgentRunPhaseCancelled},
		{"Unknown", aotv1alpha1.AgentRunPhasePending},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapPhase(tt.input)
			if got != tt.expected {
				t.Errorf("mapPhase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	if !isTerminal(aotv1alpha1.AgentRunPhaseSucceeded) {
		t.Error("Succeeded should be terminal")
	}
	if !isTerminal(aotv1alpha1.AgentRunPhaseFailed) {
		t.Error("Failed should be terminal")
	}
	if !isTerminal(aotv1alpha1.AgentRunPhaseCancelled) {
		t.Error("Cancelled should be terminal")
	}
	if isTerminal(aotv1alpha1.AgentRunPhaseRunning) {
		t.Error("Running should not be terminal")
	}
	if isTerminal(aotv1alpha1.AgentRunPhasePending) {
		t.Error("Pending should not be terminal")
	}
}

// ---------------------------------------------------------------------------
// 3. EventBus integration
// ---------------------------------------------------------------------------

// recordingBus captures published events for test assertions.
type recordingBus struct {
	mu     sync.Mutex
	events []*apiv1.AgentRunEvent
}

func (r *recordingBus) Publish(_ string, event *apiv1.AgentRunEvent) {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()
}

func (r *recordingBus) Subscribe(string) (<-chan *apiv1.AgentRunEvent, int) {
	return make(chan *apiv1.AgentRunEvent, 1), 0
}

func (r *recordingBus) Unsubscribe(string, int) {}

func TestEventBus_NilBusDoesNotPanic(t *testing.T) {
	reconciler := &AgentRunReconciler{}
	// EventBus is nil by default — emitPhaseEvent should not panic
	ar := newAgentRun("no-bus")
	reconciler.emitPhaseEvent(ar, apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED)
	// No panic means pass
}

// Verify the interface is satisfied at compile time.
var _ eventbus.EventBus = (*recordingBus)(nil)

// ---------------------------------------------------------------------------
// 4. Reconcile lifecycle tests using fake client + temporal mocks
// ---------------------------------------------------------------------------

// newFakeReconciler creates an AgentRunReconciler backed by a fake k8s client
// and a testify-mock Temporal client. It pre-seeds the given objects.
func newFakeReconciler(t *testing.T, tc *temporalmocks.Client, objs ...client.Object) (*AgentRunReconciler, client.Client) {
	t.Helper()
	s := newScheme()
	k8s := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		WithStatusSubresource(&aotv1alpha1.AgentRun{}).
		Build()
	r := &AgentRunReconciler{
		Client:         k8s,
		Scheme:         s,
		TemporalClient: tc,
		TaskQueue:      "test-queue",
	}
	return r, k8s
}

// TestReconcile_NewAgentRunTransitionsToPending verifies that a brand-new
// AgentRun does not get stuck: the first reconcile adds the finalizer (and
// returns), the second reconcile starts the Temporal workflow and sets the
// phase to Running.
func TestReconcile_NewAgentRunTransitionsToPending(t *testing.T) {
	ar := newAgentRun("new-run")

	tc := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("GetID").Return("agentrun-new-run")
	mockRun.On("GetRunID").Return("run-id-1")
	// ExecuteWorkflow will be called on the second reconcile.
	// Use mock.Anything for the workflow func and input args since testify
	// cannot match function values directly.
	tc.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRun, nil)

	r, k8s := newFakeReconciler(t, tc, ar)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "new-run", Namespace: "default"}}

	// First reconcile: must add finalizer and return without starting workflow.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}
	var after1 aotv1alpha1.AgentRun
	if err := k8s.Get(context.Background(), req.NamespacedName, &after1); err != nil {
		t.Fatalf("get after reconcile 1: %v", err)
	}
	if !hasFinalizer(after1.Finalizers, finalizerName) {
		t.Error("expected finalizer to be added after first reconcile")
	}
	// Workflow must NOT have started yet.
	if after1.Annotations[annotationWorkflowID] != "" {
		t.Error("workflow should not start in the same reconcile as finalizer add")
	}

	// Second reconcile: must start the workflow and set the annotation.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (start workflow): %v", err)
	}

	var after2 aotv1alpha1.AgentRun
	if err := k8s.Get(context.Background(), req.NamespacedName, &after2); err != nil {
		t.Fatalf("get after reconcile 2: %v", err)
	}
	if after2.Annotations[annotationWorkflowID] == "" {
		t.Error("expected workflow-id annotation to be set after second reconcile")
	}
	if after2.Status.Phase != aotv1alpha1.AgentRunPhaseRunning {
		t.Errorf("expected phase Running, got %s", after2.Status.Phase)
	}
}

// TestReconcile_DeletedAgentRunWithFinalizer verifies that a deleted AgentRun
// that has a finalizer completes deletion: the finalizer is removed and no
// error is returned.
func TestReconcile_DeletedAgentRunWithFinalizer(t *testing.T) {
	now := metav1.NewTime(time.Now())
	ar := newAgentRun("del-run", func(a *aotv1alpha1.AgentRun) {
		a.Finalizers = []string{finalizerName}
		a.Annotations = map[string]string{annotationWorkflowID: "agentrun-del-run"}
		a.DeletionTimestamp = &now
	})

	tc := &temporalmocks.Client{}
	// CancelWorkflow should be called once; simulate success.
	tc.On("CancelWorkflow", context.Background(), "agentrun-del-run", "").Return(nil)

	r, k8s := newFakeReconciler(t, tc, ar)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "del-run", Namespace: "default"}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (deletion): %v", err)
	}

	// After removing the last finalizer, the fake client deletes the object.
	// NotFound is the expected outcome — it proves the finalizer was removed.
	var afterDel aotv1alpha1.AgentRun
	if err := k8s.Get(context.Background(), req.NamespacedName, &afterDel); err == nil {
		if hasFinalizer(afterDel.Finalizers, finalizerName) {
			t.Error("expected finalizer to be removed after deletion reconcile")
		}
	}

	tc.AssertCalled(t, "CancelWorkflow", context.Background(), "agentrun-del-run", "")
}

// TestReconcile_ArchivedRunSetsAnnotation verifies that when status.Archived=true the
// reconciler marks the object with the annotationArchived annotation exactly once and
// does not attempt to delete resources on subsequent reconcile passes.
func TestReconcile_ArchivedRunSetsAnnotation(t *testing.T) {
	ar := newAgentRun("archived-run", func(a *aotv1alpha1.AgentRun) {
		a.Finalizers = []string{finalizerName}
		a.Annotations = map[string]string{annotationWorkflowID: "agentrun-archived-run"}
		a.Status.Archived = true
		a.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
	})

	tc := &temporalmocks.Client{}
	r, k8s := newFakeReconciler(t, tc, ar)

	// Seed the archived status (fake client requires separate status update).
	ar.Status.Archived = true
	if err := k8s.Status().Update(context.Background(), ar); err != nil {
		t.Fatalf("seed status: %v", err)
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "archived-run", Namespace: "default"}}

	// First reconcile: should set annotationArchived and return.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile 1: %v", err)
	}

	var after1 aotv1alpha1.AgentRun
	if err := k8s.Get(context.Background(), req.NamespacedName, &after1); err != nil {
		t.Fatalf("get after reconcile 1: %v", err)
	}
	if after1.Annotations[annotationArchived] != "true" {
		t.Errorf("expected annotationArchived=true after first reconcile, got %v", after1.Annotations)
	}

	// Second reconcile: alreadyCleaned==true, so no further work (no panics, no extra calls).
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile 2: %v", err)
	}
	// TemporalClient methods must not have been called.
	tc.AssertNotCalled(t, "CancelWorkflow")
}

// hasFinalizer is a local helper so tests don't import controllerutil.
func hasFinalizer(finalizers []string, name string) bool {
	for _, f := range finalizers {
		if f == name {
			return true
		}
	}
	return false
}
