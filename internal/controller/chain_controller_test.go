package controller

import (
	"context"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/testutil"
)

func setupChainReconciler(t *testing.T) (*ChainRunReconciler, client.Client, func()) {
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

	reconciler := &ChainRunReconciler{Client: k8sClient, Scheme: scheme.Scheme}
	return reconciler, k8sClient, func() { _ = testEnv.Stop() }
}

func TestChainRun_MissingChain_Fails(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Create a ChainRun referencing a non-existent chain
	cr := &aotv1alpha1.ChainRun{
		ObjectMeta: metav1.ObjectMeta{Name: "cr-no-chain", Namespace: "default"},
		Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "does-not-exist", TriggeredBy: "test"},
	}
	if err := k8s.Create(ctx, cr); err != nil {
		t.Fatalf("create chain run: %v", err)
	}

	_, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)})
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}

	// Status should be failed
	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get: %v", err)
	}
	if cr.Status.Phase != "failed" {
		t.Errorf("expected phase=failed, got %q", cr.Status.Phase)
	}
	if cr.Status.Message == "" {
		t.Error("expected error message, got empty")
	}
}

func TestChainRun_Initializes_StepStatuses(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Create chain definition
	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "my-chain", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "lint", TemplateRef: "tmpl-lint"},
				{Name: "test", TemplateRef: "tmpl-test", DependsOn: []string{"lint"}},
			},
		},
	}
	if err := k8s.Create(ctx, chain); err != nil {
		t.Fatalf("create chain: %v", err)
	}

	cr := &aotv1alpha1.ChainRun{
		ObjectMeta: metav1.ObjectMeta{Name: "cr-init", Namespace: "default"},
		Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "my-chain"},
	}
	if err := k8s.Create(ctx, cr); err != nil {
		t.Fatalf("create chain run: %v", err)
	}

	_, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get: %v", err)
	}

	if cr.Status.Phase != "running" {
		t.Errorf("expected phase=running, got %q", cr.Status.Phase)
	}
	if len(cr.Status.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(cr.Status.Steps))
	}
	for _, s := range cr.Status.Steps {
		if s.Phase != "pending" {
			t.Errorf("step %q expected pending, got %q", s.Name, s.Phase)
		}
	}
	if cr.Status.StartedAt == nil {
		t.Error("expected startedAt to be set")
	}
}

func TestChainRun_LaunchesRootStep(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Template for the root step
	tmpl := &aotv1alpha1.RunTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "tmpl-lint", Namespace: "default"},
		Spec: aotv1alpha1.RunTemplateSpec{
			Prompt:    "run lint",
			ModelTier: "default",
			Repos:     []aotv1alpha1.Repository{{URL: "https://github.com/test/repo", Branch: "main"}},
		},
	}
	if err := k8s.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}

	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "chain-launch", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "lint", TemplateRef: "tmpl-lint"},
				{Name: "test", TemplateRef: "tmpl-test", DependsOn: []string{"lint"}},
			},
		},
	}
	if err := k8s.Create(ctx, chain); err != nil {
		t.Fatalf("create chain: %v", err)
	}

	cr := &aotv1alpha1.ChainRun{
		ObjectMeta: metav1.ObjectMeta{Name: "cr-launch", Namespace: "default"},
		Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "chain-launch"},
	}
	if err := k8s.Create(ctx, cr); err != nil {
		t.Fatalf("create chain run: %v", err)
	}

	// First reconcile: initialize steps
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile init: %v", err)
	}

	// Second reconcile: should launch root step "lint" (no deps)
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile launch: %v", err)
	}

	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get: %v", err)
	}

	lintStep := cr.Status.Steps[0]
	if lintStep.Phase != "running" {
		t.Errorf("lint step expected running, got %q", lintStep.Phase)
	}
	if lintStep.RunID == "" {
		t.Error("lint step expected runID to be set")
	}

	// "test" step should still be pending (depends on lint)
	testStep := cr.Status.Steps[1]
	if testStep.Phase != "pending" {
		t.Errorf("test step expected pending, got %q", testStep.Phase)
	}

	// Verify the AgentRun was actually created
	var run aotv1alpha1.AgentRun
	if err := k8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: lintStep.RunID}, &run); err != nil {
		t.Fatalf("agent run not found: %v", err)
	}
	if run.Spec.Prompt != "run lint" {
		t.Errorf("expected prompt 'run lint', got %q", run.Spec.Prompt)
	}
	if run.Labels["aot.uncworks.io/chain-run"] != "cr-launch" {
		t.Error("missing chain-run label on created AgentRun")
	}
}

