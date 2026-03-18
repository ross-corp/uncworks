package temporal

import (
	"testing"
)

func TestParseDecompositionPlan_ValidJSON(t *testing.T) {
	output := `Here is the decomposition plan:
{
  "tasks": [
    {"name": "fix-auth", "prompt": "Fix the auth module", "repos": ["github.com/org/api"]},
    {"name": "update-tests", "prompt": "Update all tests"}
  ],
  "integration_prompt": "Review and integrate all changes"
}`

	plan := parseDecompositionPlan(output)
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].Name != "fix-auth" {
		t.Fatalf("expected 'fix-auth', got %q", plan.Tasks[0].Name)
	}
	if len(plan.Tasks[0].Repos) != 1 || plan.Tasks[0].Repos[0] != "github.com/org/api" {
		t.Fatalf("expected repos [github.com/org/api], got %v", plan.Tasks[0].Repos)
	}
	if plan.Tasks[1].Name != "update-tests" {
		t.Fatalf("expected 'update-tests', got %q", plan.Tasks[1].Name)
	}
	if plan.IntegrationPrompt != "Review and integrate all changes" {
		t.Fatalf("unexpected integration prompt: %q", plan.IntegrationPrompt)
	}
}

func TestParseDecompositionPlan_Fallback(t *testing.T) {
	// Non-JSON output should return nil (fallback to single run).
	output := "I think this is a simple task, no decomposition needed."
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for non-JSON output")
	}
}

func TestParseDecompositionPlan_EmptyTasks(t *testing.T) {
	// Empty tasks means "simple enough for one agent" — fallback.
	output := `{"tasks": [], "integration_prompt": ""}`
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for empty tasks")
	}
}

func TestParseDecompositionPlan_TruncateTo7(t *testing.T) {
	output := `{
  "tasks": [
    {"name": "t1", "prompt": "p1"},
    {"name": "t2", "prompt": "p2"},
    {"name": "t3", "prompt": "p3"},
    {"name": "t4", "prompt": "p4"},
    {"name": "t5", "prompt": "p5"},
    {"name": "t6", "prompt": "p6"},
    {"name": "t7", "prompt": "p7"},
    {"name": "t8", "prompt": "p8"},
    {"name": "t9", "prompt": "p9"},
    {"name": "t10", "prompt": "p10"}
  ],
  "integration_prompt": "merge all"
}`

	plan := parseDecompositionPlan(output)
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(plan.Tasks) != 7 {
		t.Fatalf("expected 7 tasks (truncated from 10), got %d", len(plan.Tasks))
	}
	// Verify first and last tasks are correct after truncation
	if plan.Tasks[0].Name != "t1" {
		t.Fatalf("first task should be t1, got %q", plan.Tasks[0].Name)
	}
	if plan.Tasks[6].Name != "t7" {
		t.Fatalf("last task should be t7, got %q", plan.Tasks[6].Name)
	}
}

func TestParseDecompositionPlan_JSONWithSurroundingText(t *testing.T) {
	// Ensure parser extracts JSON even with surrounding prose.
	output := `Sure! Here's my analysis:

{"tasks": [{"name": "single-task", "prompt": "Do the thing"}], "integration_prompt": ""}

Let me know if you have questions.`

	plan := parseDecompositionPlan(output)
	if plan == nil {
		t.Fatal("expected non-nil plan when JSON is embedded in text")
	}
	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].Name != "single-task" {
		t.Fatalf("expected 'single-task', got %q", plan.Tasks[0].Name)
	}
}

func TestParseDecompositionPlan_MalformedJSON(t *testing.T) {
	output := `{"tasks": [{"name": "broken`
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for malformed JSON")
	}
}

func TestParseDecompositionPlan_NoBraces(t *testing.T) {
	output := "no json at all"
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for output without braces")
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/my-repo.git", "my-repo"},
		{"https://github.com/org/my-repo", "my-repo"},
		{"git@github.com:org/my-repo.git", "my-repo"},
		{"https://github.com/org/repo-name.git", "repo-name"},
		{"my-repo.git", "my-repo"},
		{"my-repo", "my-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestModelIDFromTier(t *testing.T) {
	tests := []struct {
		tier string
		want string
	}{
		{"", "litellm/default"},
		{"default", "litellm/default"},
		{"default-cloud", "litellm/default-cloud"},
		{"premium", "litellm/premium"},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := modelIDFromTier(tt.tier)
			if got != tt.want {
				t.Errorf("modelIDFromTier(%q) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}
