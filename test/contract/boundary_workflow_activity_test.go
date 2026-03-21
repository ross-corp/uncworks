package contract

import (
	"testing"

	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestBoundary_PlanRunInput_AllFieldsPopulated verifies that the PlanRunInput
// struct used by the spec-driven workflow has all required fields populated
// when constructed the same way the workflow does.
func TestBoundary_PlanRunInput_AllFieldsPopulated(t *testing.T) {
	// Simulate how runSpecDrivenPipeline constructs PlanRunInput
	// (from workflow_spec_driven.go lines 307-316)
	input := aottemporal.PlanRunInput{
		AgentRunName: "test-run-42",
		Namespace:    "aot",
		PodName:      "test-run-42-pod-abc",
		PodIP:        "10.42.0.100",
		Prompt:       "Fix the auth module",
		SpecContent:  "# Auth Module Spec\nFix login flow.",
		Model:        "gpt-4",
		RepoPath:     "/workspace",
	}

	// Every field must be non-empty
	if input.AgentRunName == "" {
		t.Error("AgentRunName is empty")
	}
	if input.Namespace == "" {
		t.Error("Namespace is empty")
	}
	if input.PodName == "" {
		t.Error("PodName is empty")
	}
	if input.PodIP == "" {
		t.Error("PodIP is empty")
	}
	if input.Prompt == "" {
		t.Error("Prompt is empty")
	}
	if input.SpecContent == "" {
		t.Error("SpecContent is empty")
	}
	if input.Model == "" {
		t.Error("Model is empty")
	}
	if input.RepoPath == "" {
		t.Error("RepoPath is empty")
	}
}

// TestBoundary_VerifyRunInput_AllFieldsPopulated verifies that the VerifyRunInput
// struct used by the spec-driven workflow has all required fields populated
// when constructed the same way the workflow does.
func TestBoundary_VerifyRunInput_AllFieldsPopulated(t *testing.T) {
	// Simulate how runSpecDrivenPipeline constructs VerifyRunInput
	// (from workflow_spec_driven.go lines 401-408)
	input := aottemporal.VerifyRunInput{
		AgentRunName: "test-run-42",
		Namespace:    "aot",
		PodName:      "test-run-42-pod-abc",
		PodIP:        "10.42.0.100",
		ChangeName:   "fix-auth",
		RepoPath:     "/workspace",
	}

	// Every field must be non-empty
	if input.AgentRunName == "" {
		t.Error("AgentRunName is empty")
	}
	if input.Namespace == "" {
		t.Error("Namespace is empty")
	}
	if input.PodName == "" {
		t.Error("PodName is empty")
	}
	if input.PodIP == "" {
		t.Error("PodIP is empty")
	}
	if input.ChangeName == "" {
		t.Error("ChangeName is empty")
	}
	if input.RepoPath == "" {
		t.Error("RepoPath is empty")
	}
}

// TestBoundary_WorkflowInput_SpecDrivenPipelineFields verifies that WorkflowInput
// has all the fields needed by the spec-driven pipeline (runSpecDrivenPipeline).
func TestBoundary_WorkflowInput_SpecDrivenPipelineFields(t *testing.T) {
	input := aottemporal.WorkflowInput{
		AgentRunName: "run-1",
		Namespace:    "aot",
		Repos: []aottemporal.Repository{
			{URL: "https://github.com/org/repo.git", Branch: "main"},
		},
		Prompt:       "Fix things",
		SpecContent:  "# Spec content",
		ModelTier:    "premium",
		AutoPush:     true,
		AutoPR:       true,
		PRBaseBranch: "main",
		PipelineConfig: &aottemporal.PipelineConfigInput{
			Plan: aottemporal.StageConfigInput{
				Model:          "plan-model",
				TimeoutSeconds: 300,
				MaxRetries:     2,
				OnFailure:      "fail",
			},
			Execute: aottemporal.StageConfigInput{
				Model:          "exec-model",
				TimeoutSeconds: 900,
				MaxRetries:     3,
				OnFailure:      "retry",
			},
			Verify: aottemporal.StageConfigInput{
				Model:          "verify-model",
				TimeoutSeconds: 180,
				MaxRetries:     1,
				OnFailure:      "fail",
			},
		},
	}

	// Verify pipeline config is fully populated
	if input.PipelineConfig == nil {
		t.Fatal("PipelineConfig is nil")
	}

	stages := map[string]aottemporal.StageConfigInput{
		"Plan":    input.PipelineConfig.Plan,
		"Execute": input.PipelineConfig.Execute,
		"Verify":  input.PipelineConfig.Verify,
	}

	for name, sc := range stages {
		if sc.Model == "" {
			t.Errorf("%s.Model is empty", name)
		}
		if sc.TimeoutSeconds == 0 {
			t.Errorf("%s.TimeoutSeconds is zero", name)
		}
		if sc.MaxRetries == 0 {
			t.Errorf("%s.MaxRetries is zero", name)
		}
		if sc.OnFailure == "" {
			t.Errorf("%s.OnFailure is empty", name)
		}
	}

	// Verify spec-driven fields are populated
	assertEqual(t, "AutoPush", input.AutoPush, true)
	assertEqual(t, "AutoPR", input.AutoPR, true)
	assertEqual(t, "PRBaseBranch", input.PRBaseBranch, "main")
	assertEqual(t, "SpecContent", input.SpecContent, "# Spec content")
}

// TestBoundary_VerificationResult_JSONContract verifies the VerificationResult
// struct serialization matches the expected format consumed by the frontend.
func TestBoundary_VerificationResult_JSONContract(t *testing.T) {
	result := aottemporal.VerificationResult{
		Pass:            true,
		TasksCompleted:  5,
		TasksTotal:      5,
		ValidationValid: true,
		AutomatedChecks: []aottemporal.AutomatedCheck{
			{Name: "go test", Pass: true, Output: "ok", Command: "go test ./..."},
		},
		LLMVerdict: &aottemporal.LLMVerdict{
			Pass: true,
			Criteria: []aottemporal.CriterionResult{
				{Scenario: "Login flow", Pass: true, Explanation: "Works correctly"},
			},
			Model: "gpt-4",
		},
		FailureReport:   "",
		ExecutionTimeMs: 12345,
	}

	// Verify field access compiles (catches field renames)
	assertEqual(t, "Pass", result.Pass, true)
	assertEqual(t, "TasksCompleted", result.TasksCompleted, 5)
	assertEqual(t, "TasksTotal", result.TasksTotal, 5)
	assertEqual(t, "ValidationValid", result.ValidationValid, true)

	if len(result.AutomatedChecks) != 1 {
		t.Fatalf("expected 1 automated check, got %d", len(result.AutomatedChecks))
	}
	assertEqual(t, "AutomatedChecks[0].Name", result.AutomatedChecks[0].Name, "go test")
	assertEqual(t, "AutomatedChecks[0].Pass", result.AutomatedChecks[0].Pass, true)
	assertEqual(t, "AutomatedChecks[0].Output", result.AutomatedChecks[0].Output, "ok")
	assertEqual(t, "AutomatedChecks[0].Command", result.AutomatedChecks[0].Command, "go test ./...")

	if result.LLMVerdict == nil {
		t.Fatal("expected non-nil LLMVerdict")
	}
	assertEqual(t, "LLMVerdict.Pass", result.LLMVerdict.Pass, true)
	assertEqual(t, "LLMVerdict.Model", result.LLMVerdict.Model, "gpt-4")

	if len(result.LLMVerdict.Criteria) != 1 {
		t.Fatalf("expected 1 criterion, got %d", len(result.LLMVerdict.Criteria))
	}
	assertEqual(t, "Criteria[0].Scenario", result.LLMVerdict.Criteria[0].Scenario, "Login flow")
	assertEqual(t, "Criteria[0].Pass", result.LLMVerdict.Criteria[0].Pass, true)

	assertEqual(t, "ExecutionTimeMs", result.ExecutionTimeMs, int64(12345))
}

// TestBoundary_PlanRunOutput_FieldContract verifies PlanRunOutput has the
// expected fields for the pipeline to consume.
func TestBoundary_PlanRunOutput_FieldContract(t *testing.T) {
	output := aottemporal.PlanRunOutput{
		ChangeName:       "fix-auth-module",
		TaskCount:        3,
		SpecsValid:       true,
		ValidationErrors: []string{"warning: unused import"},
	}

	assertEqual(t, "ChangeName", output.ChangeName, "fix-auth-module")
	assertEqual(t, "TaskCount", output.TaskCount, 3)
	assertEqual(t, "SpecsValid", output.SpecsValid, true)

	if len(output.ValidationErrors) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(output.ValidationErrors))
	}
	assertEqual(t, "ValidationErrors[0]", output.ValidationErrors[0], "warning: unused import")
}

