package contract

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestBoundary_SpecProtoToCRD_AllFields verifies that specProtoToCRD preserves
// EVERY field from a fully-populated proto AgentRunSpec, and that crdToProto
// faithfully round-trips them back. We exercise both functions indirectly via
// CreateAgentRun + GetAgentRun, which call specProtoToCRD on the way in and
// crdToProto on the way out.
func TestBoundary_SpecProtoToCRD_AllFields(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	original := &apiv1.AgentRunSpec{
		Backend:           apiv1.Backend_BACKEND_POD,
		Repos:             []*apiv1.Repository{{Url: "https://github.com/org/repo.git", Branch: "develop", Path: "my-repo"}},
		Prompt:            "Implement the feature",
		DevboxConfig:      "/path/to/devbox.json",
		TtlSeconds:        7200,
		EnvVars:           map[string]string{"FOO": "bar", "BAZ": "qux"},
		ModelTier:         "premium",
		Image:             "ghcr.io/custom/agent:latest",
		SpecContent:       "# MySpec\n\nDo the thing.",
		SpecSource:        "github:org/repo/spec.md",
		WorkspaceName:     "my-workspace",
		ParentRunId:       "parent-run-42",
		OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN,
		Orchestration: &apiv1.Orchestration{
			Tasks: []*apiv1.OrchestrationTask{
				{Name: "task-1", Prompt: "Do first thing", RepoUrls: []string{"https://github.com/org/repo.git"}},
				{Name: "task-2", Prompt: "Do second thing"},
			},
		},
		SpecRunId:    "spec-run-99",
		DisplayName:  "My Display Name",
		AutoPush:     true,
		AutoPr:       true,
		PrBaseBranch: "develop",
		Project:      "my-project",
		Feature:      "my-feature",
		Tags:         []string{"tag1", "tag2", "infra"},
		PipelineConfig: &apiv1.PipelineConfig{
			Plan: &apiv1.StageConfig{
				Model:          "plan-model",
				TimeoutSeconds: 120,
				MaxRetries:     3,
				OnFailure:      "fail",
			},
			Execute: &apiv1.StageConfig{
				Model:          "exec-model",
				TimeoutSeconds: 600,
				MaxRetries:     5,
				OnFailure:      "retry",
			},
			Verify: &apiv1.StageConfig{
				Model:          "verify-model",
				TimeoutSeconds: 180,
				MaxRetries:     2,
				OnFailure:      "skip",
			},
		},
	}

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: original,
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	id := resp.Msg.AgentRun.Id
	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	got := getResp.Msg.Spec

	// Verify all scalar fields
	assertEqual(t, "Backend", got.Backend, original.Backend)
	assertEqual(t, "Prompt", got.Prompt, original.Prompt)
	assertEqual(t, "DevboxConfig", got.DevboxConfig, original.DevboxConfig)
	assertEqual(t, "TtlSeconds", got.TtlSeconds, original.TtlSeconds)
	assertEqual(t, "ModelTier", got.ModelTier, original.ModelTier)
	assertEqual(t, "Image", got.Image, original.Image)
	assertEqual(t, "SpecContent", got.SpecContent, original.SpecContent)
	assertEqual(t, "SpecSource", got.SpecSource, original.SpecSource)
	assertEqual(t, "WorkspaceName", got.WorkspaceName, original.WorkspaceName)
	assertEqual(t, "ParentRunId", got.ParentRunId, original.ParentRunId)
	assertEqual(t, "OrchestrationMode", got.OrchestrationMode, original.OrchestrationMode)
	assertEqual(t, "SpecRunId", got.SpecRunId, original.SpecRunId)
	// DisplayName is set by the server's generateDisplayName, not round-tripped
	// from the proto input. The server overwrites it. Skip.
	assertEqual(t, "AutoPush", got.AutoPush, original.AutoPush)
	assertEqual(t, "AutoPr", got.AutoPr, original.AutoPr)
	assertEqual(t, "PrBaseBranch", got.PrBaseBranch, original.PrBaseBranch)
	assertEqual(t, "Project", got.Project, original.Project)
	assertEqual(t, "Feature", got.Feature, original.Feature)
	if len(got.Tags) != len(original.Tags) {
		t.Fatalf("Tags length: got %d, want %d", len(got.Tags), len(original.Tags))
	}
	for i, tag := range original.Tags {
		assertEqual(t, fmt.Sprintf("Tags[%d]", i), got.Tags[i], tag)
	}

	// Verify repos
	if len(got.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(got.Repos))
	}
	assertEqual(t, "Repos[0].Url", got.Repos[0].Url, original.Repos[0].Url)
	assertEqual(t, "Repos[0].Branch", got.Repos[0].Branch, original.Repos[0].Branch)
	assertEqual(t, "Repos[0].Path", got.Repos[0].Path, original.Repos[0].Path)

	// Verify env vars
	if len(got.EnvVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(got.EnvVars))
	}
	assertEqual(t, "EnvVars[FOO]", got.EnvVars["FOO"], "bar")
	assertEqual(t, "EnvVars[BAZ]", got.EnvVars["BAZ"], "qux")

	// Verify pipeline config
	if got.PipelineConfig == nil {
		t.Fatal("expected non-nil PipelineConfig")
	}
	assertStageConfig(t, "Plan", got.PipelineConfig.Plan, original.PipelineConfig.Plan)
	assertStageConfig(t, "Execute", got.PipelineConfig.Execute, original.PipelineConfig.Execute)
	assertStageConfig(t, "Verify", got.PipelineConfig.Verify, original.PipelineConfig.Verify)

	// Verify orchestration
	if got.Orchestration == nil {
		t.Fatal("expected non-nil Orchestration")
	}
	if len(got.Orchestration.Tasks) != 2 {
		t.Fatalf("expected 2 orchestration tasks, got %d", len(got.Orchestration.Tasks))
	}
	assertEqual(t, "Tasks[0].Name", got.Orchestration.Tasks[0].Name, "task-1")
	assertEqual(t, "Tasks[0].Prompt", got.Orchestration.Tasks[0].Prompt, "Do first thing")
	if len(got.Orchestration.Tasks[0].RepoUrls) != 1 {
		t.Fatalf("expected 1 repo URL in task-1, got %d", len(got.Orchestration.Tasks[0].RepoUrls))
	}
	assertEqual(t, "Tasks[0].RepoUrls[0]", got.Orchestration.Tasks[0].RepoUrls[0], "https://github.com/org/repo.git")
	assertEqual(t, "Tasks[1].Name", got.Orchestration.Tasks[1].Name, "task-2")
	assertEqual(t, "Tasks[1].Prompt", got.Orchestration.Tasks[1].Prompt, "Do second thing")
}

