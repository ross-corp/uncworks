package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
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

	aotv1alpha1.AddToScheme(scheme.Scheme)

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		testEnv.Stop()
		t.Fatalf("create client: %v", err)
	}

	reconciler := &AgentRunReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}

	return reconciler, k8sClient, func() { testEnv.Stop() }
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
// 1. buildAgentPod
// ---------------------------------------------------------------------------

func TestBuildAgentPod_DefaultImage(t *testing.T) {
	r := &AgentRunReconciler{}
	ar := newAgentRun("test-build")
	pod := r.buildAgentPod(ar, "agentrun-test-build")

	// Pod metadata
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

	// Init container
	if len(pod.Spec.InitContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(pod.Spec.InitContainers))
	}
	init := pod.Spec.InitContainers[0]
	if init.Name != "hydration" {
		t.Errorf("expected init container name hydration, got %s", init.Name)
	}
	if init.Image != initImage {
		t.Errorf("expected init image %s, got %s", initImage, init.Image)
	}

	// Containers: agent + sidecar
	if len(pod.Spec.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(pod.Spec.Containers))
	}

	agent := pod.Spec.Containers[0]
	if agent.Name != "agent" {
		t.Errorf("expected agent container, got %s", agent.Name)
	}
	if agent.Image != defaultAgentImage {
		t.Errorf("expected default image %s, got %s", defaultAgentImage, agent.Image)
	}

	sidecar := pod.Spec.Containers[1]
	if sidecar.Name != "rpc-gateway" {
		t.Errorf("expected rpc-gateway container, got %s", sidecar.Name)
	}
	if sidecar.Image != sidecarImage {
		t.Errorf("expected sidecar image %s, got %s", sidecarImage, sidecar.Image)
	}
	if len(sidecar.Ports) != 1 || sidecar.Ports[0].ContainerPort != 50052 {
		t.Error("expected sidecar grpc port 50052")
	}

	// Volumes
	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(pod.Spec.Volumes))
	}
	if pod.Spec.Volumes[0].Name != "workspace" {
		t.Error("expected workspace volume")
	}
	if pod.Spec.Volumes[0].EmptyDir == nil {
		t.Error("expected emptyDir volume source")
	}

	// Volume mounts on init and agent containers
	if len(init.VolumeMounts) != 1 || init.VolumeMounts[0].MountPath != "/workspace" {
		t.Error("init container missing /workspace mount")
	}
	if len(agent.VolumeMounts) != 1 || agent.VolumeMounts[0].MountPath != "/workspace" {
		t.Error("agent container missing /workspace mount")
	}

	// Restart policy
	if pod.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("expected RestartPolicyNever, got %s", pod.Spec.RestartPolicy)
	}
}

func TestBuildAgentPod_CustomImage(t *testing.T) {
	r := &AgentRunReconciler{}
	ar := newAgentRun("custom-img", func(a *aotv1alpha1.AgentRun) {
		a.Spec.Image = "my-registry.io/agent:v2"
	})
	pod := r.buildAgentPod(ar, "agentrun-custom-img")

	if pod.Spec.Containers[0].Image != "my-registry.io/agent:v2" {
		t.Errorf("expected custom image, got %s", pod.Spec.Containers[0].Image)
	}
}

func TestBuildAgentPod_EnvVars(t *testing.T) {
	r := &AgentRunReconciler{}
	ar := newAgentRun("env-test", func(a *aotv1alpha1.AgentRun) {
		a.Spec.EnvVars = map[string]string{"CUSTOM_KEY": "CUSTOM_VAL"}
	})
	pod := r.buildAgentPod(ar, "agentrun-env-test")

	agentEnv := pod.Spec.Containers[0].Env
	envMap := make(map[string]string, len(agentEnv))
	for _, e := range agentEnv {
		envMap[e.Name] = e.Value
	}

	// Core env vars
	for _, key := range []string{"AOT_AGENT_RUN_ID", "AOT_REPO_URL", "AOT_BRANCH", "AOT_PROMPT"} {
		if _, ok := envMap[key]; !ok {
			t.Errorf("missing expected env var %s", key)
		}
	}
	if envMap["AOT_AGENT_RUN_ID"] != "env-test" {
		t.Errorf("expected AOT_AGENT_RUN_ID=env-test, got %s", envMap["AOT_AGENT_RUN_ID"])
	}

	// Custom env var
	if envMap["CUSTOM_KEY"] != "CUSTOM_VAL" {
		t.Errorf("expected CUSTOM_KEY=CUSTOM_VAL, got %s", envMap["CUSTOM_KEY"])
	}

	// Sidecar should only have AOT_AGENT_RUN_ID
	sidecarEnv := pod.Spec.Containers[1].Env
	if len(sidecarEnv) != 1 || sidecarEnv[0].Name != "AOT_AGENT_RUN_ID" {
		t.Error("sidecar should only have AOT_AGENT_RUN_ID env var")
	}
}

