package sidecar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

func startTestGateway(t *testing.T) (agentv1connect.AgentSidecarServiceClient, func()) {
	t.Helper()

	gw := NewGateway(0)
	mux := http.NewServeMux()
	path, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestGetStatus_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&agentv1.GetStatusRequest{AgentRunId: "test"}))
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if resp.Msg.State != agentv1.AgentProcessState_AGENT_PROCESS_STATE_UNSPECIFIED {
		t.Errorf("expected UNSPECIFIED state, got %v", resp.Msg.State)
	}
}

func TestStopAgent_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	resp, err := client.StopAgent(context.Background(), connect.NewRequest(&agentv1.StopAgentRequest{AgentRunId: "test"}))
	if err != nil {
		t.Fatalf("StopAgent: %v", err)
	}
	if !resp.Msg.Stopped {
		t.Error("expected stopped=true")
	}
}

func TestSendInput_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	_, err := client.SendInput(context.Background(), connect.NewRequest(&agentv1.SendInputRequest{
		AgentRunId: "test",
		Data:       []byte("hello"),
	}))
	if err == nil {
		t.Fatal("expected error when no process running")
	}
}

func TestStreamOutput_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	stream, err := client.StreamOutput(context.Background(), connect.NewRequest(&agentv1.StreamOutputRequest{AgentRunId: "test"}))
	if err != nil {
		t.Fatalf("StreamOutput: %v", err)
	}
	if stream.Receive() {
		t.Fatal("expected no messages when no process running")
	}
	if stream.Err() == nil {
		t.Fatal("expected error from stream")
	}
}
