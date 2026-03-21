package integration

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/internal/sidecar"
)

// realisticJSONL is a realistic JSONL input that exercises the full agent event lifecycle:
// session, agent_start, turn_start, message_start/end (user), message_start/end (assistant
// with thinking), message_update (thinking_delta, text_delta), tool_execution, and agent_end.
var realisticJSONL = strings.Join([]string{
	// Session event
	`{"type":"session","timestamp":"2026-03-20T14:00:00Z"}`,
	// Agent starts
	`{"type":"agent_start"}`,
	// Turn starts
	`{"type":"turn_start"}`,
	// User message start
	`{"type":"message_start"}`,
	// User message end
	`{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"Fix the bug in parser.go"}],"timestamp":"2026-03-20T14:00:01Z"}}`,
	// Assistant message start (with thinking)
	`{"type":"message_start"}`,
	// Thinking deltas
	`{"type":"message_update","assistantMessageEvent":{"type":"thinking_delta","delta":"Let me analyze "}}`,
	`{"type":"message_update","assistantMessageEvent":{"type":"thinking_delta","delta":"the parser code "}}`,
	`{"type":"message_update","assistantMessageEvent":{"type":"thinking_delta","delta":"to find the bug."}}`,
	// Text delta
	`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"I'll read the parser file first."}}`,
	// Assistant message end with tool_use
	`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"I'll read the parser file first."},{"type":"tool_use","id":"tc_read1","name":"Read","input":{"file_path":"parser.go"}}],"timestamp":"2026-03-20T14:00:05Z","model":"claude-opus-4-20250514"}}`,
	// Tool execution
	`{"type":"tool_execution_start","toolName":"Read","toolCallId":"tc_read1","args":{"file_path":"parser.go"}}`,
	`{"type":"tool_execution_end","toolName":"Read","toolCallId":"tc_read1","isError":false,"result":{"content":"package main\n\nfunc Parse(s string) (int, error) {\n\treturn 0, nil\n}"}}`,
	// Second assistant message
	`{"type":"message_start"}`,
	// Assistant message end with text
	`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"The parser function returns 0 for all inputs. I'll fix it."}],"timestamp":"2026-03-20T14:00:10Z","model":"claude-opus-4-20250514"}}`,
	// Agent ends
	`{"type":"agent_end","messages":[{"role":"assistant","content":[{"type":"text","text":"The parser function returns 0 for all inputs. I'll fix it."}],"timestamp":"2026-03-20T14:00:10Z","model":"claude-opus-4-20250514"}]}`,
}, "\n")

// TestSidecarSpans_ConsistentCounts verifies that processing the same JSONL
// through both the ParseAgentJSONL log parser and the sidecar's
// ExtractToolCallSignature produces consistent results for tool detection.
func TestSidecarSpans_ConsistentCounts(t *testing.T) {
	// --- Parse with server.ParseAgentJSONL ---
	entries := server.ParseAgentJSONL(realisticJSONL)
	if len(entries) == 0 {
		t.Fatal("ParseAgentJSONL returned no entries")
	}

	// Count tool_call entries from the log parser.
	var toolCallCount int
	for _, e := range entries {
		if e.Type == "tool_call" {
			toolCallCount++
		}
	}

	t.Logf("ParseAgentJSONL: %d total entries, %d tool_call entries", len(entries), toolCallCount)

	if toolCallCount < 1 {
		t.Errorf("expected at least 1 tool_call from ParseAgentJSONL, got %d", toolCallCount)
	}

	// --- Parse events individually with sidecar.ExtractToolCallSignature ---
	lines := strings.Split(realisticJSONL, "\n")
	var sidecarToolCount int
	for _, line := range lines {
		if sig := sidecar.ExtractToolCallSignature(line); sig != "" {
			sidecarToolCount++
			t.Logf("ExtractToolCallSignature detected tool: %s", sig)
		}
	}

	// ExtractToolCallSignature only fires on message_end events with tool_use content blocks.
	// The JSONL has one message_end with a tool_use block (the Read tool call).
	if sidecarToolCount < 1 {
		t.Errorf("expected at least 1 tool signature from ExtractToolCallSignature, got %d", sidecarToolCount)
	}

	// --- Verify consistency ---
	// Both parsers should detect the same tool call events.
	// ParseAgentJSONL counts from tool_execution_start events.
	// ExtractToolCallSignature counts from message_end events with tool_use blocks.
	// Both should detect exactly 1 tool call (Read) in our test input.
	if toolCallCount != sidecarToolCount {
		t.Errorf("tool count mismatch: ParseAgentJSONL found %d tool_calls, ExtractToolCallSignature found %d tool signatures",
			toolCallCount, sidecarToolCount)
	}

	// --- Verify entry types cover all expected categories ---
	typeCounts := make(map[string]int)
	for _, e := range entries {
		typeCounts[e.Type]++
	}

	t.Logf("Entry type distribution: %v", typeCounts)

	if typeCounts["assistant"] < 1 {
		t.Errorf("expected at least 1 assistant entry, got %d", typeCounts["assistant"])
	}
	if typeCounts["tool_result"] < 1 {
		t.Errorf("expected at least 1 tool_result entry, got %d", typeCounts["tool_result"])
	}

	// --- Verify tool_call has correct tool name ---
	for _, e := range entries {
		if e.Type == "tool_call" {
			if e.ToolName != "Read" {
				t.Errorf("expected tool_call ToolName=Read, got %q", e.ToolName)
			}
			break
		}
	}

	// --- Verify tool_result has content ---
	for _, e := range entries {
		if e.Type == "tool_result" {
			if e.Content == "" {
				t.Error("expected non-empty tool_result content")
			}
			if e.ToolName != "Read" {
				t.Errorf("expected tool_result ToolName=Read, got %q", e.ToolName)
			}
			break
		}
	}
}

