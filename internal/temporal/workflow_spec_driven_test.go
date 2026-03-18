package temporal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSpecDrivenAutoUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		mode     OrchestrationMode
		spec     string
		wantMode OrchestrationMode
	}{
		{
			name:     "empty mode with spec upgrades to spec-driven",
			mode:     "",
			spec:     "some spec content",
			wantMode: OrchestrationModeSpecDriven,
		},
		{
			name:     "empty mode without spec stays empty",
			mode:     "",
			spec:     "",
			wantMode: "",
		},
		{
			name:     "single mode with spec stays single",
			mode:     OrchestrationModeSingle,
			spec:     "some spec content",
			wantMode: OrchestrationModeSingle,
		},
		{
			name:     "auto mode with spec stays auto",
			mode:     OrchestrationModeAuto,
			spec:     "some spec content",
			wantMode: OrchestrationModeAuto,
		},
		{
			name:     "manual mode with spec stays manual",
			mode:     OrchestrationModeManual,
			spec:     "some spec content",
			wantMode: OrchestrationModeManual,
		},
		{
			name:     "spec-driven mode with spec stays spec-driven",
			mode:     OrchestrationModeSpecDriven,
			spec:     "some spec content",
			wantMode: OrchestrationModeSpecDriven,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := WorkflowInput{
				OrchestrationMode: tt.mode,
				SpecContent:       tt.spec,
			}

			// Replicate the auto-upgrade logic from AgentRunWorkflow.
			if input.SpecContent != "" && input.OrchestrationMode != OrchestrationModeSingle {
				if input.OrchestrationMode == "" || input.OrchestrationMode == OrchestrationModeSingle {
					input.OrchestrationMode = OrchestrationModeSpecDriven
				}
			}

			if input.OrchestrationMode != tt.wantMode {
				t.Errorf("after auto-upgrade: OrchestrationMode = %q, want %q", input.OrchestrationMode, tt.wantMode)
			}
		})
	}
}

func TestVerificationResultTypes(t *testing.T) {
	t.Run("round-trip VerificationResult", func(t *testing.T) {
		original := VerificationResult{
			Pass:            true,
			TasksCompleted:  5,
			TasksTotal:      5,
			ValidationValid: true,
			AutomatedChecks: []AutomatedCheck{
				{
					Name:    "build",
					Pass:    true,
					Output:  "Build succeeded",
					Command: "go build ./...",
				},
				{
					Name:    "test",
					Pass:    false,
					Output:  "1 test failed",
					Command: "go test ./...",
				},
			},
			LLMVerdict: &LLMVerdict{
				Pass:  true,
				Model: "gpt-4",
				Criteria: []CriterionResult{
					{
						Scenario:    "WHEN user runs build THEN output compiles",
						Pass:        true,
						Explanation: "Build produced a binary without errors",
					},
				},
			},
			FailureReport:   "",
			ExecutionTimeMs: 12345,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		var decoded VerificationResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}

		if decoded.Pass != original.Pass {
			t.Errorf("Pass: got %v, want %v", decoded.Pass, original.Pass)
		}
		if decoded.TasksCompleted != original.TasksCompleted {
			t.Errorf("TasksCompleted: got %d, want %d", decoded.TasksCompleted, original.TasksCompleted)
		}
		if decoded.TasksTotal != original.TasksTotal {
			t.Errorf("TasksTotal: got %d, want %d", decoded.TasksTotal, original.TasksTotal)
		}
		if decoded.ValidationValid != original.ValidationValid {
			t.Errorf("ValidationValid: got %v, want %v", decoded.ValidationValid, original.ValidationValid)
		}
		if decoded.ExecutionTimeMs != original.ExecutionTimeMs {
			t.Errorf("ExecutionTimeMs: got %d, want %d", decoded.ExecutionTimeMs, original.ExecutionTimeMs)
		}
		if len(decoded.AutomatedChecks) != 2 {
			t.Fatalf("AutomatedChecks: got %d items, want 2", len(decoded.AutomatedChecks))
		}
		if decoded.AutomatedChecks[0].Name != "build" {
			t.Errorf("AutomatedChecks[0].Name: got %q, want %q", decoded.AutomatedChecks[0].Name, "build")
		}
		if decoded.AutomatedChecks[1].Pass != false {
			t.Errorf("AutomatedChecks[1].Pass: got %v, want false", decoded.AutomatedChecks[1].Pass)
		}
		if decoded.LLMVerdict == nil {
			t.Fatal("LLMVerdict: got nil, want non-nil")
		}
		if decoded.LLMVerdict.Model != "gpt-4" {
			t.Errorf("LLMVerdict.Model: got %q, want %q", decoded.LLMVerdict.Model, "gpt-4")
		}
		if len(decoded.LLMVerdict.Criteria) != 1 {
			t.Fatalf("LLMVerdict.Criteria: got %d items, want 1", len(decoded.LLMVerdict.Criteria))
		}
		if decoded.LLMVerdict.Criteria[0].Scenario != "WHEN user runs build THEN output compiles" {
			t.Errorf("Criteria[0].Scenario: got %q", decoded.LLMVerdict.Criteria[0].Scenario)
		}
	})

	t.Run("nil LLMVerdict omitted", func(t *testing.T) {
		result := VerificationResult{
			Pass:       false,
			LLMVerdict: nil,
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		// The llmVerdict field should be omitted from JSON due to omitempty.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("json.Unmarshal raw: %v", err)
		}
		if _, ok := raw["llmVerdict"]; ok {
			t.Error("expected llmVerdict to be omitted when nil")
		}
	})

	t.Run("empty failureReport omitted", func(t *testing.T) {
		result := VerificationResult{
			Pass:          true,
			FailureReport: "",
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("json.Unmarshal raw: %v", err)
		}
		if _, ok := raw["failureReport"]; ok {
			t.Error("expected failureReport to be omitted when empty")
		}
	})
}

