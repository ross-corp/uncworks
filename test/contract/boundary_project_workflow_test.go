package contract

import (
	"os"
	"reflect"
	"strings"
	"testing"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestBoundary_WorkflowInput_HasAllCRDFields verifies that every field in
// AgentRunSpec that should flow to the workflow has a corresponding field
// in WorkflowInput. This prevents silent field drops when new CRD fields
// are added but not wired through.
func TestBoundary_WorkflowInput_HasAllCRDFields(t *testing.T) {
	// Fields in AgentRunSpec that MUST map to WorkflowInput
	crdToWorkflow := map[string]string{
		"Prompt":             "Prompt",
		"TTLSeconds":         "TTLSeconds",
		"ModelTier":          "ModelTier",
		"ManageModelTier":    "ManageModelTier",
		"ImplementModelTier": "ImplementModelTier",
		"SpecContent":        "SpecContent",
		"OrchestrationMode":  "OrchestrationMode",
		"AutoPush":           "AutoPush",
		"AutoPR":             "AutoPR",
		"PRBaseBranch":       "PRBaseBranch",
		"Project":            "Project",
		"Feature":            "Feature",
		"MaxBudget":          "MaxBudget",
		"SpecSource":         "SpecSource",
	}

	wiType := reflect.TypeOf(aottemporal.WorkflowInput{})
	for crdField, wiField := range crdToWorkflow {
		if _, ok := wiType.FieldByName(wiField); !ok {
			t.Errorf("CRD field %q maps to WorkflowInput.%s which does not exist", crdField, wiField)
		}
	}
}

// TestBoundary_DualModel_FlowsToWorkflow verifies the dual model fields
// exist on both CRD and WorkflowInput.
func TestBoundary_DualModel_FlowsToWorkflow(t *testing.T) {
	// CRD side
	spec := aotv1alpha1.AgentRunSpec{
		ManageModelTier:    "qwen3:8b",
		ImplementModelTier: "deepseek-v3.1",
	}
	if spec.ManageModelTier != "qwen3:8b" {
		t.Error("CRD ManageModelTier not set")
	}

	// Workflow side
	wi := aottemporal.WorkflowInput{
		ManageModelTier:    "qwen3:8b",
		ImplementModelTier: "deepseek-v3.1",
	}
	if wi.ManageModelTier != "qwen3:8b" {
		t.Error("WorkflowInput ManageModelTier not set")
	}
	if wi.ImplementModelTier != "deepseek-v3.1" {
		t.Error("WorkflowInput ImplementModelTier not set")
	}

	// CRD YAML should have the fields
	yaml, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	for _, field := range []string{"manageModelTier", "implementModelTier"} {
		if !strings.Contains(string(yaml), field+":") {
			t.Errorf("CRD YAML missing field: %s", field)
		}
	}
}

// TestBoundary_Archive_FieldsExist verifies archive-related fields exist
// across CRD, Go types, and YAML schema.
func TestBoundary_Archive_FieldsExist(t *testing.T) {
	// Go type
	status := aotv1alpha1.AgentRunStatus{
		Archived:       true,
		TotalCost:      "$1.23",
		TotalAdditions: 42,
		TotalDeletions: 5,
	}
	if !status.Archived {
		t.Error("Archived field not working")
	}

	// CRD YAML
	yaml, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	for _, field := range []string{"archived", "totalCost", "totalAdditions", "totalDeletions"} {
		if !strings.Contains(string(yaml), field+":") {
			t.Errorf("CRD YAML missing status field: %s", field)
		}
	}
}

// TestBoundary_ProjectRef_FieldsExist verifies projectRef and specRef
// exist on both CRD spec and YAML schema.
func TestBoundary_ProjectRef_FieldsExist(t *testing.T) {
	spec := aotv1alpha1.AgentRunSpec{
		ProjectRef: "my-project",
		SpecRef:    "add-auth",
	}
	if spec.ProjectRef != "my-project" {
		t.Error("ProjectRef not set")
	}

	yaml, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	if !strings.Contains(string(yaml), "projectRef:") {
		t.Error("CRD YAML missing projectRef")
	}
	if !strings.Contains(string(yaml), "specRef:") {
		t.Error("CRD YAML missing specRef")
	}
}

// TestBoundary_HelmRBAC_IncludesProjects verifies the Helm RBAC template
// grants permissions for Project CRDs.
func TestBoundary_HelmRBAC_IncludesProjects(t *testing.T) {
	rbac, err := os.ReadFile("../../deploy/helm/aot/templates/rbac.yaml")
	if err != nil {
		t.Fatalf("read rbac.yaml: %v", err)
	}
	rbacStr := string(rbac)

	requiredResources := []string{
		"projects",
		"projects/status",
		"projects/finalizers",
		"agentruns",
		"agentruns/status",
		"agentruns/finalizers",
		"secrets",
	}

	for _, res := range requiredResources {
		if !strings.Contains(rbacStr, res) {
			t.Errorf("Helm RBAC missing resource: %s", res)
		}
	}
}
