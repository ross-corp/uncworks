package contract

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestProtovalidate_* tests verify that the protovalidate interceptor
// correctly rejects invalid requests with INVALID_ARGUMENT error codes.

func TestProtovalidate_CreateAgentRun_EmptyPrompt(t *testing.T) {
	client, cleanup := startAOTServer(t, true)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "", // violates min_len = 1
		},
	}))
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestProtovalidate_CreateAgentRun_InvalidRepoURL(t *testing.T) {
	client, cleanup := startAOTServer(t, true)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "not-a-url"}}, // violates uri = true
			Prompt:  "do something",
		},
	}))
	if err == nil {
		t.Fatal("expected error for invalid repo URL")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestProtovalidate_CreateAgentRun_UnspecifiedBackend(t *testing.T) {
	client, cleanup := startAOTServer(t, true)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_UNSPECIFIED, // violates not_in: [0]
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "do something",
		},
	}))
	if err == nil {
		t.Fatal("expected error for UNSPECIFIED backend")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestProtovalidate_CreateAgentRun_NegativeTTL(t *testing.T) {
	client, cleanup := startAOTServer(t, true)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:     "do something",
			TtlSeconds: -1, // violates gte = 0
		},
	}))
	if err == nil {
		t.Fatal("expected error for negative TTL")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestProtovalidate_CreateAgentRun_ValidInput(t *testing.T) {
	client, cleanup := startAOTServer(t, true)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:     "valid prompt",
			TtlSeconds: 3600,
		},
	}))
	if err != nil {
		t.Fatalf("expected valid request to succeed: %v", err)
	}
	if resp.Msg.AgentRun.Id == "" {
		t.Error("expected non-empty ID")
	}
}