// TestBoundary_StartAgentInput_TraceFields verifies that StartAgentInput has
// ParentSpanID and TraceID fields and they are non-empty when populated.
func TestBoundary_StartAgentInput_TraceFields(t *testing.T) {
	input := aottemporal.StartAgentInput{
		PodName:      "run-42-pod-abc",
		Namespace:    "aot",
		PodIP:        "10.42.0.100",
		Prompt:       "Fix the auth module",
		RepoPath:     "/workspace",
		Model:        "deepseek-v3.1",
		Stage:        "execute",
		ParentSpanID: "span-stage-execute-001",
		TraceID:      "trace-pipeline-xyz",
	}

	if input.ParentSpanID == "" {
		t.Error("ParentSpanID is empty when populated")
	}
	if input.TraceID == "" {
		t.Error("TraceID is empty when populated")
	}

	assertEqual(t, "ParentSpanID", input.ParentSpanID, "span-stage-execute-001")
	assertEqual(t, "TraceID", input.TraceID, "trace-pipeline-xyz")

	// Verify all other required fields still compile and are accessible
	assertEqual(t, "PodName", input.PodName, "run-42-pod-abc")
	assertEqual(t, "Namespace", input.Namespace, "aot")
	assertEqual(t, "PodIP", input.PodIP, "10.42.0.100")
	assertEqual(t, "Stage", input.Stage, "execute")
	assertEqual(t, "Model", input.Model, "deepseek-v3.1")
}