// TestSidecarSpans_EmptyInput verifies that both parsers handle empty input gracefully.
func TestSidecarSpans_EmptyInput(t *testing.T) {
	entries := server.ParseAgentJSONL("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}

	sig := sidecar.ExtractToolCallSignature("")
	if sig != "" {
		t.Errorf("expected empty signature for empty input, got %q", sig)
	}
}

// TestSidecarSpans_NoToolCalls verifies consistent zero counts when no tools are used.
func TestSidecarSpans_NoToolCalls(t *testing.T) {
	noToolJSONL := strings.Join([]string{
		`{"type":"session","timestamp":"2026-03-20T14:00:00Z"}`,
		`{"type":"agent_start"}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Hello!"}}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Hello!"}],"timestamp":"2026-03-20T14:00:01Z","model":"claude-opus-4-20250514"}}`,
		`{"type":"agent_end","messages":[]}`,
	}, "\n")

	entries := server.ParseAgentJSONL(noToolJSONL)
	var toolCalls int
	for _, e := range entries {
		if e.Type == "tool_call" {
			toolCalls++
		}
	}

	lines := strings.Split(noToolJSONL, "\n")
	var sidecarTools int
	for _, line := range lines {
		if sig := sidecar.ExtractToolCallSignature(line); sig != "" {
			sidecarTools++
		}
	}

	if toolCalls != 0 {
		t.Errorf("expected 0 tool_calls from ParseAgentJSONL (no tools), got %d", toolCalls)
	}
	if sidecarTools != 0 {
		t.Errorf("expected 0 tool signatures from ExtractToolCallSignature (no tools), got %d", sidecarTools)
	}
	if toolCalls != sidecarTools {
		t.Errorf("tool count mismatch in no-tool scenario: JSONL=%d, sidecar=%d", toolCalls, sidecarTools)
	}
}

// --- Task 4.1: appendTraceSpan + readSpansFile round-trip ---

// TestSidecarSpans_JSONLRoundTrip verifies that sidecar.TraceSpan written as JSONL
// can be deserialized by server.TraceSpan (the readSpansFile equivalent), preserving
// all fields including traceId, status, parentId, hasDiff, and nested diff.files.
func TestSidecarSpans_JSONLRoundTrip(t *testing.T) {
	// Create a temp file to write JSONL
	tmpFile, err := os.CreateTemp(t.TempDir(), "spans-*.jsonl")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write sidecar-format spans as JSONL (simulating appendTraceSpan behavior)
	spans := []sidecar.TraceSpan{
		{
			ID:        "span-stage-1",
			TraceID:   "trace-abc",
			Name:      "PLAN",
			Type:      "stage",
			StartTime: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 3, 20, 10, 5, 0, 0, time.UTC),
			Status:    "ok",
			Metadata:  map[string]interface{}{"stage": "plan"},
		},
		{
			ID:        "span-thought-1",
			TraceID:   "trace-abc",
			ParentID:  "span-stage-1",
			Name:      "manage.thought",
			Type:      "thought",
			StartTime: time.Date(2026, 3, 20, 10, 0, 1, 0, time.UTC),
			EndTime:   time.Date(2026, 3, 20, 10, 0, 5, 0, time.UTC),
			Status:    "ok",
			Metadata: map[string]interface{}{
				"gen_ai.usage.input_tokens":  float64(1500),
				"gen_ai.usage.output_tokens": float64(300),
			},
		},
		{
			ID:        "span-tool-1",
			TraceID:   "trace-abc",
			ParentID:  "span-stage-1",
			Name:      "implement.bash",
			Type:      "tool",
			StartTime: time.Date(2026, 3, 20, 10, 1, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 3, 20, 10, 1, 3, 0, time.UTC),
			HasDiff:   true,
			Diff: &sidecar.SpanDiff{
				Files: []sidecar.FileDiff{
					{Path: "main.go", Patch: "+fmt.Println(\"hello\")"},
					{Path: "util.go", Patch: "-old line\n+new line"},
				},
			},
			Metadata: map[string]interface{}{
				"toolInput": `{"command":"go build ./..."}`,
			},
		},
	}

	for _, span := range spans {
		data, err := json.Marshal(span)
		if err != nil {
			t.Fatalf("marshal span %s: %v", span.ID, err)
		}
		if _, err := tmpFile.Write(append(data, '\n')); err != nil {
			t.Fatalf("write span %s: %v", span.ID, err)
		}
	}
	_ = tmpFile.Close()

	// Read back using the same approach as server.readSpansFile
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("open spans file: %v", err)
	}
	defer func() { _ = file.Close() }()

	var decoded []server.TraceSpan
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var span server.TraceSpan
		if err := json.Unmarshal(line, &span); err != nil {
			t.Fatalf("unmarshal span: %v", err)
		}
		decoded = append(decoded, span)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(decoded) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(decoded))
	}

	// Verify stage span (span 0)
	stage := decoded[0]
	if stage.ID != "span-stage-1" {
		t.Errorf("stage.ID = %q, want %q", stage.ID, "span-stage-1")
	}
	if stage.TraceID != "trace-abc" {
		t.Errorf("stage.TraceID = %q, want %q", stage.TraceID, "trace-abc")
	}
	if stage.Status != "ok" {
		t.Errorf("stage.Status = %q, want %q", stage.Status, "ok")
	}
	if stage.Type != "stage" {
		t.Errorf("stage.Type = %q, want %q", stage.Type, "stage")
	}
	if stage.ParentID != "" {
		t.Errorf("stage.ParentID = %q, want empty", stage.ParentID)
	}

	// Verify thought span (span 1) — parentId links to stage
	thought := decoded[1]
	if thought.ParentID != "span-stage-1" {
		t.Errorf("thought.ParentID = %q, want %q", thought.ParentID, "span-stage-1")
	}
	if thought.TraceID != "trace-abc" {
		t.Errorf("thought.TraceID = %q, want %q", thought.TraceID, "trace-abc")
	}
	// Token usage metadata preserved
	inputTokens, ok := thought.Metadata["gen_ai.usage.input_tokens"]
	if !ok {
		t.Fatal("thought span missing gen_ai.usage.input_tokens")
	}
	if v, ok := inputTokens.(float64); !ok || v != 1500 {
		t.Errorf("gen_ai.usage.input_tokens = %v, want 1500", inputTokens)
	}
	outputTokens, ok := thought.Metadata["gen_ai.usage.output_tokens"]
	if !ok {
		t.Fatal("thought span missing gen_ai.usage.output_tokens")
	}
	if v, ok := outputTokens.(float64); !ok || v != 300 {
		t.Errorf("gen_ai.usage.output_tokens = %v, want 300", outputTokens)
	}

	// Verify tool span (span 2) — hasDiff and diff.files
	tool := decoded[2]
	if !tool.HasDiff {
		t.Error("tool span: expected hasDiff=true")
	}
	if tool.Diff == nil {
		t.Fatal("tool span: expected non-nil diff")
	}
	if len(tool.Diff.Files) != 2 {
		t.Fatalf("tool span: expected 2 diff files, got %d", len(tool.Diff.Files))
	}
	if tool.Diff.Files[0].Path != "main.go" {
		t.Errorf("diff file 0 path = %q, want %q", tool.Diff.Files[0].Path, "main.go")
	}
	if tool.Diff.Files[1].Path != "util.go" {
		t.Errorf("diff file 1 path = %q, want %q", tool.Diff.Files[1].Path, "util.go")
	}
	if tool.Diff.Files[0].Patch == "" {
		t.Error("diff file 0 patch should be non-empty")
	}
	if tool.ParentID != "span-stage-1" {
		t.Errorf("tool.ParentID = %q, want %q", tool.ParentID, "span-stage-1")
	}
}
