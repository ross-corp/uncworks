package server

import (
	"encoding/json"
	"testing"
)

func TestIsHiddenDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".aot", true},
		{".bare", true},
		{".openspec", true},
		{".git", false},
		{"src", false},
		{"node_modules", false},
		{"openspec", false},
		{".env", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHiddenDir(tt.name)
			if got != tt.want {
				t.Errorf("isHiddenDir(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestParseLsOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantNames []string
		wantTypes []string
	}{
		{
			name: "normal output with files and dirs",
			input: `total 32
drwxr-xr-x 4 root root 4096 2024-01-15 10:30 src
-rw-r--r-- 1 root root 1234 2024-01-15 10:30 main.go
lrwxrwxrwx 1 root root   20 2024-01-15 10:30 link.txt -> target.txt`,
			wantCount: 3,
			wantNames: []string{"src", "main.go", "link.txt -> target.txt"},
			wantTypes: []string{"directory", "file", "symlink"},
		},
		{
			name: "skips hidden dirs (.aot, .bare, .openspec)",
			input: `total 20
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 .aot
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 .bare
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 .openspec
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 src
-rw-r--r-- 1 root root 1234 2024-01-15 10:30 README.md`,
			wantCount: 2,
			wantNames: []string{"src", "README.md"},
			wantTypes: []string{"directory", "file"},
		},
		{
			name: "skips dot and dotdot",
			input: `total 8
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 .
drwxr-xr-x 2 root root 4096 2024-01-15 10:30 ..
-rw-r--r-- 1 root root 1234 2024-01-15 10:30 file.txt`,
			wantCount: 1,
			wantNames: []string{"file.txt"},
			wantTypes: []string{"file"},
		},
		{
			name:      "empty output",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "only total line",
			input:     "total 0",
			wantCount: 0,
		},
		{
			name: "file with spaces in name",
			input: `total 4
-rw-r--r-- 1 root root 1234 2024-01-15 10:30 my file name.txt`,
			wantCount: 1,
			wantNames: []string{"my file name.txt"},
			wantTypes: []string{"file"},
		},
		{
			name: "parses size and modified time",
			input: `total 4
-rw-r--r-- 1 root root 5678 2024-03-20 14:00 data.json`,
			wantCount: 1,
			wantNames: []string{"data.json"},
			wantTypes: []string{"file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := parseLsOutput(tt.input)
			if len(entries) != tt.wantCount {
				t.Fatalf("parseLsOutput returned %d entries, want %d: %+v", len(entries), tt.wantCount, entries)
			}
			for i, name := range tt.wantNames {
				if entries[i].Name != name {
					t.Errorf("entry[%d].Name = %q, want %q", i, entries[i].Name, name)
				}
			}
			for i, typ := range tt.wantTypes {
				if entries[i].Type != typ {
					t.Errorf("entry[%d].Type = %q, want %q", i, entries[i].Type, typ)
				}
			}
			// Check size/modified for last test case
			if tt.name == "parses size and modified time" && len(entries) > 0 {
				if entries[0].Size != 5678 {
					t.Errorf("entry[0].Size = %d, want 5678", entries[0].Size)
				}
				if entries[0].Modified != "2024-03-20 14:00" {
					t.Errorf("entry[0].Modified = %q, want %q", entries[0].Modified, "2024-03-20 14:00")
				}
			}
		})
	}
}

func TestParseAgentJSONL_DedupToolResults(t *testing.T) {
	// Simulate a JSONL stream where tool_execution_end provides a tool result,
	// then turn_end and agent_end repeat the same tool result content.
	// parseAgentJSONL should NOT emit duplicate tool_result entries.
	lines := []string{
		// session start
		`{"type":"session","timestamp":"2024-01-01T00:00:00Z"}`,
		// agent_start
		`{"type":"agent_start"}`,
		// tool_execution_start
		`{"type":"tool_execution_start","toolName":"bash","toolCallId":"tc-1","args":{"command":"ls"}}`,
		// tool_execution_end with result
		`{"type":"tool_execution_end","toolName":"bash","toolCallId":"tc-1","isError":false,"result":{"content":"file1.txt\nfile2.txt"}}`,
		// turn_end with the same toolResult content (should be deduped)
		`{"type":"turn_end","message":{"role":"assistant","timestamp":"2024-01-01T00:00:01Z","content":[{"type":"toolResult","toolName":"bash","content":"file1.txt\nfile2.txt"}]}}`,
		// agent_end with the same tool result in messages (should be skipped because role=toolResult)
		`{"type":"agent_end","messages":[{"role":"toolResult","content":[{"type":"text","text":"file1.txt\nfile2.txt"}]}]}`,
	}

	raw := ""
	for _, l := range lines {
		raw += l + "\n"
	}

	entries := parseAgentJSONL(raw)

	// Count tool_result entries
	toolResultCount := 0
	for _, e := range entries {
		if e.Type == "tool_result" {
			toolResultCount++
		}
	}

	if toolResultCount != 1 {
		t.Errorf("expected exactly 1 tool_result entry (deduped), got %d", toolResultCount)
		for i, e := range entries {
			t.Logf("  entry[%d]: type=%q toolName=%q content=%q", i, e.Type, e.ToolName, truncate(e.Content, 60))
		}
	}
}

func TestParseAgentJSONL_DedupToolCalls(t *testing.T) {
	// Same tool call ID appearing in tool_execution_start and then message_end
	// should only produce one tool_call entry.
	lines := []string{
		`{"type":"session","timestamp":"2024-01-01T00:00:00Z"}`,
		`{"type":"tool_execution_start","toolName":"bash","toolCallId":"tc-42","args":{"command":"pwd"}}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"toolCall","name":"bash","id":"tc-42","arguments":{"command":"pwd"}}]}}`,
	}

	raw := ""
	for _, l := range lines {
		raw += l + "\n"
	}

	entries := parseAgentJSONL(raw)

	toolCallCount := 0
	for _, e := range entries {
		if e.Type == "tool_call" {
			toolCallCount++
		}
	}

	if toolCallCount != 1 {
		t.Errorf("expected exactly 1 tool_call entry (deduped by toolCallId), got %d", toolCallCount)
		for i, e := range entries {
			t.Logf("  entry[%d]: type=%q toolName=%q", i, e.Type, e.ToolName)
		}
	}
}

func TestParseAgentJSONL_MalformedLines(t *testing.T) {
	raw := `not json at all
{"type":"session","timestamp":"2024-01-01T00:00:00Z"}
{invalid json}
{"type":"agent_start"}
`
	entries := parseAgentJSONL(raw)

	// Should gracefully skip malformed lines and still parse valid ones
	systemCount := 0
	for _, e := range entries {
		if e.Type == "system" && e.Content == "Agent started" {
			systemCount++
		}
	}
	if systemCount != 1 {
		t.Errorf("expected 1 'Agent started' system entry, got %d", systemCount)
	}
}

func TestParseAgentJSONL_EmptyInput(t *testing.T) {
	entries := parseAgentJSONL("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseAgentJSONL_DelegateTaskType(t *testing.T) {
	// delegate_task tool calls should produce "delegate" type entries
	lines := `{"type":"session","timestamp":"2024-01-01T00:00:00Z"}
{"type":"tool_execution_start","toolName":"delegate_task","toolCallId":"tc-99","args":{"task":"build the thing"}}
`
	entries := parseAgentJSONL(lines)

	delegateCount := 0
	for _, e := range entries {
		if e.Type == "delegate" {
			delegateCount++
		}
	}
	if delegateCount != 1 {
		t.Errorf("expected 1 delegate entry for delegate_task, got %d", delegateCount)
	}
}

func TestParseAgentJSONL_AssistantTextDedup(t *testing.T) {
	// Same assistant text from message_end and turn_end should be deduped
	raw := `{"type":"session","timestamp":"2024-01-01T00:00:00Z"}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"I will help you."}]}}
{"type":"turn_end","message":{"role":"assistant","content":[{"type":"text","text":"I will help you."}]}}
`
	entries := parseAgentJSONL(raw)

	assistantCount := 0
	for _, e := range entries {
		if e.Type == "assistant" {
			assistantCount++
		}
	}
	if assistantCount != 1 {
		t.Errorf("expected 1 assistant text entry (deduped), got %d", assistantCount)
	}
}

// Verify that entries round-trip through JSON properly.
func TestAgentLogEntry_JSON(t *testing.T) {
	entry := AgentLogEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Type:      "tool_call",
		ToolName:  "bash",
		ToolInput: `{"command":"ls"}`,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded AgentLogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.ToolName != "bash" {
		t.Errorf("ToolName = %q, want %q", decoded.ToolName, "bash")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
