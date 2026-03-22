package contract

import (
	"os"
	"strings"
	"testing"
)

// TestBoundary_CRDSchema_HasAllSpecFields verifies the CRD YAML schema
// includes every field defined in the Go AgentRunSpec type.
// This catches the bug where Go types had autoPush/autoPR/prBaseBranch
// but the CRD YAML didn't, causing Kubernetes to silently drop them.
func TestBoundary_CRDSchema_HasAllSpecFields(t *testing.T) {
	crdPath := "../../deploy/crds/agentrun-crd.yaml"
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("failed to read CRD YAML: %v", err)
	}
	schema := string(data)

	// These are all the spec fields defined in api/v1alpha1/types.go
	// that MUST appear in the CRD YAML schema
	requiredFields := []string{
		"prompt:",
		"displayName:",
		"autoPush:",
		"autoPR:",
		"prBaseBranch:",
		"maxBudget:",
		"project:",
		"feature:",
		"tags:",
		"ttlSeconds:",
		"modelTier:",
		"specContent:",
		"specSource:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(schema, field) {
			t.Errorf("CRD YAML schema missing field %q — add it to deploy/crds/agentrun-crd.yaml", field)
		}
	}
}
