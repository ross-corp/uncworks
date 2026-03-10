package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
	"github.com/uncworks/aot/internal/sidecar"
)

func startSidecarServer(t *testing.T) (agentv1connect.AgentSidecarServiceClient, agentv1connect.AgentNotificationServiceClient, func()) {
	t.Helper()

	gw := sidecar.NewGateway(0)
	mux := http.NewServeMux()

	path, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle(path, handler)

	nPath, nHandler := agentv1connect.NewAgentNotificationServiceHandler(gw)
	mux.Handle(nPath, nHandler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	notifClient := agentv1connect.NewAgentNotificationServiceClient(srv.Client(), srv.URL)
	return sidecarClient, notifClient, srv.Close
}

// --- GetStatus contract ---

func TestContract_GetStatus_NoProcess(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&agentv1.GetStatusRequest{
		AgentRunId: "test",
	}))
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if resp.Msg.State != agentv1.AgentProcessState_AGENT_PROCESS_STATE_UNSPECIFIED {
		t.Errorf("expected UNSPECIFIED state, got %v", resp.Msg.State)
	}
}

// --- StopAgent contract ---

func TestContract_StopAgent_NoProcess(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.StopAgent(context.Background(), connect.NewRequest(&agentv1.StopAgentRequest{
		AgentRunId: "test",
	}))
	if err != nil {
		t.Fatalf("StopAgent: %v", err)
	}
	if !resp.Msg.Stopped {
		t.Error("expected stopped=true when no process running")
	}
}

func TestContract_StopAgent_Force(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.StopAgent(context.Background(), connect.NewRequest(&agentv1.StopAgentRequest{
		AgentRunId: "test",
		Force:      true,
	}))
	if err != nil {
		t.Fatalf("StopAgent force: %v", err)
	}
	if !resp.Msg.Stopped {
		t.Error("expected stopped=true")
	}
}

// --- SendInput contract ---

func TestContract_SendInput_NoProcess(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	_, err := client.SendInput(context.Background(), connect.NewRequest(&agentv1.SendInputRequest{
		AgentRunId: "test",
		Data:       []byte("hello"),
	}))
	if err == nil {
		t.Fatal("expected error when no process running")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

// --- StreamOutput contract ---

func TestContract_StreamOutput_NoProcess(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	stream, err := client.StreamOutput(context.Background(), connect.NewRequest(&agentv1.StreamOutputRequest{
		AgentRunId: "test",
	}))
	if err != nil {
		// Error returned immediately
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("expected FailedPrecondition, got %v", connect.CodeOf(err))
		}
		return
	}
	if stream.Receive() {
		t.Fatal("expected no messages when no process running")
	}
	if stream.Err() == nil {
		t.Fatal("expected error from stream")
	}
}

// --- NotifyEvent contract ---

func TestContract_NotifyEvent_Unimplemented(t *testing.T) {
	_, notifClient, cleanup := startSidecarServer(t)
	defer cleanup()

	// Gateway embeds UnimplementedAgentNotificationServiceHandler
	// so NotifyEvent returns Unimplemented
	_, err := notifClient.NotifyEvent(context.Background(), connect.NewRequest(&agentv1.NotifyEventRequest{
		AgentRunId: "test",
		EventType:  agentv1.EventType_EVENT_TYPE_STARTED,
		Payload:    "test payload",
	}))
	if err == nil {
		t.Fatal("expected error for unimplemented NotifyEvent")
	}
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("expected Unimplemented, got %v", connect.CodeOf(err))
	}
}
