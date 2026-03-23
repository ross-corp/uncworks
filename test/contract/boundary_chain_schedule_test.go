package contract

import (
	"os"
	"strings"
	"testing"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func TestBoundary_RunTemplate_SchemaCompleteness(t *testing.T) {
	yaml, err := os.ReadFile("../../deploy/crds/runtemplate-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	y := string(yaml)

	specFields := []string{"displayName", "description", "projectRef", "repos", "prompt",
		"modelTier", "manageModelTier", "implementModelTier", "orchestrationMode",
		"ttlSeconds", "autoPush", "autoPR", "prBaseBranch", "specRef"}
	for _, f := range specFields {
		if !strings.Contains(y, f+":") {
			t.Errorf("RunTemplate CRD missing field: %s", f)
		}
	}

	// Verify Go types compile
	_ = aotv1alpha1.RunTemplate{
		Spec: aotv1alpha1.RunTemplateSpec{
			Prompt:    "test",
			ModelTier: "default",
		},
	}
}

func TestBoundary_Chain_SchemaCompleteness(t *testing.T) {
	yaml, err := os.ReadFile("../../deploy/crds/chain-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	y := string(yaml)

	for _, f := range []string{"displayName", "description", "projectRef", "steps",
		"name", "templateRef", "dependsOn", "contextFrom", "branchFrom", "condition"} {
		if !strings.Contains(y, f+":") {
			t.Errorf("Chain CRD missing field: %s", f)
		}
	}

	_ = aotv1alpha1.Chain{
		Spec: aotv1alpha1.ChainSpec{
			Steps: []aotv1alpha1.ChainStep{
				{Name: "analyze", TemplateRef: "code-analysis"},
				{Name: "fix", TemplateRef: "fix-issues", DependsOn: []string{"analyze"}, ContextFrom: "analyze"},
			},
		},
	}
}

func TestBoundary_ChainRun_SchemaCompleteness(t *testing.T) {
	yaml, err := os.ReadFile("../../deploy/crds/chainrun-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	y := string(yaml)

	for _, f := range []string{"chainRef", "triggeredBy", "phase", "steps", "runId",
		"startedAt", "completedAt", "message"} {
		if !strings.Contains(y, f+":") {
			t.Errorf("ChainRun CRD missing field: %s", f)
		}
	}

	_ = aotv1alpha1.ChainRun{
		Spec: aotv1alpha1.ChainRunSpec{ChainRef: "test", TriggeredBy: "manual"},
	}
}

func TestBoundary_Schedule_SchemaCompleteness(t *testing.T) {
	yaml, err := os.ReadFile("../../deploy/crds/schedule-crd.yaml")
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	y := string(yaml)

	for _, f := range []string{"cron", "timezone", "suspend", "concurrencyPolicy",
		"chainRef", "templateRef", "successfulRunsHistoryLimit", "failedRunsHistoryLimit",
		"lastScheduledTime", "lastRunId", "lastResult", "nextScheduleTime", "active"} {
		if !strings.Contains(y, f+":") {
			t.Errorf("Schedule CRD missing field: %s", f)
		}
	}

	_ = aotv1alpha1.Schedule{
		Spec: aotv1alpha1.ScheduleSpec{
			Cron:              "0 9 * * MON",
			ChainRef:          "weekly-review",
			ConcurrencyPolicy: "Forbid",
		},
	}
}

func TestBoundary_HelmRBAC_IncludesChainResources(t *testing.T) {
	rbac, err := os.ReadFile("../../deploy/helm/aot/templates/rbac.yaml")
	if err != nil {
		t.Fatalf("read rbac: %v", err)
	}
	y := string(rbac)

	for _, r := range []string{"runtemplates", "chains", "chainruns", "schedules"} {
		if !strings.Contains(y, r) {
			t.Errorf("Helm RBAC missing resource: %s", r)
		}
	}
}