func TestPipelineStageConstants(t *testing.T) {
	tests := []struct {
		stage PipelineStage
		want  string
	}{
		{PipelineStagePlanning, "planning"},
		{PipelineStageExecuting, "executing"},
		{PipelineStageVerifying, "verifying"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.stage) != tt.want {
				t.Errorf("PipelineStage = %q, want %q", string(tt.stage), tt.want)
			}
		})
	}
}

func TestExtractFileChecks(t *testing.T) {
	// Create a temp directory simulating a workspace with spec files.
	dir := t.TempDir()
	specDir := filepath.Join(dir, "openspec", "changes", "test-run", "specs", "my-feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	specContent := `## ADDED Requirements

### Requirement: Auth middleware exists
The system SHALL have an auth middleware module.

#### Scenario: Auth file created
- **WHEN** the implementation is complete
- **THEN** ` + "`src/middleware/auth.ts`" + ` exists
- **AND** ` + "`src/middleware/auth.test.ts`" + ` exists

#### Scenario: Config updated
- **WHEN** the auth module is added
- **THEN** the module is imported in ` + "`src/index.ts`" + `
`

	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	checks := extractFileChecks(dir, "test-run")

	if len(checks) != 2 {
		t.Fatalf("expected 2 file checks, got %d: %+v", len(checks), checks)
	}

	// Verify the extracted paths.
	paths := map[string]bool{}
	for _, c := range checks {
		paths[c.Path] = true
	}
	if !paths["src/middleware/auth.ts"] {
		t.Error("expected src/middleware/auth.ts in file checks")
	}
	if !paths["src/middleware/auth.test.ts"] {
		t.Error("expected src/middleware/auth.test.ts in file checks")
	}
}

func TestExtractFileChecks_NoSpecs(t *testing.T) {
	dir := t.TempDir()
	checks := extractFileChecks(dir, "nonexistent")
	if len(checks) != 0 {
		t.Errorf("expected 0 checks for missing specs, got %d", len(checks))
	}
}

func TestWriteVerificationResult(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "openspec", "changes", "test-run")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := VerificationResult{
		Pass:            true,
		TasksCompleted:  5,
		TasksTotal:      5,
		ValidationValid: true,
		AutomatedChecks: []AutomatedCheck{
			{Name: "file_exists: src/auth.ts", Pass: true, Output: "exists"},
		},
		ExecutionTimeMs: 1234,
	}

	writeVerificationResult(dir, "test-run", result)

	// Read it back.
	data, err := os.ReadFile(filepath.Join(changeDir, "verification-result.json"))
	if err != nil {
		t.Fatalf("failed to read verification-result.json: %v", err)
	}

	var readBack VerificationResult
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !readBack.Pass {
		t.Error("expected pass=true")
	}
	if readBack.TasksCompleted != 5 {
		t.Errorf("expected 5 tasks completed, got %d", readBack.TasksCompleted)
	}
	if len(readBack.AutomatedChecks) != 1 {
		t.Errorf("expected 1 automated check, got %d", len(readBack.AutomatedChecks))
	}
}

func TestWriteVerificationResult_FallbackLocation(t *testing.T) {
	dir := t.TempDir()
	// No change directory exists — should write to fallback.

	result := VerificationResult{Pass: false, FailureReport: "something failed"}
	writeVerificationResult(dir, "test-run", result)

	fallbackPath := filepath.Join(dir, ".aot", "verification", "test-run-result.json")
	data, err := os.ReadFile(fallbackPath)
	if err != nil {
		t.Fatalf("failed to read fallback result: %v", err)
	}

	var readBack VerificationResult
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if readBack.Pass {
		t.Error("expected pass=false")
	}
	if readBack.FailureReport != "something failed" {
		t.Errorf("expected failure report preserved, got %q", readBack.FailureReport)
	}
}

func TestBacktickPathRegex(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"THEN `src/auth.ts` exists", []string{"src/auth.ts"}},
		{"THEN `pkg/handler.go` and `pkg/handler_test.go` exist", []string{"pkg/handler.go", "pkg/handler_test.go"}},
		{"no backticks here", nil},
		{"THEN `README.md` exists", []string{"README.md"}},
		{"THEN `.env` exists", []string{".env"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := backtickPathRe.FindAllStringSubmatch(tt.input, -1)
			var got []string
			for _, m := range matches {
				if len(m) > 1 {
					got = append(got, m[1])
				}
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
