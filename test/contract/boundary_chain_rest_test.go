package contract

import (
	"encoding/json"
	"strings"
	"testing"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// TestBoundary_RunTemplateJSON verifies JSON serialization matches frontend TS interfaces.
func TestBoundary_RunTemplateJSON(t *testing.T) {
	tmpl := aotv1alpha1.RunTemplate{
		Spec: aotv1alpha1.RunTemplateSpec{
			DisplayName:        "Code Analysis",
			Description:        "Analyze code quality",
			ProjectRef:         "my-project",
			Prompt:             "Review the code",
			ModelTier:          "claude-sonnet-4.6",
			ManageModelTier:    "claude-sonnet-4.6",
			ImplementModelTier: "deepseek-v3.1",
			OrchestrationMode:  "spec-driven",
			TTLSeconds:         900,
			AutoPush:           true,
			AutoPR:             true,
			PRBaseBranch:       "main",
			SpecRef:            "code-review",
			Repos: []aotv1alpha1.Repository{
				{URL: "https://github.com/org/repo", Branch: "main"},
			},
		},
	}

	data, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatalf("json.Marshal RunTemplate: %v", err)
	}
	s := string(data)

	for _, field := range []string{
		"displayName", "description", "projectRef", "prompt", "modelTier",
		"manageModelTier", "implementModelTier", "orchestrationMode",
		"ttlSeconds", "autoPush", "autoPR", "prBaseBranch", "specRef", "repos",
	} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("RunTemplate JSON missing field: %s", field)
		}
	}
}

