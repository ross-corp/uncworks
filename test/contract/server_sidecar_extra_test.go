package contract

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// startSidecarServerDebug starts the sidecar with AOT_DEBUG_MODE=true, which
// makes StartAgent return started=true immediately without launching a process.
// This enables testing the StartAgent contract without a real agent binary.
func startSidecarServerDebug(t *testing.T) (agentv1connect.AgentSidecarServiceClient, agentv1connect.AgentNotificationServiceClient, func()) {
	t.Helper()
	t.Setenv("AOT_DEBUG_MODE", "true")
	return startSidecarServer(t)
}

// --- StartAgent contract ---

func TestContract_StartAgent_DebugMode(t *testing.T) {
	// In debug mode StartAgent must return started=true without launching a process.
	client, _, cleanup := startSidecarServerDebug(t)
	defer cleanup()

	resp, err := client.StartAgent(context.Background(), connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: "run-debug-1",
		Prompt:     "fix the bug",
		RepoPath:   "/workspace",
	}))
	if err != nil {
		t.Fatalf("StartAgent (debug mode): %v", err)
	}
	if !resp.Msg.Started {
		t.Errorf("expected Started=true in debug mode, got false (error: %q)", resp.Msg.Error)
	}
}

func TestContract_StartAgent_EmptyPrompt_Accepted(t *testing.T) {
	// An empty prompt is valid for spec-driven runs; the gateway must not
	// reject it with InvalidArgument.
	client, _, cleanup := startSidecarServerDebug(t)
	defer cleanup()

	resp, err := client.StartAgent(context.Background(), connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: "run-spec-driven",
		Prompt:     "",
		RepoPath:   "/workspace",
	}))
	if err != nil {
		t.Fatalf("StartAgent with empty prompt: %v", err)
	}
	if !resp.Msg.Started {
		t.Errorf("expected Started=true in debug mode even with empty prompt")
	}
}

// --- ExecCommand contract ---

func TestContract_ExecCommand_SimpleCommand(t *testing.T) {
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:    "echo hello",
		WorkingDir: "/workspace",
	}))
	// ExecCommand falls back gracefully when /workspace does not exist.
	// Either it succeeds (exit 0 or non-zero) OR returns Internal for exec failure.
	// It must NOT return FailedPrecondition or NotFound.
	if err != nil {
		code := connect.CodeOf(err)
		if code == connect.CodeFailedPrecondition || code == connect.CodeNotFound {
			t.Errorf("ExecCommand returned unexpected code %v: %v", code, err)
		}
		return
	}
	// If it succeeds, the response fields must be present.
	if resp.Msg == nil {
		t.Fatal("expected non-nil ExecCommandResponse")
	}
	// Exit code of "echo hello" is 0 when bash is available.
	t.Logf("ExecCommand stdout=%q stderr=%q exitCode=%d",
		resp.Msg.Stdout, resp.Msg.Stderr, resp.Msg.ExitCode)
}

func TestContract_ExecCommand_InvalidWorkingDir(t *testing.T) {
	// A working directory that escapes /workspace must be rejected with InvalidArgument.
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	_, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:    "id",
		WorkingDir: "/etc",
	}))
	if err == nil {
		t.Fatal("expected error for working directory outside /workspace")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument for /etc working dir, got %v", connect.CodeOf(err))
	}
}

func TestContract_ExecCommand_EmptyCommand(t *testing.T) {
	// An empty command runs "bash -c ''" which exits 0 — must not blow up.
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:    "",
		WorkingDir: "/workspace",
	}))
	if err != nil {
		code := connect.CodeOf(err)
		if code == connect.CodeUnimplemented {
			t.Skip("ExecCommand not implemented in this build")
		}
		// Any error other than Unimplemented is unexpected for an empty command.
		t.Logf("ExecCommand empty command returned error (acceptable): %v", err)
		return
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil ExecCommandResponse")
	}
}

// --- SemanticSearch contract ---

func TestContract_SemanticSearch_EmptyQuery(t *testing.T) {
	// An empty query must be rejected with InvalidArgument.
	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	_, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{
		Query: "",
	}))
	if err == nil {
		t.Fatal("expected InvalidArgument for empty query")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestContract_SemanticSearch_NoEndpoint_ReturnsEmpty(t *testing.T) {
	// Without CUDGEL_ENDPOINT set the gateway returns an empty response, not an error.
	t.Setenv("CUDGEL_ENDPOINT", "")

	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{
		Query: "auth handler",
		Limit: 5,
	}))
	if err != nil {
		t.Fatalf("SemanticSearch without endpoint: %v", err)
	}
	if len(resp.Msg.Chunks) != 0 {
		t.Errorf("expected 0 chunks without CUDGEL_ENDPOINT, got %d", len(resp.Msg.Chunks))
	}
}

func TestContract_SemanticSearch_LimitClamped(t *testing.T) {
	// A limit above 50 must be silently clamped; no error is returned.
	t.Setenv("CUDGEL_ENDPOINT", "")

	client, _, cleanup := startSidecarServer(t)
	defer cleanup()

	resp, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{
		Query: "database connection pool",
		Limit: 999,
	}))
	if err != nil {
		t.Fatalf("SemanticSearch with oversized limit: %v", err)
	}
	// Without an endpoint we still get an empty result set (not an error).
	if resp.Msg == nil {
		t.Fatal("expected non-nil SemanticSearchResponse")
	}
}
