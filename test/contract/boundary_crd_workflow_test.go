package contract

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/controller"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestBoundary_CRDToWorkflowInput_AllFields verifies that BuildWorkflowInput
// maps every CRD field to the corresponding WorkflowInput field.
func TestBoundary_CRDToWorkflowInput_AllFields(t *testing.T) {
	crd := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run-123",
			Namespace: "aot",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/org/repo.git", Branch: "main", Path: "my-repo"},
				{URL: "https://github.com/org/other.git", Branch: "develop"},
			},
			Prompt:            "Fix everything",
			DevboxConfig:      "/workspace/devbox.json",
			TTLSeconds:        7200,
			EnvVars:           map[string]string{"KEY": "value", "SECRET": "hidden"},
			Image:             "custom-image:v1",
			ModelTier:         "premium",
			SpecContent:       "# Spec\nDo stuff",
			WorkspaceName:     "ws-prod",
			OrchestrationMode: aotv1alpha1.OrchestrationModeSpecDriven,
			Orchestration: &aotv1alpha1.Orchestration{
				Tasks: []aotv1alpha1.OrchestrationTask{
					{Name: "init", Prompt: "Initialize", RepoURLs: []string{"https://github.com/org/repo.git"}},
					{Name: "impl", Prompt: "Implement"},
				},
			},
			ParentRunID:  "parent-42",
			SpecRunID:    "spec-77",
			DisplayName:  "Fix Everything Run",
			MaxBudget:    5.0,
			AutoPush:     true,
			AutoPR:       true,
			PRBaseBranch: "staging",
			Project:      "platform",
			Feature:      "auth-overhaul",
			SpecSource:   "editor",
			Tags:         []string{"backend", "security"},
			PipelineConfig: &aotv1alpha1.PipelineConfig{
				Plan: aotv1alpha1.StageConfig{
					Model:          "gpt-4",
					TimeoutSeconds: 300,
					MaxRetries:     2,
					OnFailure:      "fail",
				},
				Execute: aotv1alpha1.StageConfig{
					Model:          "claude-3",
					TimeoutSeconds: 900,
					MaxRetries:     5,
					OnFailure:      "retry",
				},
				Verify: aotv1alpha1.StageConfig{
					Model:          "gpt-4o",
					TimeoutSeconds: 180,
					MaxRetries:     1,
					OnFailure:      "skip",
				},
			},
		},
	}

	liteLLMURL := "http://litellm:4000"
	got := controller.BuildWorkflowInput(crd, liteLLMURL, "")

	// Scalar fields
	assertEqual(t, "AgentRunName", got.AgentRunName, "test-run-123")
	assertEqual(t, "Namespace", got.Namespace, "aot")
	assertEqual(t, "Prompt", got.Prompt, "Fix everything")
	assertEqual(t, "DevboxConfig", got.DevboxConfig, "/workspace/devbox.json")
	assertEqual(t, "TTLSeconds", got.TTLSeconds, int32(7200))
	assertEqual(t, "Image", got.Image, "custom-image:v1")
	assertEqual(t, "ModelTier", got.ModelTier, "premium")
	assertEqual(t, "LiteLLMBaseURL", got.LiteLLMBaseURL, liteLLMURL)
	assertEqual(t, "SpecContent", got.SpecContent, "# Spec\nDo stuff")
	assertEqual(t, "WorkspaceName", got.WorkspaceName, "ws-prod")
	assertEqual(t, "OrchestrationMode", got.OrchestrationMode, aottemporal.OrchestrationModeSpecDriven)
	assertEqual(t, "ParentRunID", got.ParentRunID, "parent-42")
	assertEqual(t, "SpecRunID", got.SpecRunID, "spec-77")
	assertEqual(t, "MaxBudget", got.MaxBudget, 5.0)
	assertEqual(t, "AutoPush", got.AutoPush, true)
	assertEqual(t, "AutoPR", got.AutoPR, true)
	assertEqual(t, "PRBaseBranch", got.PRBaseBranch, "staging")
	assertEqual(t, "Project", got.Project, "platform")
	assertEqual(t, "Feature", got.Feature, "auth-overhaul")
	assertEqual(t, "Backend", got.Backend, "Pod")
	assertEqual(t, "SpecSource", got.SpecSource, "editor")
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(got.Tags))
	}
	assertEqual(t, "Tags[0]", got.Tags[0], "backend")
	assertEqual(t, "Tags[1]", got.Tags[1], "security")

	// Repos
	if len(got.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(got.Repos))
	}
	assertEqual(t, "Repos[0].URL", got.Repos[0].URL, "https://github.com/org/repo.git")
	assertEqual(t, "Repos[0].Branch", got.Repos[0].Branch, "main")
	assertEqual(t, "Repos[0].Path", got.Repos[0].Path, "my-repo")
	assertEqual(t, "Repos[1].URL", got.Repos[1].URL, "https://github.com/org/other.git")
	assertEqual(t, "Repos[1].Branch", got.Repos[1].Branch, "develop")

	// EnvVars
	if len(got.EnvVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(got.EnvVars))
	}
	assertEqual(t, "EnvVars[KEY]", got.EnvVars["KEY"], "value")
	assertEqual(t, "EnvVars[SECRET]", got.EnvVars["SECRET"], "hidden")

	// Orchestration tasks
	if len(got.Orchestration) != 2 {
		t.Fatalf("expected 2 orchestration tasks, got %d", len(got.Orchestration))
	}
	assertEqual(t, "Orchestration[0].Name", got.Orchestration[0].Name, "init")
	assertEqual(t, "Orchestration[0].Prompt", got.Orchestration[0].Prompt, "Initialize")
	if len(got.Orchestration[0].RepoURLs) != 1 {
		t.Fatalf("expected 1 repo URL in task 0, got %d", len(got.Orchestration[0].RepoURLs))
	}
	assertEqual(t, "Orchestration[0].RepoURLs[0]", got.Orchestration[0].RepoURLs[0], "https://github.com/org/repo.git")
	assertEqual(t, "Orchestration[1].Name", got.Orchestration[1].Name, "impl")
	assertEqual(t, "Orchestration[1].Prompt", got.Orchestration[1].Prompt, "Implement")

	// Pipeline config
	if got.PipelineConfig == nil {
		t.Fatal("expected non-nil PipelineConfig")
	}
	assertWorkflowStageConfig(t, "Plan", got.PipelineConfig.Plan, aottemporal.StageConfigInput{
		Model: "gpt-4", TimeoutSeconds: 300, MaxRetries: 2, OnFailure: "fail",
	})
	assertWorkflowStageConfig(t, "Execute", got.PipelineConfig.Execute, aottemporal.StageConfigInput{
		Model: "claude-3", TimeoutSeconds: 900, MaxRetries: 5, OnFailure: "retry",
	})
	assertWorkflowStageConfig(t, "Verify", got.PipelineConfig.Verify, aottemporal.StageConfigInput{
		Model: "gpt-4o", TimeoutSeconds: 180, MaxRetries: 1, OnFailure: "skip",
	})
}

