package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/testutil"
)

func setupScheduleReconciler(t *testing.T) (*ScheduleReconciler, client.Client, func()) {
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

	reconciler := &ScheduleReconciler{Client: k8sClient, Scheme: scheme.Scheme}
	return reconciler, k8sClient, func() { _ = testEnv.Stop() }
}

func TestSchedule_Suspended_NoOp(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-suspended", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:        "0 0 * * *",
			Suspend:     true,
			ChainRef:    "my-chain",
			TemplateRef: "",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	result, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	// Suspended schedules should return immediately without requeue
	if result.RequeueAfter != 0 {
		t.Errorf("suspended schedule should not requeue, got %v", result.RequeueAfter)
	}
}

func TestSchedule_InvalidCron_NoRequeue(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-bad-cron", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:     "not a cron expression",
			ChainRef: "my-chain",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	result, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	// Invalid cron should not crash, just skip
	if result.RequeueAfter != 0 {
		t.Errorf("invalid cron should not requeue, got %v", result.RequeueAfter)
	}
}

func TestSchedule_NotDue_Requeues(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Schedule for far future (minute 59 of every hour — likely not right now)
	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-future", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:     "59 23 31 12 *", // Dec 31 at 23:59
			ChainRef: "my-chain",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Set last scheduled to now so nothing is missed
	now := metav1.Now()
	sched.Status.LastScheduledTime = &now
	if err := k8s.Status().Update(ctx, sched); err != nil {
		t.Fatalf("status update: %v", err)
	}

	result, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter <= 0 {
		t.Error("expected requeue for future schedule")
	}
}

func TestSchedule_Fires_ChainRef(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Schedule that fires every minute — set last scheduled to 2 minutes ago so it's due
	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-fire-chain", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:     "* * * * *", // every minute
			ChainRef: "weekly-review",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Set last scheduled to 2 min ago to ensure it's due
	twoMinAgo := metav1.NewTime(time.Now().Add(-2 * time.Minute))
	sched.Status.LastScheduledTime = &twoMinAgo
	if err := k8s.Status().Update(ctx, sched); err != nil {
		t.Fatalf("status update: %v", err)
	}

	result, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Should requeue for next tick
	if result.RequeueAfter <= 0 {
		t.Error("expected requeue after firing")
	}

	// Verify ChainRun was created
	if err := k8s.Get(ctx, client.ObjectKeyFromObject(sched), sched); err != nil {
		t.Fatalf("get sched: %v", err)
	}
	if sched.Status.LastRunID == "" {
		t.Error("expected lastRunID to be set")
	}
	if sched.Status.LastResult != "running" {
		t.Errorf("expected lastResult=running, got %q", sched.Status.LastResult)
	}
	if len(sched.Status.Active) == 0 {
		t.Error("expected active list to have entry")
	}

	// Verify the ChainRun exists in K8s
	var cr aotv1alpha1.ChainRun
	if err := k8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: sched.Status.LastRunID}, &cr); err != nil {
		t.Fatalf("chain run not found: %v", err)
	}
	if cr.Spec.ChainRef != "weekly-review" {
		t.Errorf("expected chainRef=weekly-review, got %q", cr.Spec.ChainRef)
	}
	if cr.Spec.TriggeredBy != "schedule:sched-fire-chain" {
		t.Errorf("expected triggeredBy=schedule:sched-fire-chain, got %q", cr.Spec.TriggeredBy)
	}
}

func TestSchedule_Fires_TemplateRef(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	// Create the template that the schedule references
	tmpl := &aotv1alpha1.RunTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily-review", Namespace: "default"},
		Spec: aotv1alpha1.RunTemplateSpec{
			Prompt:    "Review code quality",
			ModelTier: "deepseek-v3.1",
			Repos:     []aotv1alpha1.Repository{{URL: "https://github.com/test/repo", Branch: "main"}},
		},
	}
	if err := k8s.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}

	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-fire-tmpl", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:        "* * * * *",
			TemplateRef: "daily-review",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	twoMinAgo := metav1.NewTime(time.Now().Add(-2 * time.Minute))
	sched.Status.LastScheduledTime = &twoMinAgo
	if err := k8s.Status().Update(ctx, sched); err != nil {
		t.Fatalf("status update: %v", err)
	}

	_, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if err := k8s.Get(ctx, client.ObjectKeyFromObject(sched), sched); err != nil {
		t.Fatalf("get: %v", err)
	}
	if sched.Status.LastRunID == "" {
		t.Fatal("expected lastRunID to be set")
	}

	// Verify AgentRun was created from template
	var run aotv1alpha1.AgentRun
	if err := k8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: sched.Status.LastRunID}, &run); err != nil {
		t.Fatalf("agent run not found: %v", err)
	}
	if run.Spec.Prompt != "Review code quality" {
		t.Errorf("expected prompt from template, got %q", run.Spec.Prompt)
	}
	if run.Labels["aot.uncworks.io/schedule"] != "sched-fire-tmpl" {
		t.Error("missing schedule label")
	}
	if run.Labels["aot.uncworks.io/template"] != "daily-review" {
		t.Error("missing template label")
	}
}

func TestSchedule_ForbidConcurrency(t *testing.T) {
	rec, k8s, cleanup := setupScheduleReconciler(t)
	defer cleanup()
	ctx := context.Background()

	sched := &aotv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched-forbid", Namespace: "default"},
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:              "* * * * *",
			ConcurrencyPolicy: "Forbid",
			ChainRef:          "my-chain",
		},
	}
	if err := k8s.Create(ctx, sched); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Simulate an active run
	twoMinAgo := metav1.NewTime(time.Now().Add(-2 * time.Minute))
	sched.Status.LastScheduledTime = &twoMinAgo
	sched.Status.Active = []string{"cr-existing"}
	if err := k8s.Status().Update(ctx, sched); err != nil {
		t.Fatalf("status update: %v", err)
	}

	_, err := rec.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sched)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Verify no new ChainRun was created — active list should still be 1
	if err := k8s.Get(ctx, client.ObjectKeyFromObject(sched), sched); err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(sched.Status.Active) != 1 {
		t.Errorf("Forbid policy: expected 1 active run, got %d", len(sched.Status.Active))
	}
}

func TestSchedule_NotFound_Ignored(t *testing.T) {
	rec, _, cleanup := setupScheduleReconciler(t)
	defer cleanup()

	result, err := rec.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: "default", Name: "ghost"},
	})
	if err != nil {
		t.Fatalf("expected no error for not-found, got: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Error("should not requeue for deleted schedule")
	}
}
