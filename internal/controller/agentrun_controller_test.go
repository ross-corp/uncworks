package controller

import (
	"context"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
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
			RepoURL:    "https://github.com/example/repo.git",
			Branch:     "main",
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
	input := aottemporal.CreateAgentPodInput{
		Name:         "agentrun-test-build",
		Namespace:    "default",
		AgentRunName: "test-build",
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
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
	input := aottemporal.CreateAgentPodInput{
		Name:         "agentrun-custom",
		Namespace:    "default",
		AgentRunName: "custom",
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		Prompt:       "do the thing",
		Image:        "my-registry.io/agent:v2",
	}
	pod := aottemporal.BuildAgentPod(input)

	if pod.Spec.Containers[0].Image != "my-registry.io/agent:v2" {
		t.Errorf("expected custom image, got %s", pod.Spec.Containers[0].Image)
	}
}

func TestBuildAgentPod_EnvVars(t *testing.T) {
	input := aottemporal.CreateAgentPodInput{
		Name:         "agentrun-env",
		Namespace:    "default",
		AgentRunName: "env-test",
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		Prompt:       "do the thing",
		EnvVars:      map[string]string{"CUSTOM_KEY": "CUSTOM_VAL"},
	}
	pod := aottemporal.BuildAgentPod(input)

	agentEnv := pod.Spec.Containers[0].Env
	envMap := make(map[string]string, len(agentEnv))
	for _, e := range agentEnv {
		envMap[e.Name] = e.Value
	}

	for _, key := range []string{"AOT_AGENT_RUN_ID", "AOT_REPO_URL", "AOT_BRANCH", "AOT_PROMPT"} {
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
	if len(sidecarEnv) != 1 || sidecarEnv[0].Name != "AOT_AGENT_RUN_ID" {
		t.Error("sidecar should only have AOT_AGENT_RUN_ID env var")
	}
}

// ---------------------------------------------------------------------------
// 2. handleNotImplemented
// ---------------------------------------------------------------------------

func TestHandleNotImplemented_KubeVirt(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("not-impl-kv", func(a *aotv1alpha1.AgentRun) {
		a.Spec.Backend = aotv1alpha1.BackendKubeVirt
	})
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	_, err := reconciler.handleNotImplemented(ctx, ar, "KubeVirt")
	if err != nil {
		t.Fatalf("handleNotImplemented: %v", err)
	}

	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseFailed {
		t.Errorf("expected Failed, got %s", updated.Status.Phase)
	}
	if updated.Status.Message != "KubeVirt backend is not yet implemented" {
		t.Errorf("unexpected message: %s", updated.Status.Message)
	}
}

func TestHandleNotImplemented_External(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("not-impl-ext", func(a *aotv1alpha1.AgentRun) {
		a.Spec.Backend = aotv1alpha1.BackendExternal
	})
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	_, err := reconciler.handleNotImplemented(ctx, ar, "External")
	if err != nil {
		t.Fatalf("handleNotImplemented: %v", err)
	}

	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseFailed {
		t.Errorf("expected Failed, got %s", updated.Status.Phase)
	}
	if updated.Status.Message != "External backend is not yet implemented" {
		t.Errorf("unexpected message: %s", updated.Status.Message)
	}
}

// ---------------------------------------------------------------------------
// 3. mapPhase
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
