package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/internal/eventbus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "https with .git suffix",
			url:  "https://github.com/example/repo.git",
			want: "repo",
		},
		{
			name: "https without .git suffix",
			url:  "https://github.com/example/my-repo",
			want: "my-repo",
		},
		{
			name: "ssh URL",
			url:  "git@github.com:example/cool-project.git",
			want: "cool-project",
		},
		{
			name: "nested path",
			url:  "https://github.com/org/sub/nested-repo.git",
			want: "nested-repo",
		},
		{
			name: "plain repo name",
			url:  "repo-name",
			want: "repo-name",
		},
		{
			name: "trailing slash",
			url:  "https://github.com/example/repo/",
			want: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestCreateAgentRun_AutoSetsLabels(t *testing.T) {
	// Use a raw fake client so we can inspect labels directly on the CRD.
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")

	resp, err := svc.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/acme/web-app.git"}},
			Prompt:  "Fix the login bug",
			Project: "my-project",
			Feature: "login-fix",
			Tags:    []string{"bugfix"},
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	if resp.Msg.AgentRun.Id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Fetch the CRD to inspect labels.
	var crd aotv1alpha1.AgentRun
	if err := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      resp.Msg.AgentRun.Id,
	}, &crd); err != nil {
		t.Fatalf("failed to get CRD: %v", err)
	}

	// Verify auto-set labels
	if crd.Labels["aot.uncworks.io/project"] != "my-project" {
		t.Errorf("expected project label 'my-project', got %q", crd.Labels["aot.uncworks.io/project"])
	}
	if crd.Labels["aot.uncworks.io/feature"] != "login-fix" {
		t.Errorf("expected feature label 'login-fix', got %q", crd.Labels["aot.uncworks.io/feature"])
	}
	if crd.Labels["aot.uncworks.io/tags"] != "bugfix" {
		t.Errorf("expected tags label 'bugfix', got %q", crd.Labels["aot.uncworks.io/tags"])
	}
	if crd.Labels["aot.uncworks.io/repo"] != "web-app" {
		t.Errorf("expected repo label 'web-app', got %q", crd.Labels["aot.uncworks.io/repo"])
	}
}

func TestCreateAgentRun_AutoSetsRepoLabelFromURL(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")

	tests := []struct {
		name    string
		repoURL string
		want    string
	}{
		{"https with .git", "https://github.com/org/my-repo.git", "my-repo"},
		{"https without .git", "https://github.com/org/cool-app", "cool-app"},
		{"ssh url", "git@github.com:org/ssh-repo.git", "ssh-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
				Spec: &apiv1.AgentRunSpec{
					Backend: apiv1.Backend_BACKEND_POD,
					Repos:   []*apiv1.Repository{{Url: tt.repoURL}},
					Prompt:  "test",
				},
			}))
			if err != nil {
				t.Fatalf("CreateAgentRun: %v", err)
			}

			var crd aotv1alpha1.AgentRun
			if err := k8sClient.Get(context.Background(), client.ObjectKey{
				Namespace: "default",
				Name:      resp.Msg.AgentRun.Id,
			}, &crd); err != nil {
				t.Fatalf("failed to get CRD: %v", err)
			}

			if crd.Labels["aot.uncworks.io/repo"] != tt.want {
				t.Errorf("expected repo label %q, got %q", tt.want, crd.Labels["aot.uncworks.io/repo"])
			}
		})
	}
}

func TestClassifyHandler_MissingPrompt(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	h := NewClassifyRunHandler(k8sClient, "default", "http://fake-llm:4000")

	mux := http.NewServeMux()
	h.RegisterClassifyHandlers(mux)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/classify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClassifyHandler_InvalidJSON(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	h := NewClassifyRunHandler(k8sClient, "default", "http://fake-llm:4000")

	mux := http.NewServeMux()
	h.RegisterClassifyHandlers(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/classify", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClassifyHandler_WithMockLLM(t *testing.T) {
	// Set up a mock LiteLLM server that returns a valid classification.
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{"project": "web-platform", "feature": "login-fix", "featureIsNew": true, "tags": ["bugfix", "auth"]}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	// Seed some existing AgentRuns with labels.
	existingRun := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ar-existing",
			Namespace: "default",
			Labels: map[string]string{
				"aot.uncworks.io/project": "web-platform",
				"aot.uncworks.io/feature": "signup-flow",
			},
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Prompt:  "existing run",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(existingRun).
		Build()

	h := NewClassifyRunHandler(k8sClient, "default", mockLLM.URL)

	mux := http.NewServeMux()
	h.RegisterClassifyHandlers(mux)

	body := `{"prompt": "Fix the login page CSS", "repos": ["https://github.com/acme/web-app.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/classify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result classifyResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Project != "web-platform" {
		t.Errorf("expected project 'web-platform', got %q", result.Project)
	}
	if result.Feature != "login-fix" {
		t.Errorf("expected feature 'login-fix', got %q", result.Feature)
	}
	if !result.FeatureIsNew {
		t.Error("expected featureIsNew to be true")
	}
	if len(result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Tags))
	}
}

func TestExtractExistingLabels(t *testing.T) {
	runs := []aotv1alpha1.AgentRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ar-1",
				Namespace: "default",
				Labels: map[string]string{
					"aot.uncworks.io/project": "proj-a",
					"aot.uncworks.io/feature": "feat-1",
				},
			},
			Spec: aotv1alpha1.AgentRunSpec{Backend: aotv1alpha1.BackendPod, Prompt: "run 1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ar-2",
				Namespace: "default",
				Labels: map[string]string{
					"aot.uncworks.io/project": "proj-b",
					"aot.uncworks.io/feature": "feat-1", // duplicate feature
				},
			},
			Spec: aotv1alpha1.AgentRunSpec{Backend: aotv1alpha1.BackendPod, Prompt: "run 2"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ar-3",
				Namespace: "default",
				// No labels
			},
			Spec: aotv1alpha1.AgentRunSpec{Backend: aotv1alpha1.BackendPod, Prompt: "run 3"},
		},
	}

	var objs []aotv1alpha1.AgentRun
	objs = append(objs, runs...)

	builder := fake.NewClientBuilder().WithScheme(testScheme)
	for i := range objs {
		builder = builder.WithObjects(&objs[i])
	}
	k8sClient := builder.Build()

	h := &ClassifyRunHandler{
		K8sClient: k8sClient,
		Namespace: "default",
	}

	projects, features, err := h.extractExistingLabels(context.Background())
	if err != nil {
		t.Fatalf("extractExistingLabels: %v", err)
	}

	// Should have 2 distinct projects
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d: %v", len(projects), projects)
	}
	// Should have 1 distinct feature (feat-1 is duplicated)
	if len(features) != 1 {
		t.Errorf("expected 1 feature, got %d: %v", len(features), features)
	}
}

func TestBuildClassificationPrompt(t *testing.T) {
	prompt := buildClassificationPrompt(
		"Fix the login bug",
		[]string{"https://github.com/acme/web.git"},
		[]string{"proj-a", "proj-b"},
		[]string{"feat-1"},
	)

	if !strings.Contains(prompt, "Fix the login bug") {
		t.Error("prompt should contain the user prompt")
	}
	if !strings.Contains(prompt, "proj-a") {
		t.Error("prompt should contain existing projects")
	}
	if !strings.Contains(prompt, "feat-1") {
		t.Error("prompt should contain existing features")
	}
	if !strings.Contains(prompt, "https://github.com/acme/web.git") {
		t.Error("prompt should contain repos")
	}
}
