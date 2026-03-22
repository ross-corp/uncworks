package controller

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// createDeploymentAndPVC creates a Deployment and PVC that mimic what the
// Temporal activity creates for an agent run.
func createDeploymentAndPVC(t *testing.T, ctx context.Context, k8sClient client.Client, name, namespace string) {
	t.Helper()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agentrun-" + name,
			Namespace: namespace,
			Labels: map[string]string{
				"aot.uncworks.io/agentrun": name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"aot.uncworks.io/agentrun": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"aot.uncworks.io/agentrun": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "agent", Image: "busybox"},
					},
				},
			},
		},
	}
	if err := k8sClient.Create(ctx, dep); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aot-ws-" + name,
			Namespace: namespace,
			Labels: map[string]string{
				"aot.uncworks.io/agentrun": name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	if err := k8sClient.Create(ctx, pvc); err != nil {
		t.Fatalf("create pvc: %v", err)
	}
}

func TestCleanupExpiredRuns_DeletesExpiredResources(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	reconciler.RetentionDays = 7
	ctx := context.Background()

	name := "expired-run"
	completedAt := metav1.NewTime(time.Now().Add(-8 * 24 * time.Hour)) // 8 days ago

	// Create the AgentRun CRD in terminal state with old completedAt
	ar := newAgentRun(name)
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	// Update status to terminal with old completedAt
	ar.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
	ar.Status.CompletedAt = &completedAt
	ar.Status.DeploymentName = "agentrun-" + name
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Create the associated Deployment and PVC
	createDeploymentAndPVC(t, ctx, k8sClient, name, "default")

	// Run cleanup
	if err := reconciler.cleanupExpiredRuns(ctx); err != nil {
		t.Fatalf("cleanupExpiredRuns: %v", err)
	}

	// Verify Deployment is deleted
	dep := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: "agentrun-" + name, Namespace: "default"}, dep)
	if !errors.IsNotFound(err) {
		t.Errorf("expected Deployment to be deleted, got err=%v", err)
	}

	// Verify PVC has been marked for deletion.
	// In envtest there is no PV controller to finalize PVCs, so they get a
	// DeletionTimestamp but remain in the API until finalizers are cleared.
	pvc := &corev1.PersistentVolumeClaim{}
	err = k8sClient.Get(ctx, client.ObjectKey{Name: "aot-ws-" + name, Namespace: "default"}, pvc)
	if err != nil {
		if !errors.IsNotFound(err) {
			t.Errorf("unexpected error fetching PVC: %v", err)
		}
		// NotFound is also acceptable — means it was fully deleted
	} else if pvc.DeletionTimestamp.IsZero() {
		t.Error("expected PVC to be marked for deletion (DeletionTimestamp set)")
	}

	// Verify archived annotation is set
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get agentrun after cleanup: %v", err)
	}
	if updated.Annotations[annotationArchived] != "true" {
		t.Errorf("expected archived annotation, got %v", updated.Annotations)
	}

	// Verify CRD still exists
	if updated.Name != name {
		t.Errorf("CRD should still exist, got name=%s", updated.Name)
	}
}

func TestCleanupExpiredRuns_SkipsNonExpiredRuns(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	reconciler.RetentionDays = 7
	ctx := context.Background()

	name := "recent-run"
	completedAt := metav1.NewTime(time.Now().Add(-2 * 24 * time.Hour)) // 2 days ago

	ar := newAgentRun(name)
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	ar.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
	ar.Status.CompletedAt = &completedAt
	ar.Status.DeploymentName = "agentrun-" + name
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	createDeploymentAndPVC(t, ctx, k8sClient, name, "default")

	// Run cleanup
	if err := reconciler.cleanupExpiredRuns(ctx); err != nil {
		t.Fatalf("cleanupExpiredRuns: %v", err)
	}

	// Verify Deployment still exists
	dep := &appsv1.Deployment{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "agentrun-" + name, Namespace: "default"}, dep); err != nil {
		t.Errorf("expected Deployment to still exist, got err=%v", err)
	}

	// Verify PVC still exists
	pvc := &corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "aot-ws-" + name, Namespace: "default"}, pvc); err != nil {
		t.Errorf("expected PVC to still exist, got err=%v", err)
	}

	// Verify no archived annotation
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Annotations != nil && updated.Annotations[annotationArchived] == "true" {
		t.Error("non-expired run should not be archived")
	}
}

func TestCleanupExpiredRuns_SkipsAlreadyArchivedRuns(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	reconciler.RetentionDays = 7
	ctx := context.Background()

	name := "already-archived"
	completedAt := metav1.NewTime(time.Now().Add(-30 * 24 * time.Hour)) // 30 days ago

	ar := newAgentRun(name, func(a *aotv1alpha1.AgentRun) {
		a.Annotations = map[string]string{
			annotationArchived: "true",
		}
	})
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	ar.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
	ar.Status.CompletedAt = &completedAt
	ar.Status.DeploymentName = "agentrun-" + name
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Do NOT create Deployment/PVC — they were already cleaned up.
	// The cleanup should skip this run without error.

	if err := reconciler.cleanupExpiredRuns(ctx); err != nil {
		t.Fatalf("cleanupExpiredRuns: %v", err)
	}

	// Verify no error occurred and annotation is unchanged
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Annotations[annotationArchived] != "true" {
		t.Error("archived annotation should remain")
	}
}

func TestCleanupExpiredRuns_ToleratesMissingResources(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	reconciler.RetentionDays = 7
	ctx := context.Background()

	name := "missing-resources"
	completedAt := metav1.NewTime(time.Now().Add(-10 * 24 * time.Hour)) // 10 days ago

	ar := newAgentRun(name)
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	ar.Status.Phase = aotv1alpha1.AgentRunPhaseCancelled
	ar.Status.CompletedAt = &completedAt
	ar.Status.DeploymentName = "agentrun-" + name
	if err := k8sClient.Status().Update(ctx, ar); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Do NOT create Deployment or PVC — simulate already-deleted resources.

	// Should not error — NotFound is tolerated
	if err := reconciler.cleanupExpiredRuns(ctx); err != nil {
		t.Fatalf("cleanupExpiredRuns: %v", err)
	}

	// Verify archived annotation is still set
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Annotations[annotationArchived] != "true" {
		t.Errorf("expected archived annotation even with missing resources, got %v", updated.Annotations)
	}
}

func TestRetentionPeriod_DefaultAndCustom(t *testing.T) {
	r := &AgentRunReconciler{RetentionDays: 0}
	if got := r.retentionPeriod(); got != 7*24*time.Hour {
		t.Errorf("default retention: expected 7d, got %v", got)
	}

	r.RetentionDays = 14
	if got := r.retentionPeriod(); got != 14*24*time.Hour {
		t.Errorf("custom retention: expected 14d, got %v", got)
	}
}
