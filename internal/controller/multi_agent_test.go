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
	"github.com/uncworks/aot/internal/testutil"
)

func setupTestEnv(t *testing.T) (client.Client, func()) {
	testutil.EnsureEnvtestAssets()
	t.Helper()

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
		t.Fatalf("create client: %v", err)
	}

	return k8sClient, func() {
		_ = testEnv.Stop()
	}
}

func TestSpawnJunior(t *testing.T) {
	k8sClient, cleanup := setupTestEnv(t)
	defer cleanup()

	ctx := context.Background()

	// Create parent
	parent := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "senior-run",
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Repos:   []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:  "Senior task",
		},
	}
	if err := k8sClient.Create(ctx, parent); err != nil {
		t.Fatalf("create parent: %v", err)
	}

	// Spawn junior
	junior, err := SpawnJunior(ctx, k8sClient, scheme.Scheme, parent, "Fix the CSS layout")
	if err != nil {
		t.Fatalf("SpawnJunior: %v", err)
	}

	if junior.Spec.Prompt != "Fix the CSS layout" {
		t.Errorf("expected junior prompt, got %q", junior.Spec.Prompt)
	}
	if junior.Labels["aot.uncworks.io/parent"] != "senior-run" {
		t.Errorf("expected parent label, got %q", junior.Labels["aot.uncworks.io/parent"])
	}
	if junior.Labels["aot.uncworks.io/role"] != "junior" {
		t.Errorf("expected junior role label")
	}

	// List juniors
	juniors, err := ListJuniors(ctx, k8sClient, "senior-run", "default")
	if err != nil {
		t.Fatalf("ListJuniors: %v", err)
	}
	if len(juniors) != 1 {
		t.Fatalf("expected 1 junior, got %d", len(juniors))
	}
}