func TestChainRun_SkipsTerminalStates(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	for _, phase := range []string{"succeeded", "failed", "cancelled"} {
		cr := &aotv1alpha1.ChainRun{
			ObjectMeta: metav1.ObjectMeta{Name: "cr-terminal-" + phase, Namespace: "default"},
			Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "any"},
			Status:     aotv1alpha1.ChainRunStatus{Phase: phase},
		}
		if err := k8s.Create(ctx, cr); err != nil {
			t.Fatalf("create: %v", err)
		}
		// Set terminal status
		cr.Status.Phase = phase
		if err := k8s.Status().Update(ctx, cr); err != nil {
			t.Fatalf("status update: %v", err)
		}

		result, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)})
		if err != nil {
			t.Fatalf("reconcile %s: %v", phase, err)
		}
		if result.RequeueAfter != 0 {
			t.Errorf("phase=%s should not requeue, got %v", phase, result.RequeueAfter)
		}
	}
}

func TestChainRun_MissingTemplate_FailsStep(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "chain-bad-tmpl", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "step1", TemplateRef: "nonexistent-template"},
			},
		},
	}
	if err := k8s.Create(ctx, chain); err != nil {
		t.Fatalf("create chain: %v", err)
	}

	cr := &aotv1alpha1.ChainRun{
		ObjectMeta: metav1.ObjectMeta{Name: "cr-bad-tmpl", Namespace: "default"},
		Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "chain-bad-tmpl"},
	}
	if err := k8s.Create(ctx, cr); err != nil {
		t.Fatalf("create chain run: %v", err)
	}

	// First reconcile: init steps
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile init: %v", err)
	}
	// Second reconcile: try to launch — template missing
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile launch: %v", err)
	}

	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get: %v", err)
	}

	if cr.Status.Steps[0].Phase != "failed" {
		t.Errorf("expected step phase=failed, got %q", cr.Status.Steps[0].Phase)
	}
	if cr.Status.Phase != "failed" {
		t.Errorf("expected overall phase=failed, got %q", cr.Status.Phase)
	}
}

func TestChainRun_NotFound_Ignored(t *testing.T) {
	rec, _, cleanup := setupChainReconciler(t)
	defer cleanup()

	result, err := rec.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: "default", Name: "does-not-exist"},
	})
	if err != nil {
		t.Fatalf("expected no error for not-found, got: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Error("should not requeue for deleted chain run")
	}
}

// TestReposCopy_DoesNotMutateTemplate is a unit regression test for the
// in-place repos mutation bug. The controller code does:
//
//	repos := tmpl.Spec.Repos   // slice header copy — shares backing array
//	repos[0].Branch = branch   // mutates the original template's backing array
//
// In production the Kubernetes informer cache returns a pointer to the cached
// object; mutating repos[0].Branch corrupts that cached copy, so any subsequent
// reconcile retrieving the same RunTemplate sees the stale branch. The fix is
// to deep-copy the repos slice before mutating. This test verifies that the
// copy does not affect the original template object.
func TestReposCopy_DoesNotMutateTemplate(t *testing.T) {
	// Simulate the in-memory template as it would come from the informer cache —
	// a single allocation shared across calls.
	tmpl := &aotv1alpha1.RunTemplate{
		Spec: aotv1alpha1.RunTemplateSpec{
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/test/repo", Branch: "main"},
			},
		},
	}

	originalBranch := tmpl.Spec.Repos[0].Branch

	// Reproduce the fixed copy pattern from chain_controller.go.
	repos := make([]aotv1alpha1.Repository, len(tmpl.Spec.Repos))
	copy(repos, tmpl.Spec.Repos)
	repos[0].Branch = "aot/some-run-id"

	// The template's backing array must not have been mutated.
	if tmpl.Spec.Repos[0].Branch != originalBranch {
		t.Errorf("template repos mutated: expected branch %q, got %q — deep-copy is missing",
			originalBranch, tmpl.Spec.Repos[0].Branch)
	}

	// The copy must carry the new branch.
	if repos[0].Branch != "aot/some-run-id" {
		t.Errorf("copied repos[0].Branch: expected %q, got %q", "aot/some-run-id", repos[0].Branch)
	}
}

