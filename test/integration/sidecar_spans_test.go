package integration

import (
	"strings"
	"testing"

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
