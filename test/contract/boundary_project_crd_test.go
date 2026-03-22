package contract

import (
	"os"
	"strings"
	"testing"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// TestBoundary_ProjectCRD_SchemaCompleteness verifies that all fields in the
// Go Project types have corresponding entries in the CRD YAML schema.
func TestBoundary_ProjectCRD_SchemaCompleteness(t *testing.T) {
	yamlBytes, err := os.ReadFile("../../deploy/crds/project-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Fields that MUST appear in the CRD YAML
	specFields := []string{
		"displayName",
		"description",
		"repos",
		"devbox",
		"defaults",
		"ide",
		"ssh",
	}

	statusFields := []string{
		"configRepoReady",
		"configRepoURL",
		"ideActive",
		"idePodName",
		"runCount",
		"lastRunId",
		"lastRunAt",
		"totalCost",
		"conditions",
	}

	devboxFields := []string{"packages"}
	defaultsFields := []string{
		"modelTier",
		"manageModelTier",
		"implementModelTier",
		"ttlSeconds",
		"orchestrationMode",
		"autoPush",
		"autoPR",
		"prBaseBranch",
	}
	ideFields := []string{"enabled", "image", "idleTimeoutMinutes"}
	sshFields := []string{"enabled", "authorizedKeys"}

	for _, f := range specFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing spec field: %s", f)
		}
	}
	for _, f := range statusFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing status field: %s", f)
		}
	}
	for _, f := range devboxFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing devbox field: %s", f)
		}
	}
	for _, f := range defaultsFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing defaults field: %s", f)
		}
	}
	for _, f := range ideFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing ide field: %s", f)
		}
	}
	for _, f := range sshFields {
		if !strings.Contains(yaml, f+":") {
			t.Errorf("Project CRD YAML missing ssh field: %s", f)
		}
	}

	// Verify Go types compile and have expected fields (compile-time check)
	_ = aotv1alpha1.Project{}
	_ = aotv1alpha1.ProjectSpec{
		Devbox:   &aotv1alpha1.DevboxConfig{Packages: []string{"go"}},
		Defaults: &aotv1alpha1.ProjectDefaults{ModelTier: "default"},
		IDE:      &aotv1alpha1.IDEConfig{Enabled: true},
		SSH:      &aotv1alpha1.SSHConfig{Enabled: true, AuthorizedKeys: []string{"ssh-ed25519 test"}},
	}
	_ = aotv1alpha1.ProjectStatus{
		ConfigRepoReady: true,
		ConfigRepoURL:   "ssh://test",
		RunCount:        5,
		TotalCost:       "$1.00",
	}
}

// TestBoundary_AgentRun_HasProjectRef verifies that AgentRunSpec has projectRef and specRef fields.
func TestBoundary_AgentRun_HasProjectRef(t *testing.T) {
	spec := aotv1alpha1.AgentRunSpec{
		ProjectRef: "my-project",
		SpecRef:    "add-auth",
	}
	if spec.ProjectRef != "my-project" {
		t.Error("ProjectRef not set correctly")
	}
	if spec.SpecRef != "add-auth" {
		t.Error("SpecRef not set correctly")
	}

	// Verify CRD YAML has the fields
	yamlBytes, err := os.ReadFile("../../deploy/crds/agentrun-crd.yaml")
	if err != nil {
		t.Fatalf("read AgentRun CRD: %v", err)
	}
	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "projectRef:") {
		t.Error("AgentRun CRD missing projectRef field")
	}
	if !strings.Contains(yaml, "specRef:") {
		t.Error("AgentRun CRD missing specRef field")
	}
}