// ---------------------------------------------------------------------------
// 2. reconcilePod - full integration via envtest
// ---------------------------------------------------------------------------

func TestReconcilePod_CreatesPod(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("reconcile-pod")
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	// Run reconcilePod
	result, err := reconciler.reconcilePod(ctx, ar)
	if err != nil {
		t.Fatalf("reconcilePod: %v", err)
	}
	if result.RequeueAfter != 30*time.Second {
		t.Errorf("expected 30s requeue, got %v", result.RequeueAfter)
	}

	// Verify pod was created
	podName := fmt.Sprintf("agentrun-%s", ar.Name)
	var pod corev1.Pod
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: podName}, &pod); err != nil {
		t.Fatalf("get pod: %v", err)
	}
	if pod.Spec.Containers[0].Name != "agent" {
		t.Error("pod missing agent container")
	}

	// Verify status was updated
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get agentrun: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseRunning {
		t.Errorf("expected Running phase, got %s", updated.Status.Phase)
	}
	if updated.Status.PodName != podName {
		t.Errorf("expected podName %s, got %s", podName, updated.Status.PodName)
	}
	if updated.Status.StartedAt == nil {
		t.Error("expected startedAt to be set")
	}
}

// ---------------------------------------------------------------------------
// 3. syncPodStatus - verify phase transitions
// ---------------------------------------------------------------------------

func TestSyncPodStatus_Succeeded(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("sync-success")
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}
	// Set initial running status
	ar.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "agentrun-sync-success", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	}

	_, err := reconciler.syncPodStatus(ctx, ar, pod)
	if err != nil {
		t.Fatalf("syncPodStatus: %v", err)
	}

	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseSucceeded {
		t.Errorf("expected Succeeded, got %s", updated.Status.Phase)
	}
	if updated.Status.CompletedAt == nil {
		t.Error("expected completedAt to be set")
	}
	if updated.Status.Message != "Agent completed successfully" {
		t.Errorf("unexpected message: %s", updated.Status.Message)
	}
}

func TestSyncPodStatus_Failed(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("sync-fail")
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}
	ar.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "agentrun-sync-fail", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodFailed},
	}

	_, err := reconciler.syncPodStatus(ctx, ar, pod)
	if err != nil {
		t.Fatalf("syncPodStatus: %v", err)
	}

	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseFailed {
		t.Errorf("expected Failed, got %s", updated.Status.Phase)
	}
	if updated.Status.Message != "Agent pod failed" {
		t.Errorf("unexpected message: %s", updated.Status.Message)
	}
}

func TestSyncPodStatus_Running(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("sync-running")
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "agentrun-sync-running", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}

	result, err := reconciler.syncPodStatus(ctx, ar, pod)
	if err != nil {
		t.Fatalf("syncPodStatus: %v", err)
	}
	if result.RequeueAfter != 30*time.Second {
		t.Errorf("expected 30s requeue for running pod, got %v", result.RequeueAfter)
	}
}

// ---------------------------------------------------------------------------
// 4. TTL enforcement
// ---------------------------------------------------------------------------

func TestReconcilePod_TTLExpired(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	ctx := context.Background()

	ar := newAgentRun("ttl-expired", func(a *aotv1alpha1.AgentRun) {
		a.Spec.TTLSeconds = 1 // 1 second TTL
	})
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	// Set status to Running with a StartedAt in the past
	pastTime := metav1.NewTime(time.Now().Add(-10 * time.Second))
	ar.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	ar.Status.StartedAt = &pastTime
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// reconcilePod should detect TTL exceeded (pod doesn't exist, status is Running with expired start)
	_, err := reconciler.reconcilePod(ctx, ar)
	if err != nil {
		t.Fatalf("reconcilePod: %v", err)
	}

	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseFailed {
		t.Errorf("expected Failed phase after TTL, got %s", updated.Status.Phase)
	}
	if updated.Status.Message != "Exceeded TTL" {
		t.Errorf("expected TTL message, got %s", updated.Status.Message)
	}
	if updated.Status.CompletedAt == nil {
		t.Error("expected completedAt to be set after TTL expiry")
	}
}

// ---------------------------------------------------------------------------
// 5. handleNotImplemented
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
