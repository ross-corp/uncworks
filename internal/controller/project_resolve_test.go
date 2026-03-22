package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func TestResolveProjectDefaults_InheritsRepos(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test-proj", Namespace: "aot"},
		Spec: aotv1alpha1.ProjectSpec{
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/org/repo", Branch: "main"},
			},
		},
	}
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			ProjectRef: "test-proj",
			Prompt:     "do stuff",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(project).Build()
	_, err := ResolveProjectDefaults(context.Background(), k8s, nil, run, "aot")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(run.Spec.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(run.Spec.Repos))
	}
	if run.Spec.Repos[0].URL != "https://github.com/org/repo" {
		t.Errorf("repo URL = %q", run.Spec.Repos[0].URL)
	}
}

func TestResolveProjectDefaults_InheritsModelAndTTL(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "proj", Namespace: "aot"},
		Spec: aotv1alpha1.ProjectSpec{
			Defaults: &aotv1alpha1.ProjectDefaults{
				ModelTier:          "qwen3:8b",
				ManageModelTier:    "deepseek-v3.1",
				ImplementModelTier: "qwen3-coder",
				TTLSeconds:         600,
				OrchestrationMode:  "spec-driven",
			},
		},
	}
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-2", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			ProjectRef: "proj",
			Prompt:     "test",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(project).Build()
	_, err := ResolveProjectDefaults(context.Background(), k8s, nil, run, "aot")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if run.Spec.ModelTier != "qwen3:8b" {
		t.Errorf("ModelTier = %q, want qwen3:8b", run.Spec.ModelTier)
	}
	if run.Spec.ManageModelTier != "deepseek-v3.1" {
		t.Errorf("ManageModelTier = %q, want deepseek-v3.1", run.Spec.ManageModelTier)
	}
	if run.Spec.ImplementModelTier != "qwen3-coder" {
		t.Errorf("ImplementModelTier = %q, want qwen3-coder", run.Spec.ImplementModelTier)
	}
	if run.Spec.TTLSeconds != 600 {
		t.Errorf("TTLSeconds = %d, want 600", run.Spec.TTLSeconds)
	}
	if run.Spec.OrchestrationMode != "spec-driven" {
		t.Errorf("OrchestrationMode = %q, want spec-driven", run.Spec.OrchestrationMode)
	}
}

func TestResolveProjectDefaults_RunOverridesProject(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "proj", Namespace: "aot"},
		Spec: aotv1alpha1.ProjectSpec{
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/org/default-repo", Branch: "main"},
			},
			Defaults: &aotv1alpha1.ProjectDefaults{
				ModelTier:  "default",
				TTLSeconds: 600,
			},
		},
	}
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-3", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			ProjectRef: "proj",
			Prompt:     "test",
			// These override the project defaults:
			Repos:     []aotv1alpha1.Repository{{URL: "https://github.com/org/custom-repo", Branch: "dev"}},
			ModelTier: "premium",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(project).Build()
	_, err := ResolveProjectDefaults(context.Background(), k8s, nil, run, "aot")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// Repos should NOT be overridden — run had explicit repos
	if len(run.Spec.Repos) != 1 || run.Spec.Repos[0].URL != "https://github.com/org/custom-repo" {
		t.Errorf("repos should not be overridden: %v", run.Spec.Repos)
	}
	// ModelTier should NOT be overridden — run had explicit model
	if run.Spec.ModelTier != "premium" {
		t.Errorf("ModelTier should not be overridden: %q", run.Spec.ModelTier)
	}
	// TTLSeconds SHOULD be inherited (run had 0)
	if run.Spec.TTLSeconds != 600 {
		t.Errorf("TTLSeconds should be inherited: %d", run.Spec.TTLSeconds)
	}
}

func TestResolveProjectDefaults_NoProjectRef(t *testing.T) {
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-4", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			Prompt: "standalone",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	url, err := ResolveProjectDefaults(context.Background(), k8s, nil, run, "aot")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if url != "" {
		t.Errorf("expected empty URL for standalone run, got %q", url)
	}
}

func TestResolveProjectDefaults_SetsProjectLabel(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "my-proj", Namespace: "aot"},
		Spec: aotv1alpha1.ProjectSpec{
			DisplayName: "My Project",
		},
	}
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-5", Namespace: "aot"},
		Spec: aotv1alpha1.AgentRunSpec{
			ProjectRef: "my-proj",
			Prompt:     "test",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(project).Build()
	_, err := ResolveProjectDefaults(context.Background(), k8s, nil, run, "aot")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if run.Spec.Project != "My Project" {
		t.Errorf("Project = %q, want 'My Project'", run.Spec.Project)
	}
}
