package sidecar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// TestExecSafeEnv_StripsSensitiveKeys verifies that execSafeEnv removes known
// secret variables while preserving safe ones like PATH.
func TestExecSafeEnv_StripsSensitiveKeys(t *testing.T) {
	sensitive := []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"LITELLM_MASTER_KEY",
		"GITHUB_TOKEN",
		"AOT_API_KEY",
	}
	for _, key := range sensitive {
		t.Setenv(key, "secret-value")
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	env := execSafeEnv()
	kvMap := make(map[string]string, len(env))
	for _, kv := range env {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		kvMap[kv[:idx]] = kv[idx+1:]
	}

	for _, key := range sensitive {
		if _, found := kvMap[key]; found {
			t.Errorf("sensitive key %q should be stripped but was present", key)
		}
	}
	if kvMap["PATH"] != "/usr/bin:/bin" {
		t.Errorf("PATH should be preserved, got %q", kvMap["PATH"])
	}
}

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

func TestStageSystemPrompt(t *testing.T) {
	tests := []struct {
		stage        string
		wantNonEmpty bool
		wantContains []string // at least one must match
	}{
		{
			stage:        "plan",
			wantNonEmpty: true,
			wantContains: []string{"openspec", "OpenSpec"},
		},
		{
			stage:        "execute",
			wantNonEmpty: true,
			wantContains: []string{"implement", "Implement", "apply", "Apply"},
		},
		{
			stage:        "verify",
			wantNonEmpty: true,
			wantContains: []string{"evaluation", "Evaluation", "verify", "Verify", "verification", "Verification"},
		},
		{
			// Empty/unset stage uses the default single-agent prompt so the
			// agent is oriented to its working directory even without a pipeline stage.
			stage:        "",
			wantNonEmpty: true,
			wantContains: []string{"coding agent", "working directory"},
		},
		{
			// Unrecognised stage also falls through to the default prompt.
			stage:        "unknown",
			wantNonEmpty: true,
			wantContains: []string{"coding agent", "working directory"},
		},
	}

	for _, tt := range tests {
		name := tt.stage
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			got := stageSystemPrompt(tt.stage)
			if tt.wantNonEmpty && got == "" {
				t.Fatalf("stageSystemPrompt(%q) returned empty string, want non-empty", tt.stage)
			}
			if !tt.wantNonEmpty && got != "" {
				t.Fatalf("stageSystemPrompt(%q) returned %q, want empty string", tt.stage, got)
			}
			if len(tt.wantContains) > 0 {
				found := false
				lower := strings.ToLower(got)
				for _, substr := range tt.wantContains {
					if strings.Contains(lower, strings.ToLower(substr)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("stageSystemPrompt(%q) = %q\nwant it to contain one of %v", tt.stage, got, tt.wantContains)
				}
			}
		})
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

func TestExecCommand_Success(t *testing.T) {
	gw := &Gateway{}
	mux := http.NewServeMux()
	_, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle("/", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "echo hello",
		TimeoutSeconds: 5,
	}))
	if err != nil {
		t.Fatalf("ExecCommand: %v", err)
	}
	if resp.Msg.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", resp.Msg.ExitCode)
	}
	if !strings.Contains(resp.Msg.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", resp.Msg.Stdout)
	}
}

func TestExecCommand_Failure(t *testing.T) {
	gw := &Gateway{}
	mux := http.NewServeMux()
	_, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle("/", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "exit 42",
		TimeoutSeconds: 5,
	}))
	if err != nil {
		t.Fatalf("ExecCommand: %v", err)
	}
	if resp.Msg.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", resp.Msg.ExitCode)
	}
}

func TestExecCommand_Timeout(t *testing.T) {
	gw := &Gateway{}
	mux := http.NewServeMux()
	_, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle("/", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "sleep 30",
		TimeoutSeconds: 1,
	}))
	if err != nil {
		t.Fatalf("ExecCommand: %v", err)
	}
	// Should have been killed — non-zero exit code
	if resp.Msg.ExitCode == 0 {
		t.Error("expected non-zero exit code for timed-out command")
	}
}

func TestExecCommand_CaptureStderr(t *testing.T) {
	gw := &Gateway{}
	mux := http.NewServeMux()
	_, handler := agentv1connect.NewAgentSidecarServiceHandler(gw)
	mux.Handle("/", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := agentv1connect.NewAgentSidecarServiceClient(srv.Client(), srv.URL)
	resp, err := client.ExecCommand(context.Background(), connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "echo error >&2 && exit 1",
		TimeoutSeconds: 5,
	}))
	if err != nil {
		t.Fatalf("ExecCommand: %v", err)
	}
	if resp.Msg.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", resp.Msg.ExitCode)
	}
	if !strings.Contains(resp.Msg.Stderr, "error") {
		t.Errorf("expected stderr to contain 'error', got %q", resp.Msg.Stderr)
	}
}