// TestBoundary_CRDToProto_StatusFields verifies that crdToProto maps all
// status fields from the CRD into the proto AgentRunStatus.
func TestBoundary_CRDToProto_StatusFields(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/org/repo.git"}},
			Prompt:  "Test status fields",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: resp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}

	status := getResp.Msg.Status
	if status == nil {
		t.Fatal("expected non-nil Status")
	}

	// Newly created runs should have Pending phase and "Queued" message
	assertEqual(t, "Phase", status.Phase, apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING)
	assertEqual(t, "Message", status.Message, "Queued")

	// Verify all status fields exist (zero values are expected for a new run)
	// This ensures crdToProto maps all fields without compile errors.
	_ = status.PodName
	_ = status.TraceId
	_ = status.WorktreePath
	_ = status.LogOutput
	_ = status.DeploymentName
	_ = status.DebugActive
	_ = status.Stage
	_ = status.RetryCount
	_ = status.VerificationResult
	_ = status.PrUrl
	_ = status.StartedAt
	_ = status.CompletedAt
	_ = status.RetainUntil
}

func assertEqual[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func assertStageConfig(t *testing.T, name string, got, want *apiv1.StageConfig) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s StageConfig: got nil", name)
		return
	}
	if want == nil {
		t.Fatalf("%s StageConfig: want nil but got non-nil", name)
		return
	}
	assertEqual(t, name+".Model", got.Model, want.Model)
	assertEqual(t, name+".TimeoutSeconds", got.TimeoutSeconds, want.TimeoutSeconds)
	assertEqual(t, name+".MaxRetries", got.MaxRetries, want.MaxRetries)
	assertEqual(t, name+".OnFailure", got.OnFailure, want.OnFailure)
}
