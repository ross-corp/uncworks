package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
)

func startTestServer(t *testing.T) (apiv1connect.AOTServiceClient, func()) {
	t.Helper()

	svc := NewAOTServiceHandler(&eventbus.NoOpEventBus{})
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestCreateAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "Fix the tests",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	if resp.Msg.AgentRun.Id == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		t.Errorf("expected PENDING phase, got %v", resp.Msg.AgentRun.Status.Phase)
	}
}

func TestCreateAgentRun_NilSpec(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{}))
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestGetAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create first
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "Test get",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Get it
	run, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: resp.Msg.AgentRun.Id}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if run.Msg.Spec.Prompt != "Test get" {
		t.Errorf("expected prompt 'Test get', got %q", run.Msg.Spec.Prompt)
	}
}

func TestGetAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: "nonexistent"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestListAgentRuns(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create two runs
	for _, prompt := range []string{"task 1", "task 2"} {
		if _, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				RepoUrl: "https://github.com/example/repo.git",
				Prompt:  prompt,
			},
		})); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestListAgentRuns_WithLimit(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		if _, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				RepoUrl: "https://github.com/example/repo.git",
				Prompt:  "task",
			},
		})); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 2}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs with limit, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestCancelAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "cancel me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	cancelResp, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: resp.Msg.AgentRun.Id}))
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	if cancelResp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
		t.Errorf("expected CANCELLED, got %v", cancelResp.Msg.AgentRun.Status.Phase)
	}
}

func TestCancelAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: "nonexistent"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestSendHumanInput_NotWaiting(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "not waiting",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	_, err = client.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: resp.Msg.AgentRun.Id,
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error: agent not waiting for input")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

func TestSendHumanInput_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: "nonexistent",
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

// startTestServerWithBus returns a client, the event bus, and a cleanup function.
func startTestServerWithBus(t *testing.T) (apiv1connect.AOTServiceClient, *eventbus.ChannelBus, *AOTServiceHandler, func()) {
	t.Helper()

	bus := eventbus.NewChannelBus()
	svc := NewAOTServiceHandler(bus)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return client, bus, svc, srv.Close
}

func TestWatchAgentRun_InitialState(t *testing.T) {
	client, _, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "watch me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: resp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun: %v", err)
	}

	// Should receive initial state
	if !stream.Receive() {
		t.Fatalf("expected initial event, got error: %v", stream.Err())
	}
	event := stream.Msg()
	if event.AgentRunId != resp.Msg.AgentRun.Id {
		t.Errorf("expected run ID %s, got %s", resp.Msg.AgentRun.Id, event.AgentRunId)
	}
	if event.Type != apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED {
		t.Errorf("expected PHASE_CHANGED, got %v", event.Type)
	}
}

func TestWatchAgentRun_EventStreaming(t *testing.T) {
	client, bus, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "stream events",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: resp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun: %v", err)
	}

	// Receive initial state
	if !stream.Receive() {
		t.Fatalf("expected initial event: %v", stream.Err())
	}

	// Publish an event via the bus
	go func() {
		time.Sleep(100 * time.Millisecond)
		bus.Publish(resp.Msg.AgentRun.Id, &apiv1.AgentRunEvent{
			AgentRunId: resp.Msg.AgentRun.Id,
			Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
			Payload:    "Running",
		})
		time.Sleep(100 * time.Millisecond)
		bus.Publish(resp.Msg.AgentRun.Id, &apiv1.AgentRunEvent{
			AgentRunId: resp.Msg.AgentRun.Id,
			Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED,
			Payload:    "Succeeded",
		})
	}()

	// Should receive the phase change event
	if !stream.Receive() {
		t.Fatalf("expected phase change event: %v", stream.Err())
	}
	if stream.Msg().Payload != "Running" {
		t.Errorf("expected Running payload, got %s", stream.Msg().Payload)
	}

	// Should receive the completion event
	if !stream.Receive() {
		t.Fatalf("expected completion event: %v", stream.Err())
	}
	if stream.Msg().Type != apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED {
		t.Errorf("expected COMPLETED, got %v", stream.Msg().Type)
	}
}

func TestWatchAgentRun_NotFound(t *testing.T) {
	client, _, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	ctx := context.Background()
	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: "nonexistent",
	}))
	if err != nil {
		// Some Connect implementations return error immediately
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
		}
		return
	}
	// For server-streaming, error comes on first Receive()
	if stream.Receive() {
		t.Fatal("expected no messages for nonexistent run")
	}
	if stream.Err() == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(stream.Err()) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(stream.Err()))
	}
}