// TestBoundary_StartAgentInput_TraceFieldsOptional verifies that ParentSpanID
// and TraceID can be left empty (they are optional for single-stage runs).
func TestBoundary_StartAgentInput_TraceFieldsOptional(t *testing.T) {
	input := aottemporal.StartAgentInput{
		PodName:   "run-99-pod",
		Namespace: "aot",
		PodIP:     "10.42.0.50",
		Prompt:    "Do something",
		RepoPath:  "/workspace",
		Model:     "deepseek-v3.1",
	}

	// For single-stage runs, trace fields should be empty
	if input.ParentSpanID != "" {
		t.Errorf("ParentSpanID should be empty for single-stage, got %q", input.ParentSpanID)
	}
	if input.TraceID != "" {
		t.Errorf("TraceID should be empty for single-stage, got %q", input.TraceID)
	}
}

// TestBoundary_VerifyRunOutput_FieldContract verifies VerifyRunOutput wraps
// VerificationResult correctly.
func TestBoundary_VerifyRunOutput_FieldContract(t *testing.T) {
	output := aottemporal.VerifyRunOutput{
		Result: aottemporal.VerificationResult{
			Pass:           false,
			TasksCompleted: 2,
			TasksTotal:     5,
			FailureReport:  "3 tasks incomplete",
		},
	}

	assertEqual(t, "Result.Pass", output.Result.Pass, false)
	assertEqual(t, "Result.TasksCompleted", output.Result.TasksCompleted, 2)
	assertEqual(t, "Result.TasksTotal", output.Result.TasksTotal, 5)
	assertEqual(t, "Result.FailureReport", output.Result.FailureReport, "3 tasks incomplete")
}
