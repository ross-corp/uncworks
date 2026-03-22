package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func TestBuildWorkflowInput_BasicFields(t *testing.T) {
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ar-test", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/org/repo", Branch: "main"}},
			Prompt:     "do stuff",
			ModelTier:  "qwen3:8b",
			TTLSeconds: 600,
		},
	}

	input := BuildWorkflowInput(ar, "http://litellm:4000", "github-token")
	if input.AgentRunName != "ar-test" {
		t.Errorf("AgentRunName = %q", input.AgentRunName)
	}
	if input.Prompt != "do stuff" {
		t.Errorf("Prompt = %q", input.Prompt)
	}
	if input.ModelTier != "qwen3:8b" {
		t.Errorf("ModelTier = %q", input.ModelTier)
	}
	if input.TTLSeconds != 600 {
		t.Errorf("TTLSeconds = %d", input.TTLSeconds)
	}
	if input.LiteLLMBaseURL != "http://litellm:4000" {
		t.Errorf("LiteLLMBaseURL = %q", input.LiteLLMBaseURL)
	}
	if len(input.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(input.Repos))
	}
	if input.Repos[0].URL != "https://github.com/org/repo" {
		t.Errorf("repo URL = %q", input.Repos[0].URL)
	}
}

func TestBuildWorkflowInput_DualModels(t *testing.T) {
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ar-dual", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			Prompt:             "test",
			ModelTier:          "default",
			ManageModelTier:    "qwen3:8b",
			ImplementModelTier: "deepseek-v3.1",
		},
	}

	input := BuildWorkflowInput(ar, "", "")
	if input.ManageModelTier != "qwen3:8b" {
		t.Errorf("ManageModelTier = %q, want qwen3:8b", input.ManageModelTier)
	}
	if input.ImplementModelTier != "deepseek-v3.1" {
		t.Errorf("ImplementModelTier = %q, want deepseek-v3.1", input.ImplementModelTier)
	}
}

func TestBuildWorkflowInput_ProjectRef(t *testing.T) {
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ar-proj", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			Prompt:     "from project",
			ProjectRef: "my-project",
			SpecRef:    "add-auth",
		},
	}

	input := BuildWorkflowInput(ar, "", "")
	// ProjectRef and SpecRef don't flow into WorkflowInput directly
	// (they're resolved before BuildWorkflowInput is called)
	// But the resolved fields should be present
	if input.Prompt != "from project" {
		t.Errorf("Prompt = %q", input.Prompt)
	}
}

func TestBuildWorkflowInput_SpecDriven(t *testing.T) {
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ar-spec", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			Prompt:            "build auth",
			OrchestrationMode: "spec-driven",
			SpecContent:       "## Requirements\nAdd auth",
			AutoPush:          true,
			AutoPR:            true,
			PRBaseBranch:      "develop",
			Project:           "my-project",
			Feature:           "auth-system",
			Tags:              []string{"security", "backend"},
		},
	}

	input := BuildWorkflowInput(ar, "", "")
	if string(input.OrchestrationMode) != "spec-driven" {
		t.Errorf("OrchestrationMode = %q", input.OrchestrationMode)
	}
	if input.SpecContent != "## Requirements\nAdd auth" {
		t.Errorf("SpecContent = %q", input.SpecContent)
	}
	if !input.AutoPush {
		t.Error("AutoPush should be true")
	}
	if !input.AutoPR {
		t.Error("AutoPR should be true")
	}
	if input.PRBaseBranch != "develop" {
		t.Errorf("PRBaseBranch = %q", input.PRBaseBranch)
	}
	if input.Project != "my-project" {
		t.Errorf("Project = %q", input.Project)
	}
	if input.Feature != "auth-system" {
		t.Errorf("Feature = %q", input.Feature)
	}
	if len(input.Tags) != 2 {
		t.Errorf("Tags = %v", input.Tags)
	}
}