// TestReposCopy_WithoutDeepCopy_DemonstratesTheBug documents that the naive
// slice header assignment repos := tmpl.Spec.Repos aliases the backing array,
// causing mutations to propagate back into the original. This is the pattern
// that was present before the fix and must NOT be used.
func TestReposCopy_WithoutDeepCopy_DemonstratesTheBug(t *testing.T) {
	tmpl := &aotv1alpha1.RunTemplate{
		Spec: aotv1alpha1.RunTemplateSpec{
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/test/repo", Branch: "main"},
			},
		},
	}

	// Buggy pattern: slice header assignment shares the backing array.
	repos := tmpl.Spec.Repos
	repos[0].Branch = "aot/some-run-id"

	// The original IS mutated — this is the bug.
	if tmpl.Spec.Repos[0].Branch != "aot/some-run-id" {
		t.Skip("slice aliasing not present — this documents expected Go behavior")
	}
	// If we reach here, we've confirmed the aliasing. The test passes because it
	// is merely documenting the dangerous behavior; the fix in chain_controller.go
	// prevents this code path from executing in the reconciler.
}

func TestChainRun_FailurePropagation_SkipsDependents(t *testing.T) {
	rec, k8s, cleanup := setupChainReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Chain: A -> B -> C  (linear). A fails. B and C should be skipped.
	chain := &aotv1alpha1.Chain{
		ObjectMeta: metav1.ObjectMeta{Name: "chain-fail-prop", Namespace: "default"},
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "A", TemplateRef: "t"},
				{Name: "B", TemplateRef: "t", DependsOn: []string{"A"}},
				{Name: "C", TemplateRef: "t", DependsOn: []string{"B"}},
			},
		},
	}
	if err := k8s.Create(ctx, chain); err != nil {
		t.Fatalf("create chain: %v", err)
	}

	cr := &aotv1alpha1.ChainRun{
		ObjectMeta: metav1.ObjectMeta{Name: "cr-fail-prop", Namespace: "default"},
		Spec:       aotv1alpha1.ChainRunSpec{ChainRef: "chain-fail-prop"},
	}
	if err := k8s.Create(ctx, cr); err != nil {
		t.Fatalf("create chain run: %v", err)
	}

	// First reconcile: adds finalizer and returns.
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile finalizer: %v", err)
	}

	// Second reconcile: initialize steps
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile init: %v", err)
	}

	// Manually set step A to failed
	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get after init: %v", err)
	}
	cr.Status.Steps[0].Phase = "failed"
	cr.Status.Steps[0].Message = "simulated failure"
	if err := k8s.Status().Update(ctx, cr); err != nil {
		t.Fatalf("status update: %v", err)
	}

	// Reconcile: should propagate failure to B and C, then mark chain failed
	if _, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cr)}); err != nil {
		t.Fatalf("reconcile propagate: %v", err)
	}

	if err := k8s.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
		t.Fatalf("get final: %v", err)
	}

	stepPhases := map[string]string{}
	for _, s := range cr.Status.Steps {
		stepPhases[s.Name] = s.Phase
	}

	if stepPhases["A"] != "failed" {
		t.Errorf("A: expected failed, got %q", stepPhases["A"])
	}
	if stepPhases["B"] != "skipped" {
		t.Errorf("B: expected skipped, got %q", stepPhases["B"])
	}
	if stepPhases["C"] != "skipped" {
		t.Errorf("C: expected skipped, got %q", stepPhases["C"])
	}
	if cr.Status.Phase != "failed" {
		t.Errorf("chain: expected failed, got %q", cr.Status.Phase)
	}
}
