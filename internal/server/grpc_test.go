package server

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func startTestServer(t *testing.T) (apiv1.AOTServiceClient, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := NewGRPCServer(0)
	go func() { _ = s.server.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	client := apiv1.NewAOTServiceClient(conn)
	return client, func() {
		_ = conn.Close()
		s.Stop()
	}
}

func TestCreateAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "Fix the tests",
		},
	})
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	if resp.AgentRun.Id == "" {
		t.Error("expected non-empty ID")
	}
	if resp.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		t.Errorf("expected PENDING phase, got %v", resp.AgentRun.Status.Phase)
	}
}

func TestCreateAgentRun_NilSpec(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{})
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestGetAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create first
	resp, _ := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "Test get",
		},
	})

	// Get it
	run, err := client.GetAgentRun(context.Background(), &apiv1.GetAgentRunRequest{Id: resp.AgentRun.Id})
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if run.Spec.Prompt != "Test get" {
		t.Errorf("expected prompt 'Test get', got %q", run.Spec.Prompt)
	}
}

func TestGetAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.GetAgentRun(context.Background(), &apiv1.GetAgentRunRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestListAgentRuns(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create two runs
	for _, prompt := range []string{"task 1", "task 2"} {
		if _, err := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				RepoUrl: "https://github.com/example/repo.git",
				Prompt:  prompt,
			},
		}); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), &apiv1.ListAgentRunsRequest{})
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.AgentRuns) != 2 {
		t.Errorf("expected 2 runs, got %d", len(resp.AgentRuns))
	}
}

func TestListAgentRuns_WithLimit(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		if _, err := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				RepoUrl: "https://github.com/example/repo.git",
				Prompt:  "task",
			},
		}); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), &apiv1.ListAgentRunsRequest{Limit: 2})
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.AgentRuns) != 2 {
		t.Errorf("expected 2 runs with limit, got %d", len(resp.AgentRuns))
	}
}

func TestCancelAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, _ := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "cancel me",
		},
	})

	cancelResp, err := client.CancelAgentRun(context.Background(), &apiv1.CancelAgentRunRequest{Id: resp.AgentRun.Id})
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	if cancelResp.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
		t.Errorf("expected CANCELLED, got %v", cancelResp.AgentRun.Status.Phase)
	}
}

func TestCancelAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CancelAgentRun(context.Background(), &apiv1.CancelAgentRunRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestSendHumanInput_NotWaiting(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, _ := client.CreateAgentRun(context.Background(), &apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "not waiting",
		},
	})

	_, err := client.SendHumanInput(context.Background(), &apiv1.SendHumanInputRequest{
		AgentRunId: resp.AgentRun.Id,
		Input:      "hello",
	})
	if err == nil {
		t.Fatal("expected error: agent not waiting for input")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestSendHumanInput_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SendHumanInput(context.Background(), &apiv1.SendHumanInputRequest{
		AgentRunId: "nonexistent",
		Input:      "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}
