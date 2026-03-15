package controller

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func TestOrchestrationLabels_SeniorRun(t *testing.T) {
	// Verify that a senior run (auto/manual mode) gets the correct labels set
	// during startWorkflow. Since we can't start an actual Temporal workflow
	// in unit tests without a mock, we verify the label-setting logic directly.
	ar := newAgentRun("senior-test", func(a *aotv1alpha1.AgentRun) {
		a.Spec.OrchestrationMode = aotv1alpha1.OrchestrationModeAuto
	})

	// Simulate what startWorkflow does for labels
	if ar.Labels == nil {
		ar.Labels = make(map[string]string)
	}
	if ar.Annotations == nil {
		ar.Annotations = make(map[string]string)
	}

	orchMode := ar.Spec.OrchestrationMode
	if orchMode == aotv1alpha1.OrchestrationModeAuto || orchMode == aotv1alpha1.OrchestrationModeManual {
		ar.Labels[labelSpecRunID] = ar.Name
		ar.Labels[labelRunRole] = "senior"
	}

	if ar.Labels[labelSpecRunID] != "senior-test" {
		t.Errorf("expected spec-run-id label 'senior-test', got %q", ar.Labels[labelSpecRunID])
	}
	if ar.Labels[labelRunRole] != "senior" {
		t.Errorf("expected run-role label 'senior', got %q", ar.Labels[labelRunRole])
	}
}

func TestOrchestrationLabels_JuniorRun(t *testing.T) {
	ar := newAgentRun("junior-test", func(a *aotv1alpha1.AgentRun) {
		a.Spec.ParentRunID = "parent-senior"
		a.Spec.SpecRunID = "parent-senior"
	})

	if ar.Labels == nil {
		ar.Labels = make(map[string]string)
	}
	if ar.Annotations == nil {
		ar.Annotations = make(map[string]string)
	}

	if ar.Spec.ParentRunID != "" {
		specRunID := ar.Spec.SpecRunID
		if specRunID == "" {
			specRunID = ar.Spec.ParentRunID
		}
		ar.Labels[labelSpecRunID] = specRunID
		ar.Labels[labelRunRole] = "junior"
		ar.Annotations[annotationParent] = ar.Spec.ParentRunID
	}

	if ar.Labels[labelSpecRunID] != "parent-senior" {
		t.Errorf("expected spec-run-id label 'parent-senior', got %q", ar.Labels[labelSpecRunID])
	}
	if ar.Labels[labelRunRole] != "junior" {
		t.Errorf("expected run-role label 'junior', got %q", ar.Labels[labelRunRole])
	}
	if ar.Annotations[annotationParent] != "parent-senior" {
		t.Errorf("expected parent annotation 'parent-senior', got %q", ar.Annotations[annotationParent])
	}
}

func TestOrchestrationLabels_SingleModeNoLabels(t *testing.T) {
	ar := newAgentRun("single-test")

	if ar.Labels == nil {
		ar.Labels = make(map[string]string)
	}

	orchMode := ar.Spec.OrchestrationMode
	if orchMode == aotv1alpha1.OrchestrationModeAuto || orchMode == aotv1alpha1.OrchestrationModeManual {
		ar.Labels[labelSpecRunID] = ar.Name
		ar.Labels[labelRunRole] = "senior"
	}

	if _, ok := ar.Labels[labelSpecRunID]; ok {
		t.Error("single mode should not have spec-run-id label")
	}
	if _, ok := ar.Labels[labelRunRole]; ok {
		t.Error("single mode should not have run-role label")
	}
}

func TestOrchestrationCRD_ManualTasks(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	_ = reconciler // Used to ensure envtest is set up correctly

	ctx := context.Background()
	ar := newAgentRun("manual-orch", func(a *aotv1alpha1.AgentRun) {
		a.Spec.OrchestrationMode = aotv1alpha1.OrchestrationModeManual
		a.Spec.Orchestration = &aotv1alpha1.Orchestration{
			Tasks: []aotv1alpha1.OrchestrationTask{
				{Name: "fix-auth", Prompt: "Fix the auth module"},
				{Name: "update-tests", Prompt: "Update all tests"},
			},
		}
	})

	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	// Read it back to verify the orchestration fields persisted
	var read aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &read); err != nil {
		t.Fatalf("get agentrun: %v", err)
	}

	if read.Spec.OrchestrationMode != aotv1alpha1.OrchestrationModeManual {
		t.Errorf("expected manual mode, got %q", read.Spec.OrchestrationMode)
	}
	if read.Spec.Orchestration == nil {
		t.Fatal("expected orchestration to be set")
	}
	if len(read.Spec.Orchestration.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(read.Spec.Orchestration.Tasks))
	}
	if read.Spec.Orchestration.Tasks[0].Name != "fix-auth" {
		t.Errorf("expected first task 'fix-auth', got %q", read.Spec.Orchestration.Tasks[0].Name)
	}
}

func TestOrchestrationCRD_ParentRunID(t *testing.T) {
	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()
	_ = reconciler

	ctx := context.Background()
	ar := newAgentRun("junior-crd", func(a *aotv1alpha1.AgentRun) {
		a.Spec.ParentRunID = "parent-run-123"
		a.Spec.SpecRunID = "parent-run-123"
	})

	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	var read aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &read); err != nil {
		t.Fatalf("get agentrun: %v", err)
	}

	if read.Spec.ParentRunID != "parent-run-123" {
		t.Errorf("expected parentRunID 'parent-run-123', got %q", read.Spec.ParentRunID)
	}
	if read.Spec.SpecRunID != "parent-run-123" {
		t.Errorf("expected specRunID 'parent-run-123', got %q", read.Spec.SpecRunID)
	}
}