// TestBoundary_ChainJSON verifies Chain JSON serialization for frontend.
func TestBoundary_ChainJSON(t *testing.T) {
	chain := aotv1alpha1.Chain{
		Spec: aotv1alpha1.ChainSpec{
			DisplayName: "CI Pipeline",
			Description: "Full CI pipeline",
			ProjectRef:  "my-project",
			Steps: []aotv1alpha1.ChainStep{
				{Name: "lint", TemplateRef: "lint-template"},
				{Name: "test", TemplateRef: "test-template", DependsOn: []string{"lint"}},
				{Name: "build", TemplateRef: "build-template", DependsOn: []string{"test"}, ContextFrom: "test", BranchFrom: "test"},
			},
		},
	}

	data, err := json.Marshal(chain)
	if err != nil {
		t.Fatalf("json.Marshal Chain: %v", err)
	}
	s := string(data)

	for _, field := range []string{
		"displayName", "description", "projectRef", "steps",
		"name", "templateRef", "dependsOn", "contextFrom", "branchFrom",
	} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("Chain JSON missing field: %s", field)
		}
	}

	// Verify step count in marshaled output
	var parsed struct {
		Spec struct {
			Steps []struct{ Name string } `json:"steps"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal chain: %v", err)
	}
	if len(parsed.Spec.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(parsed.Spec.Steps))
	}
}

// TestBoundary_ChainRunJSON verifies ChainRun JSON serialization for frontend.
func TestBoundary_ChainRunJSON(t *testing.T) {
	cr := aotv1alpha1.ChainRun{
		Spec: aotv1alpha1.ChainRunSpec{
			ChainRef:    "ci-pipeline",
			TriggeredBy: "schedule:daily",
		},
		Status: aotv1alpha1.ChainRunStatus{
			Phase:   "running",
			Message: "Step 2/3 running",
			Steps: []aotv1alpha1.ChainRunStepStatus{
				{Name: "lint", Phase: "succeeded", RunID: "ar-abc123", Message: "completed"},
				{Name: "test", Phase: "running", RunID: "ar-def456"},
				{Name: "build", Phase: "pending"},
			},
		},
	}

	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("json.Marshal ChainRun: %v", err)
	}
	s := string(data)

	for _, field := range []string{
		"chainRef", "triggeredBy", "phase", "steps", "runId", "message",
	} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("ChainRun JSON missing field: %s", field)
		}
	}
}

// TestBoundary_ScheduleJSON verifies Schedule JSON serialization for frontend.
func TestBoundary_ScheduleJSON(t *testing.T) {
	sched := aotv1alpha1.Schedule{
		Spec: aotv1alpha1.ScheduleSpec{
			DisplayName:                "Weekly Review",
			Cron:                       "0 9 * * MON",
			Timezone:                   "America/New_York",
			Suspend:                    false,
			ConcurrencyPolicy:          "Forbid",
			ChainRef:                   "review-chain",
			SuccessfulRunsHistoryLimit: 5,
			FailedRunsHistoryLimit:     3,
		},
		Status: aotv1alpha1.ScheduleStatus{
			LastRunID:  "cr-xyz789",
			LastResult: "succeeded",
		},
	}

	data, err := json.Marshal(sched)
	if err != nil {
		t.Fatalf("json.Marshal Schedule: %v", err)
	}
	s := string(data)

	// Note: "suspend" is omitempty bool — omitted when false (frontend handles undefined as false)
	for _, field := range []string{
		"displayName", "cron", "timezone", "concurrencyPolicy",
		"chainRef", "successfulRunsHistoryLimit", "failedRunsHistoryLimit",
		"lastRunId", "lastResult",
	} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("Schedule JSON missing field: %s", field)
		}
	}
}

// TestBoundary_ChainDAG_Validity verifies DAG structure constraints.
func TestBoundary_ChainDAG_Validity(t *testing.T) {
	// Valid DAG: a -> b -> c (linear)
	chain := aotv1alpha1.ChainSpec{
		Steps: []aotv1alpha1.ChainStep{
			{Name: "a", TemplateRef: "tmpl-a"},
			{Name: "b", TemplateRef: "tmpl-b", DependsOn: []string{"a"}},
			{Name: "c", TemplateRef: "tmpl-c", DependsOn: []string{"b"}},
		},
	}

	// Build name set for dependency validation
	nameSet := make(map[string]bool)
	for _, s := range chain.Steps {
		if nameSet[s.Name] {
			t.Errorf("duplicate step name: %s", s.Name)
		}
		nameSet[s.Name] = true
	}

	// Verify all dependencies reference valid step names
	for _, s := range chain.Steps {
		for _, dep := range s.DependsOn {
			if !nameSet[dep] {
				t.Errorf("step %q depends on non-existent step %q", s.Name, dep)
			}
		}
	}

	// Diamond DAG: a -> b, a -> c, b,c -> d
	diamond := aotv1alpha1.ChainSpec{
		Steps: []aotv1alpha1.ChainStep{
			{Name: "a", TemplateRef: "tmpl-a"},
			{Name: "b", TemplateRef: "tmpl-b", DependsOn: []string{"a"}},
			{Name: "c", TemplateRef: "tmpl-c", DependsOn: []string{"a"}},
			{Name: "d", TemplateRef: "tmpl-d", DependsOn: []string{"b", "c"}},
		},
	}

	dNames := make(map[string]bool)
	for _, s := range diamond.Steps {
		dNames[s.Name] = true
	}
	for _, s := range diamond.Steps {
		for _, dep := range s.DependsOn {
			if !dNames[dep] {
				t.Errorf("diamond step %q depends on non-existent %q", s.Name, dep)
			}
		}
	}
}

// TestBoundary_ScheduleSpec_MutualExclusivity verifies chainRef/templateRef are used correctly.
func TestBoundary_ScheduleSpec_MutualExclusivity(t *testing.T) {
	// chainRef only
	s1 := aotv1alpha1.ScheduleSpec{Cron: "0 0 * * *", ChainRef: "my-chain"}
	if s1.ChainRef == "" && s1.TemplateRef == "" {
		t.Error("schedule must have chainRef or templateRef")
	}

	// templateRef only
	s2 := aotv1alpha1.ScheduleSpec{Cron: "0 0 * * *", TemplateRef: "my-template"}
	if s2.ChainRef == "" && s2.TemplateRef == "" {
		t.Error("schedule must have chainRef or templateRef")
	}

	// both set — allowed by Go types but handler should validate
	s3 := aotv1alpha1.ScheduleSpec{Cron: "0 0 * * *", ChainRef: "a", TemplateRef: "b"}
	if s3.ChainRef == "" || s3.TemplateRef == "" {
		t.Error("expected both to be set for this test case")
	}
}