// TestBoundary_CRDToWorkflowInput_NilOptionals verifies that optional fields
// produce correct zero-value outputs when not set.
func TestBoundary_CRDToWorkflowInput_NilOptionals(t *testing.T) {
	crd := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal-run",
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Repos:   []aotv1alpha1.Repository{{URL: "https://github.com/org/repo.git"}},
			Prompt:  "Just do it",
		},
	}

	got := controller.BuildWorkflowInput(crd, "", "")

	assertEqual(t, "AgentRunName", got.AgentRunName, "minimal-run")
	assertEqual(t, "Prompt", got.Prompt, "Just do it")
	assertEqual(t, "AutoPush", got.AutoPush, false)
	assertEqual(t, "AutoPR", got.AutoPR, false)
	assertEqual(t, "PRBaseBranch", got.PRBaseBranch, "")
	assertEqual(t, "Backend", got.Backend, "Pod")
	assertEqual(t, "SpecSource", got.SpecSource, "")

	if got.PipelineConfig != nil {
		t.Error("expected nil PipelineConfig for CRD without pipeline config")
	}
	if len(got.Orchestration) != 0 {
		t.Errorf("expected 0 orchestration tasks, got %d", len(got.Orchestration))
	}
}

// TestBoundary_CRDWorkflow_BackendMapped verifies BuildWorkflowInput copies
// Backend for each BackendType variant.
func TestBoundary_CRDWorkflow_BackendMapped(t *testing.T) {
	tests := []struct {
		backend aotv1alpha1.BackendType
		want    string
	}{
		{aotv1alpha1.BackendPod, "Pod"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			crd := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: tt.backend,
					Repos:   []aotv1alpha1.Repository{{URL: "https://github.com/org/repo.git"}},
					Prompt:  "test",
				},
			}
			got := controller.BuildWorkflowInput(crd, "", "")
			assertEqual(t, "Backend", got.Backend, tt.want)
		})
	}
}

// TestBoundary_CRDWorkflow_SpecSourceMapped verifies BuildWorkflowInput copies
// SpecSource when present and returns empty string when absent.
func TestBoundary_CRDWorkflow_SpecSourceMapped(t *testing.T) {
	// With SpecSource set
	crd := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/org/repo.git"}},
			Prompt:     "test",
			SpecSource: "github:org/repo/spec.md",
		},
	}
	got := controller.BuildWorkflowInput(crd, "", "")
	assertEqual(t, "SpecSource (set)", got.SpecSource, "github:org/repo/spec.md")

	// Without SpecSource
	crd2 := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-2", Namespace: "default"},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Repos:   []aotv1alpha1.Repository{{URL: "https://github.com/org/repo.git"}},
			Prompt:  "test",
		},
	}
	got2 := controller.BuildWorkflowInput(crd2, "", "")
	assertEqual(t, "SpecSource (empty)", got2.SpecSource, "")
}

func assertWorkflowStageConfig(t *testing.T, name string, got, want aottemporal.StageConfigInput) {
	t.Helper()
	assertEqual(t, name+".Model", got.Model, want.Model)
	assertEqual(t, name+".TimeoutSeconds", got.TimeoutSeconds, want.TimeoutSeconds)
	assertEqual(t, name+".MaxRetries", got.MaxRetries, want.MaxRetries)
	assertEqual(t, name+".OnFailure", got.OnFailure, want.OnFailure)
}