func TestExtractToolCallSignature(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "tool_use event returns name:inputlen",
			line: `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"ls -la"}}]}}`,
			want: "bash:20",
		},
		{
			name: "non-message_end event returns empty",
			line: `{"type":"message_start","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"ls"}}]}}`,
			want: "",
		},
		{
			name: "text content returns empty",
			line: `{"type":"message_end","message":{"content":[{"type":"text","text":"hello world"}]}}`,
			want: "",
		},
		{
			name: "empty line returns empty",
			line: "",
			want: "",
		},
		{
			name: "invalid JSON returns empty",
			line: "not json at all",
			want: "",
		},
		{
			name: "message_end with no message returns empty",
			line: `{"type":"message_end"}`,
			want: "",
		},
		{
			name: "message_end with null content returns empty",
			line: `{"type":"message_end","message":{"content":null}}`,
			want: "",
		},
		{
			name: "different tool name produces different signature",
			line: `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"write_file","input":{"path":"/tmp/a.txt","content":"hello"}}]}}`,
			want: "write_file:39",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolCallSignature(tt.line)
			if got != tt.want {
				t.Errorf("extractToolCallSignature() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractToolCallSignature_IdenticalCallsSameSig(t *testing.T) {
	line1 := `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"cat /etc/hosts"}}]}}`
	line2 := `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"cat /etc/hosts"}}]}}`

	sig1 := extractToolCallSignature(line1)
	sig2 := extractToolCallSignature(line2)

	if sig1 == "" {
		t.Fatal("expected non-empty signature for tool_use event")
	}
	if sig1 != sig2 {
		t.Errorf("identical tool calls produced different signatures: %q vs %q", sig1, sig2)
	}
}

func TestExtractToolCallSignature_DifferentCallsDifferentSig(t *testing.T) {
	line1 := `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"ls"}}]}}`
	line2 := `{"type":"message_end","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"pwd"}}]}}`

	sig1 := extractToolCallSignature(line1)
	sig2 := extractToolCallSignature(line2)

	if sig1 == "" || sig2 == "" {
		t.Fatal("expected non-empty signatures")
	}
	if sig1 == sig2 {
		t.Errorf("different tool calls produced same signature: %q", sig1)
	}
}

// --- resolveWorkDir regression tests ---

// --- extractToolFromEvent tests ---

func TestExtractToolFromEvent_ToolCallStartFormat(t *testing.T) {
	// pi's toolcall_start format: partial.content[] has toolCall blocks
	partial := `{"content":[{"type":"toolCall","name":"bash","id":"tc-1","arguments":{"command":"ls -la"}}]}`
	ame := &piAssistantEvent{
		Partial: json.RawMessage(partial),
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "bash" {
		t.Errorf("name = %q, want %q", name, "bash")
	}
	if inputJSON == "" {
		t.Fatal("inputJSON is empty, expected non-empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.Fatalf("inputJSON is not valid JSON: %v", err)
	}
	if args["command"] != "ls -la" {
		t.Errorf("args[command] = %v, want %q", args["command"], "ls -la")
	}
}

func TestExtractToolFromEvent_LegacyToolUseFormat(t *testing.T) {
	// Legacy format: ame.Tool contains tool info
	tool := `{"name":"write_file","input":{"path":"/tmp/test.txt","content":"hello"}}`
	ame := &piAssistantEvent{
		Tool: json.RawMessage(tool),
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "write_file" {
		t.Errorf("name = %q, want %q", name, "write_file")
	}
	if inputJSON == "" {
		t.Fatal("inputJSON is empty, expected non-empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.Fatalf("inputJSON is not valid JSON: %v", err)
	}
	if args["path"] != "/tmp/test.txt" {
		t.Errorf("args[path] = %v, want %q", args["path"], "/tmp/test.txt")
	}
}

func TestExtractToolFromEvent_NoToolInfo(t *testing.T) {
	// Event with no tool information should return empty strings
	ame := &piAssistantEvent{
		Type:  "assistant_message",
		Delta: "some text content",
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if inputJSON != "" {
		t.Errorf("inputJSON = %q, want empty", inputJSON)
	}
}

func TestExtractToolFromEvent_PartialWithNoToolCall(t *testing.T) {
	// Partial content with non-toolCall blocks should return empty
	partial := `{"content":[{"type":"text","text":"hello world"}]}`
	ame := &piAssistantEvent{
		Partial: json.RawMessage(partial),
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "" {
		t.Errorf("name = %q, want empty for text block", name)
	}
	if inputJSON != "" {
		t.Errorf("inputJSON = %q, want empty for text block", inputJSON)
	}
}

func TestExtractToolFromEvent_PartialTakesPrecedenceOverLegacy(t *testing.T) {
	// When both formats are present, partial (toolcall_start) takes precedence
	partial := `{"content":[{"type":"toolCall","name":"bash","arguments":{"command":"pwd"}}]}`
	tool := `{"name":"write_file","input":{"path":"/tmp/a"}}`
	ame := &piAssistantEvent{
		Partial: json.RawMessage(partial),
		Tool:    json.RawMessage(tool),
	}

	name, _ := extractToolFromEvent(ame)
	if name != "bash" {
		t.Errorf("name = %q, want %q (partial should take precedence over legacy)", name, "bash")
	}
}

func TestExtractToolFromEvent_ToolCallNoArguments(t *testing.T) {
	// toolCall with no arguments should return name but empty inputJSON
	partial := `{"content":[{"type":"toolCall","name":"get_status","id":"tc-2"}]}`
	ame := &piAssistantEvent{
		Partial: json.RawMessage(partial),
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "get_status" {
		t.Errorf("name = %q, want %q", name, "get_status")
	}
	if inputJSON != "" {
		t.Errorf("inputJSON = %q, want empty when no arguments", inputJSON)
	}
}

func TestExtractToolFromEvent_LegacyToolNoInput(t *testing.T) {
	// Legacy tool with no input should return name but empty inputJSON
	tool := `{"name":"list_files"}`
	ame := &piAssistantEvent{
		Tool: json.RawMessage(tool),
	}

	name, inputJSON := extractToolFromEvent(ame)
	if name != "list_files" {
		t.Errorf("name = %q, want %q", name, "list_files")
	}
	if inputJSON != "" {
		t.Errorf("inputJSON = %q, want empty when no input", inputJSON)
	}
}

// --- Task 2.3: Span naming convention (spanPrefix) ---

func TestSpanPrefix_PlanReturnsManage(t *testing.T) {
	setCurrentStage("plan", "/workspace")
	got := spanPrefix()
	if got != "manage" {
		t.Errorf("spanPrefix() for plan = %q, want %q", got, "manage")
	}
}

func TestSpanPrefix_VerifyReturnsManage(t *testing.T) {
	setCurrentStage("verify", "/workspace")
	got := spanPrefix()
	if got != "manage" {
		t.Errorf("spanPrefix() for verify = %q, want %q", got, "manage")
	}
}

func TestSpanPrefix_ExecuteReturnsImplement(t *testing.T) {
	setCurrentStage("execute", "/workspace")
	got := spanPrefix()
	if got != "implement" {
		t.Errorf("spanPrefix() for execute = %q, want %q", got, "implement")
	}
}

func TestSpanPrefix_SingleReturnsImplement(t *testing.T) {
	setCurrentStage("", "/workspace")
	got := spanPrefix()
	if got != "implement" {
		t.Errorf("spanPrefix() for empty/single = %q, want %q", got, "implement")
	}
}

func TestSpanPrefix_ToolSpanName(t *testing.T) {
	// Verify that the convention {prefix}.{toolName} produces correct span names.
	setCurrentStage("execute", "/workspace")
	prefix := spanPrefix()
	spanName := prefix + ".bash"
	if spanName != "implement.bash" {
		t.Errorf("span name = %q, want %q", spanName, "implement.bash")
	}
	// Verify it does NOT produce generic name
	if spanName == "implement.tool" {
		t.Error("span name should be implement.bash, not implement.tool")
	}

	setCurrentStage("plan", "/workspace")
	prefix = spanPrefix()
	spanName = prefix + ".write"
	if spanName != "manage.write" {
		t.Errorf("span name = %q, want %q", spanName, "manage.write")
	}
	if spanName == "manage.tool" {
		t.Error("span name should be manage.write, not manage.tool")
	}
}

// --- resolveWorkDir regression tests ---

func TestResolveWorkDirAt_RepoGitDetected(t *testing.T) {
	// When /workspace/<repo>/.git exists, returns /workspace/<repo>.
	base := t.TempDir()
	repoDir := filepath.Join(base, "myrepo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt = %q, want %q", got, repoDir)
	}
}

func TestResolveWorkDirAt_RootClone(t *testing.T) {
	// When /workspace/.git exists (root clone), returns /workspace.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt = %q, want %q (root clone)", got, base)
	}
}

func TestResolveWorkDirAt_SkipsBare(t *testing.T) {
	// .bare directories must be skipped even if they contain .git.
	base := t.TempDir()

	// .bare/repo/.git — should be skipped
	if err := os.MkdirAll(filepath.Join(base, ".bare", "repo", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No actual repo dir with .git

	got := ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt = %q, want %q (.bare should be skipped)", got, base)
	}
}

func TestResolveWorkDirAt_SkipsAot(t *testing.T) {
	// .aot directories must be skipped.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".aot", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt = %q, want %q (.aot should be skipped)", got, base)
	}
}

func TestResolveWorkDirAt_SkipsDevcontainer(t *testing.T) {
	// .devcontainer directories must be skipped.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".devcontainer", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt = %q, want %q (.devcontainer should be skipped)", got, base)
	}
}

func TestResolveWorkDirAt_SkipsOpenspecAndSpec(t *testing.T) {
	// "openspec" and "spec" directories must be skipped.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "openspec", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "spec", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt = %q, want %q (openspec/spec should be skipped)", got, base)
	}
}

func TestResolveWorkDirAt_DetectsGitFile(t *testing.T) {
	// Worktrees create a .git *file* (not directory). ResolveWorkDirAt should
	// detect it via os.Stat (which works for both files and directories).
	base := t.TempDir()
	repoDir := filepath.Join(base, "myrepo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// .git as a file (worktree pointer)
	if err := os.WriteFile(filepath.Join(repoDir, ".git"), []byte("gitdir: /workspace/.bare/myrepo"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt = %q, want %q (should detect .git file)", got, repoDir)
	}
}

// --- Compaction event detection tests ---

func TestParsePiDcpPruned_ValidLine(t *testing.T) {
	pruned, total := parsePiDcpPruned("[pi-dcp] Pruned 5 / 20 messages")
	if pruned != 5 {
		t.Errorf("pruned = %d, want 5", pruned)
	}
	if total != 20 {
		t.Errorf("total = %d, want 20", total)
	}
}

func TestParsePiDcpPruned_LargeNumbers(t *testing.T) {
	pruned, total := parsePiDcpPruned("[pi-dcp] Pruned 150 / 300 messages")
	if pruned != 150 {
		t.Errorf("pruned = %d, want 150", pruned)
	}
	if total != 300 {
		t.Errorf("total = %d, want 300", total)
	}
}

func TestParsePiDcpPruned_NoMatch(t *testing.T) {
	pruned, total := parsePiDcpPruned("some other log line")
	if pruned != 0 || total != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", pruned, total)
	}
}

func TestParsePiDcpPruned_EmptyLine(t *testing.T) {
	pruned, total := parsePiDcpPruned("")
	if pruned != 0 || total != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", pruned, total)
	}
}

func TestMaybeCaptureStreamEvent_CompactionCreatesSpan(t *testing.T) {
	// Set up trace dir in a temp location — we override the constant via a temp dir that
	// matches the const path only when running in the right environment.
	// Since traceSpansPath is a constant pointing to /workspace/.aot/traces/spans.jsonl,
	// in CI/test we create that dir structure so appendTraceSpan can write.
	dir := t.TempDir()
	spansDir := filepath.Join(dir, "traces")
	if err := os.MkdirAll(spansDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spansFile := filepath.Join(spansDir, "spans.jsonl")

	// We can't redirect the const in a unit test, so instead test the event parsing
	// logic by verifying the compaction event type is handled (no panic, correct flow).
	// The actual span writing is integration-level.

	evt := &piEvent{
		Type:    "compaction",
		Message: json.RawMessage(`{"tokensBefore":10000,"tokensAfter":5000}`),
	}

	// Set up state
	setCurrentStage("execute", "/workspace")
	setCurrentParentSpan("parent-123", "trace-456")

	// Call the function — it will fail to write to /workspace/.aot/traces/spans.jsonl
	// in test, but should not panic.
	maybeCaptureStreamEvent(evt, `{"type":"compaction","tokensBefore":10000,"tokensAfter":5000}`)

	// Also test context_compaction variant
	evt2 := &piEvent{
		Type:    "context_compaction",
		Message: json.RawMessage(`{"tokensBefore":8000,"tokensAfter":3000}`),
	}
	maybeCaptureStreamEvent(evt2, `{"type":"context_compaction","tokensBefore":8000,"tokensAfter":3000}`)

	_ = spansFile // used if we ever redirect
}

func TestMaybeCaptureStreamEvent_CompactionMissingFields(t *testing.T) {
	// Compaction event with no token counts — should still not panic
	setCurrentStage("plan", "/workspace")
	setCurrentParentSpan("p1", "t1")

	evt := &piEvent{
		Type: "compaction",
	}
	// Should not panic with nil Message
	maybeCaptureStreamEvent(evt, `{"type":"compaction"}`)

	// With empty message object
	evt2 := &piEvent{
		Type:    "compaction",
		Message: json.RawMessage(`{}`),
	}
	maybeCaptureStreamEvent(evt2, `{"type":"compaction"}`)
}

func TestMaybeCaptureStreamEvent_CompactionEmptyMessage(t *testing.T) {
	// Compaction event with empty JSON object — graceful degradation
	setCurrentStage("", "/workspace")
	setCurrentParentSpan("", "")

	evt := &piEvent{
		Type:    "compaction",
		Message: json.RawMessage(`{}`),
	}
	// Should not panic, should create a span with zero token counts
	maybeCaptureStreamEvent(evt, `{"type":"compaction"}`)
}

func TestResolveWorkDirAt_BareSkippedRealRepoFound(t *testing.T) {
	// Full debug pod layout: .bare exists but is skipped, real worktree is found.
	base := t.TempDir()

	// .bare/repo — should be skipped
	if err := os.MkdirAll(filepath.Join(base, ".bare", "repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	// .aot — should be skipped
	if err := os.MkdirAll(filepath.Join(base, ".aot"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The actual repo worktree
	repoDir := filepath.Join(base, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".git"), []byte("gitdir: "+filepath.Join(base, ".bare", "repo")), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt = %q, want %q", got, repoDir)
	}
}

// --- SemanticSearch tests ---

// startTestGatewayWithCudgel creates a fake cudgel HTTP server, sets CUDGEL_ENDPOINT,
// and returns a gateway client and cleanup func.
func startTestGatewayWithCudgel(t *testing.T, cudgelHandler http.HandlerFunc) (agentv1connect.AgentSidecarServiceClient, func()) {
	t.Helper()

	cudgelSrv := httptest.NewServer(cudgelHandler)
	t.Setenv("CUDGEL_ENDPOINT", cudgelSrv.URL)

	client, gwCleanup := startTestGateway(t)
	return client, func() {
		gwCleanup()
		cudgelSrv.Close()
	}
}

func TestSemanticSearch_EmptyQuery(t *testing.T) {
	client, cleanup := startTestGateway(t)
	defer cleanup()

	_, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{Query: ""}))
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
}

func TestSemanticSearch_NoEndpoint(t *testing.T) {
	t.Setenv("CUDGEL_ENDPOINT", "")
	client, cleanup := startTestGateway(t)
	defer cleanup()

	resp, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{Query: "auth middleware", Limit: 5}))
	if err != nil {
		t.Fatalf("expected empty response, got error: %v", err)
	}
	if len(resp.Msg.Chunks) != 0 {
		t.Errorf("expected no chunks, got %d", len(resp.Msg.Chunks))
	}
}

func TestSemanticSearch_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"Foo","kind":"function","file":"foo.go","line":1,"snippet":"func Foo()","score":0.9}]`))
	})
	client, cleanup := startTestGatewayWithCudgel(t, handler)
	defer cleanup()

	resp, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{Query: "auth", Limit: 5}))
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	if len(resp.Msg.Chunks) != 1 || resp.Msg.Chunks[0].Name != "Foo" {
		t.Errorf("unexpected chunks: %+v", resp.Msg.Chunks)
	}
}

func TestSemanticSearch_CudgelUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	})
	client, cleanup := startTestGatewayWithCudgel(t, handler)
	defer cleanup()

	// Should return empty response, not an error
	resp, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{Query: "auth", Limit: 5}))
	if err != nil {
		t.Fatalf("expected empty response on cudgel failure, got error: %v", err)
	}
	if len(resp.Msg.Chunks) != 0 {
		t.Errorf("expected no chunks on failure, got %d", len(resp.Msg.Chunks))
	}
}

func TestSemanticSearch_LimitClamping(t *testing.T) {
	var gotLimit int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Limit int `json:"limit"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotLimit = body.Limit
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	client, cleanup := startTestGatewayWithCudgel(t, handler)
	defer cleanup()

	_, err := client.SemanticSearch(context.Background(), connect.NewRequest(&agentv1.SemanticSearchRequest{Query: "auth", Limit: 100}))
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	if gotLimit != 50 {
		t.Errorf("expected clamped limit 50, got %d", gotLimit)
	}
}
