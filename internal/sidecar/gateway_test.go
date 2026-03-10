package sidecar

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
)

func startTestGateway(t *testing.T) (agentv1.AgentSidecarServiceClient, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	gw := NewGateway(0)
	gw.grpcServer = grpc.NewServer()
	agentv1.RegisterAgentSidecarServiceServer(gw.grpcServer, gw)

	go gw.grpcServer.Serve(lis)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	client := agentv1.NewAgentSidecarServiceClient(conn)
	return client, func() {
		conn.Close()
		gw.grpcServer.GracefulStop()
	}
}

func TestGetStatus_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	resp, err := client.GetStatus(context.Background(), &agentv1.GetStatusRequest{AgentRunId: "test"})
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if resp.State != agentv1.AgentProcessState_AGENT_PROCESS_STATE_UNSPECIFIED {
		t.Errorf("expected UNSPECIFIED state, got %v", resp.State)
	}
}

func TestStopAgent_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	resp, err := client.StopAgent(context.Background(), &agentv1.StopAgentRequest{AgentRunId: "test"})
	if err != nil {
		t.Fatalf("StopAgent: %v", err)
	}
	if !resp.Stopped {
		t.Error("expected stopped=true")
	}
}

func TestSendInput_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	_, err := client.SendInput(context.Background(), &agentv1.SendInputRequest{
		AgentRunId: "test",
		Data:       []byte("hello"),
	})
	if err == nil {
		t.Fatal("expected error when no process running")
	}
}

func TestStreamOutput_NoProcess(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	stream, err := client.StreamOutput(context.Background(), &agentv1.StreamOutputRequest{AgentRunId: "test"})
	if err != nil {
		t.Fatalf("StreamOutput: %v", err)
	}
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error when no process running")
	}
}
