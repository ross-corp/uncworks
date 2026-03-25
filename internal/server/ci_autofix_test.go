package server

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func ciScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func makeCheckRunPayload(action, conclusion, branch, repo string) []byte {
	p := checkRunPayload{
		Action: action,
		CheckRun: checkRun{
			ID:         123,
			Name:       "CI",
			Conclusion: conclusion,
			HeadSHA:    "abc123def456",
			CheckSuite: checkSuite{
				ID:         456,
				HeadBranch: branch,
			},
		},
	}
	p.Repository.FullName = repo
	data, _ := json.Marshal(p)
	return data
}

func TestHandleCheckRunEvent_FailureOnAotBranch(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3)

	payload := makeCheckRunPayload("completed", "failure", "aot/ar-test", "org/repo")
	triggered, err := ci.HandleCheckRunEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !triggered {
		t.Error("expected fix to be triggered for aot/ branch failure")
	}
}

func TestHandleCheckRunEvent_SuccessIgnored(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3)

	payload := makeCheckRunPayload("completed", "success", "aot/ar-test", "org/repo")
	triggered, err := ci.HandleCheckRunEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("should not trigger on success")
	}
}

func TestHandleCheckRunEvent_NonAotBranchIgnored(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3)

	payload := makeCheckRunPayload("completed", "failure", "main", "org/repo")
	triggered, err := ci.HandleCheckRunEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("should not trigger on non-aot branch")
	}
}

func TestHandleCheckRunEvent_PendingIgnored(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3)

	payload := makeCheckRunPayload("created", "failure", "aot/ar-test", "org/repo")
	triggered, err := ci.HandleCheckRunEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("should not trigger on non-completed action")
	}
}

func TestCondenseCIErrors_FilterErrorLines(t *testing.T) {
	raw := `Step 1: Installing dependencies
npm install
Step 2: Running tests
FAIL src/app.test.ts
  Error: Expected true to be false
Step 3: Build
Build successful`

	result := condenseCIErrors(raw)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if len(result) == 0 {
		t.Error("result should contain error lines")
	}
	// Should contain the error line
	if !contains(result, "FAIL") && !contains(result, "Error:") {
		t.Errorf("result should contain error indicators, got: %s", result[:min(200, len(result))])
	}
}

func TestCondenseCIErrors_TruncateLongOutput(t *testing.T) {
	// Generate a large error output
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "Error: something went wrong on line "+string(rune('0'+i%10)))
	}
	raw := ""
	for _, l := range lines {
		raw += l + "\n"
	}

	result := condenseCIErrors(raw)
	if len(result) > 9000 { // 8000 + truncation marker
		t.Errorf("result too long: %d chars (max ~8000)", len(result))
	}
}

func TestGetFixAttemptCount(t *testing.T) {
	runs := []aotv1alpha1.AgentRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-fix1", Namespace: "default",
				Annotations: map[string]string{
					"aot.uncworks.io/pr-branch": "aot/ar-original",
				},
			},
			Spec: aotv1alpha1.AgentRunSpec{
				SpecSource: "ci-autofix:org/repo#abc123",
				Prompt:     "fix CI",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-fix2", Namespace: "default",
				Annotations: map[string]string{
					"aot.uncworks.io/pr-branch": "aot/ar-original",
				},
			},
			Spec: aotv1alpha1.AgentRunSpec{
				SpecSource: "ci-autofix:org/repo#def456",
				Prompt:     "fix CI again",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-other", Namespace: "default",
				Annotations: map[string]string{
					"aot.uncworks.io/pr-branch": "aot/ar-different",
				},
			},
			Spec: aotv1alpha1.AgentRunSpec{
				SpecSource: "ci-autofix:org/repo#ghi789",
				Prompt:     "fix different branch",
			},
		},
	}

	objs := make([]runtime.Object, len(runs))
	for i := range runs {
		objs[i] = &runs[i]
	}
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).WithRuntimeObjects(objs...).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3)

	count, err := ci.getFixAttemptCount(context.Background(), "aot/ar-original")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 fix attempts for aot/ar-original, got %d", count)
	}

	count, err = ci.getFixAttemptCount(context.Background(), "aot/ar-different")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 fix attempt for aot/ar-different, got %d", count)
	}
}

func TestCircuitBreaker_MaxRetriesReached(t *testing.T) {
	runs := []aotv1alpha1.AgentRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-fix1", Namespace: "default",
				Annotations: map[string]string{"aot.uncworks.io/pr-branch": "aot/ar-maxed"},
			},
			Spec: aotv1alpha1.AgentRunSpec{SpecSource: "ci-autofix:org/repo#1", Prompt: "fix"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-fix2", Namespace: "default",
				Annotations: map[string]string{"aot.uncworks.io/pr-branch": "aot/ar-maxed"},
			},
			Spec: aotv1alpha1.AgentRunSpec{SpecSource: "ci-autofix:org/repo#2", Prompt: "fix"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ar-fix3", Namespace: "default",
				Annotations: map[string]string{"aot.uncworks.io/pr-branch": "aot/ar-maxed"},
			},
			Spec: aotv1alpha1.AgentRunSpec{SpecSource: "ci-autofix:org/repo#3", Prompt: "fix"},
		},
	}

	objs := make([]runtime.Object, len(runs))
	for i := range runs {
		objs[i] = &runs[i]
	}
	k8s := fake.NewClientBuilder().WithScheme(ciScheme()).WithRuntimeObjects(objs...).Build()
	ci := NewCIAutofix(context.Background(), k8s, "default", nil, 3) // max 3 retries

	payload := makeCheckRunPayload("completed", "failure", "aot/ar-maxed", "org/repo")
	triggered, err := ci.HandleCheckRunEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered {
		t.Error("should NOT trigger when max retries reached")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
