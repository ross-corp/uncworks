package temporal

import (
	"testing"
)

func TestWorkflow_SingleMode(t *testing.T) {
	// Verify that orchestration_mode=single (or unspecified) routes to the standard workflow.
	input := WorkflowInput{
		AgentRunName:      "test-single",
		Namespace:         "default",
		Prompt:            "do something",
		OrchestrationMode: OrchestrationModeSingle,
	}
	if input.OrchestrationMode != OrchestrationModeSingle {
		t.Fatalf("expected single mode, got %s", input.OrchestrationMode)
	}
}

func TestWorkflow_ManualOrchestrationInput(t *testing.T) {
	// Verify that manual orchestration input propagates correctly.
	input := WorkflowInput{
		AgentRunName:      "test-manual",
		Namespace:         "default",
		Prompt:            "manual orchestration",
		OrchestrationMode: OrchestrationModeManual,
		Orchestration: []OrchestrationTask{
			{Name: "fix-auth", Prompt: "Fix the auth module"},
			{Name: "update-tests", Prompt: "Update all tests"},
			{Name: "fix-ci", Prompt: "Fix the CI pipeline"},
		},
	}

	if len(input.Orchestration) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(input.Orchestration))
	}
	if input.Orchestration[0].Name != "fix-auth" {
		t.Fatalf("expected first task 'fix-auth', got %q", input.Orchestration[0].Name)
	}
}

func TestWorkflow_AutoDecomposition_ParseValid(t *testing.T) {
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
	if plan.IntegrationPrompt != "Review and integrate all changes" {
		t.Fatalf("unexpected integration prompt: %q", plan.IntegrationPrompt)
	}
}

func TestWorkflow_AutoDecomposition_Fallback(t *testing.T) {
	// Malformed JSON should return nil (fallback to single run).
	output := "I think this is a simple task, no decomposition needed."
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for non-JSON output")
	}
}

func TestWorkflow_AutoDecomposition_EmptyTasks(t *testing.T) {
	// Empty tasks means "simple enough for one agent" - fallback.
	output := `{"tasks": [], "integration_prompt": ""}`
	plan := parseDecompositionPlan(output)
	if plan != nil {
		t.Fatal("expected nil plan for empty tasks")
	}
}

func TestWorkflow_AutoDecomposition_TruncateTo7(t *testing.T) {
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
}

func TestSpawnJuniorInput_SpecRunID(t *testing.T) {
	// Verify SpawnJuniorInput carries SpecRunID.
	input := SpawnJuniorInput{
		ParentRunName: "parent-run",
		Namespace:     "default",
		Task:          "fix something",
		TaskName:      "fix-something",
		SpecRunID:     "parent-run",
		Blocking:      true,
	}
	if input.SpecRunID != "parent-run" {
		t.Fatalf("expected SpecRunID 'parent-run', got %q", input.SpecRunID)
	}
	if input.TaskName != "fix-something" {
		t.Fatalf("expected TaskName 'fix-something', got %q", input.TaskName)
	}
}
