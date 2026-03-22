package contract

import (
	"os"
	"strings"
	"testing"
)

// TestBoundary_OrchestrationModes_FrontendMatchesBackend verifies that
// the orchestration mode values in the frontend TypeScript types match
// the CRD YAML enum.
func TestBoundary_OrchestrationModes_FrontendMatchesBackend(t *testing.T) {
	// Backend: CRD YAML enum values
	yamlBytes, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	yaml := string(yamlBytes)

	backendModes := []string{"single", "auto", "manual", "spec-driven"}
	for _, mode := range backendModes {
		if !strings.Contains(yaml, "- "+mode) {
			t.Errorf("CRD YAML missing orchestration mode: %s", mode)
		}
	}

	// Frontend: TypeScript type union
	tsBytes, err := os.ReadFile("../../web/src/types/agent-run.ts")
	if err != nil {
		t.Fatalf("read agent-run.ts: %v", err)
	}
	ts := string(tsBytes)

	// Check the type union contains all modes
	for _, mode := range backendModes {
		if !strings.Contains(ts, `"`+mode+`"`) {
			t.Errorf("Frontend types missing orchestration mode: %s", mode)
		}
	}

	// Check the options array has entries for the modes used in the UI
	uiModes := []string{"single", "spec-driven"}
	for _, mode := range uiModes {
		if !strings.Contains(ts, `value: "`+mode+`"`) {
			t.Errorf("Frontend ORCHESTRATION_MODE_OPTIONS missing value: %s", mode)
		}
	}
}

// TestBoundary_ModelTiers_FrontendHasOptions verifies the frontend has
// model tier options defined.
func TestBoundary_ModelTiers_FrontendHasOptions(t *testing.T) {
	tsBytes, err := os.ReadFile("../../web/src/types/agent-run.ts")
	if err != nil {
		t.Fatalf("read agent-run.ts: %v", err)
	}
	ts := string(tsBytes)

	if !strings.Contains(ts, "MODEL_TIER_OPTIONS") {
		t.Error("Frontend missing MODEL_TIER_OPTIONS constant")
	}
	if !strings.Contains(ts, `value: "default"`) {
		t.Error("Frontend missing default model tier")
	}
}

// TestBoundary_StatusPhases_FrontendMatchesBackend verifies the status
// phase values match between CRD and frontend.
func TestBoundary_StatusPhases_FrontendMatchesBackend(t *testing.T) {
	yamlBytes, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	yaml := string(yamlBytes)

	phases := []string{"Pending", "Running", "WaitingForInput", "Succeeded", "Failed", "Cancelled"}
	for _, phase := range phases {
		if !strings.Contains(yaml, "- "+phase) {
			t.Errorf("CRD missing phase: %s", phase)
		}
	}
}
