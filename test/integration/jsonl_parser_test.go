package integration

import (
	"strings"
	"testing"

	"github.com/uncworks/aot/internal/server"
)

// fullConversationJSONL simulates a complete agent conversation with session,
// agent_start, message lifecycle, tool execution, and agent_end events.
var fullConversationJSONL = strings.Join([]string{
	// Session start
	`{"type":"session","timestamp":"2025-06-15T10:00:00Z"}`,
	// Agent starts
	`{"type":"agent_start"}`,
	// First assistant message: thinking + text
	`{"type":"message_start"}`,
	`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Let me read "}}`,
	`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"the file."}}`,
	`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Let me read the file."},{"type":"toolCall","id":"tc_1","name":"Read","arguments":{"file_path":"/src/main.go"}}],"timestamp":"2025-06-15T10:00:01Z","model":"claude-opus-4-20250514"}}`,
	// Tool execution
	`{"type":"tool_execution_start","toolName":"Read","toolCallId":"tc_1","args":{"file_path":"/src/main.go"}}`,
	`{"type":"tool_execution_end","toolName":"Read","toolCallId":"tc_1","isError":false,"result":{"content":"package main\nfunc main() {}"}}`,
	// Second assistant message after tool result
	`{"type":"message_start"}`,
	`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"The file contains a simple Go program."}}`,
	`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"The file contains a simple Go program."}],"timestamp":"2025-06-15T10:00:02Z","model":"claude-opus-4-20250514"}}`,
	// Agent finishes
	`{"type":"agent_end","messages":[{"role":"assistant","content":[{"type":"text","text":"The file contains a simple Go program."}],"timestamp":"2025-06-15T10:00:02Z","model":"claude-opus-4-20250514"}]}`,
}, "\n")

func TestParseAgentJSONL_FullConversation(t *testing.T) {
	entries := server.ParseAgentJSONL(fullConversationJSONL)

	if len(entries) == 0 {
		t.Fatal("expected non-empty entries from ParseAgentJSONL")
	}

	// Verify we get expected entry types.
	typeCounts := make(map[string]int)
	for _, e := range entries {
		typeCounts[e.Type]++
	}

	// We expect:
	// - "system" entries for agent_start and agent_end (2 total)
	// - "assistant" entries for the assistant text (deduplicated)
	// - "tool_call" for the Read call
	// - "tool_result" for the Read result
	if typeCounts["system"] < 1 {
		t.Errorf("expected at least 1 system entry, got %d", typeCounts["system"])
	}
	if typeCounts["tool_call"] < 1 {
		t.Errorf("expected at least 1 tool_call entry, got %d", typeCounts["tool_call"])
	}
	if typeCounts["tool_result"] < 1 {
		t.Errorf("expected at least 1 tool_result entry, got %d", typeCounts["tool_result"])
	}
	if typeCounts["assistant"] < 1 {
		t.Errorf("expected at least 1 assistant entry, got %d", typeCounts["assistant"])
	}

	// Verify the tool_call has the right tool name.
	for _, e := range entries {
		if e.Type == "tool_call" {
			if e.ToolName != "Read" {
				t.Errorf("expected tool_call ToolName=Read, got %q", e.ToolName)
			}
			break
		}
	}

	// Verify the tool_result has content.
	for _, e := range entries {
		if e.Type == "tool_result" {
			if e.Content == "" {
				t.Error("expected non-empty tool_result content")
			}
			break
		}
	}

	t.Logf("Parsed %d entries: %v", len(entries), typeCounts)
}

func TestParseAgentJSONL_EmptyInput(t *testing.T) {
	entries := server.ParseAgentJSONL("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseAgentJSONL_DelegateTask(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"session","timestamp":"2025-06-15T10:00:00Z"}`,
		`{"type":"tool_execution_start","toolName":"delegate_task","toolCallId":"tc_d1","args":{"task":"do something"}}`,
		`{"type":"tool_execution_end","toolName":"delegate_task","toolCallId":"tc_d1","isError":false,"result":{"content":"task done"}}`,
	}, "\n")

	entries := server.ParseAgentJSONL(raw)

	var foundDelegate bool
	for _, e := range entries {
		if e.Type == "delegate" {
			foundDelegate = true
			if e.ToolName != "delegate_task" {
				t.Errorf("expected delegate ToolName=delegate_task, got %q", e.ToolName)
			}
		}
	}
	if !foundDelegate {
		t.Error("expected a 'delegate' type entry for delegate_task tool call")
	}
}

func TestParseThinkingFromLines_CompletedConversation(t *testing.T) {
	// A completed message (has message_end after message_start) should NOT
	// report thinking=true.
	lines := []string{
		`{"type":"session","timestamp":"2025-06-15T10:00:00Z"}`,
		`{"type":"agent_start"}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Hello world"}}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Hello world"}]}}`,
		`{"type":"agent_end","messages":[]}`,
	}

	result := server.ParseThinkingFromLines(lines)
	if result.Thinking {
		t.Error("expected Thinking=false for a completed conversation, got true")
	}
}

func TestParseThinkingFromLines_InProgressMessage(t *testing.T) {
	// An in-progress message (message_start with updates but no message_end)
	// should report thinking=true.
	lines := []string{
		`{"type":"session","timestamp":"2025-06-15T10:00:00Z"}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Working on "}}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"the solution..."}}`,
	}

	result := server.ParseThinkingFromLines(lines)
	if !result.Thinking {
		t.Error("expected Thinking=true for in-progress message, got false")
	}
	if result.Text != "Working on the solution..." {
		t.Errorf("expected accumulated text 'Working on the solution...', got %q", result.Text)
	}
}

func TestParseAgentJSONLAndThinkingAgree(t *testing.T) {
	// Feed both parsers the same complete conversation and verify that:
	// - ParseAgentJSONL extracts entries successfully
	// - ParseThinkingFromLines reports no in-progress thinking
	lines := strings.Split(fullConversationJSONL, "\n")

	entries := server.ParseAgentJSONL(fullConversationJSONL)
	if len(entries) == 0 {
		t.Fatal("ParseAgentJSONL returned no entries for full conversation")
	}

	thinking := server.ParseThinkingFromLines(lines)
	if thinking.Thinking {
		t.Error("ParseThinkingFromLines reports thinking=true on a completed conversation")
	}

	// Verify that text content from JSONL entries is consistent.
	var assistantTexts []string
	for _, e := range entries {
		if e.Type == "assistant" && e.Content != "" {
			assistantTexts = append(assistantTexts, e.Content)
		}
	}
	if len(assistantTexts) == 0 {
		t.Error("expected at least one assistant text entry from ParseAgentJSONL")
	}
}
