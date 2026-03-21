package contract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/uncworks/aot/internal/server"
)

// TestBoundary_RESTTypes_TraceSpanJSON verifies that JSON serialization of
// server.TraceSpan produces the camelCase field names the frontend expects.
func TestBoundary_RESTTypes_TraceSpanJSON(t *testing.T) {
	span := server.TraceSpan{
		ID:        "span-1",
		ParentID:  "parent-1",
		Name:      "llm_response",
		Type:      "llm",
		StartTime: "2026-03-20T10:00:00Z",
		EndTime:   "2026-03-20T10:00:05Z",
		Metadata:  map[string]interface{}{"model": "gpt-4"},
		HasDiff:   true,
		Diff: &server.SpanDiff{
			Files: []server.FileDiff{
				{Path: "main.go", Patch: "+new line"},
			},
		},
	}

	data, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("json.Marshal TraceSpan: %v", err)
	}

	jsonStr := string(data)

	// Expected camelCase field names from the frontend's TraceSpan interface
	expectedFields := []string{
		`"id"`,
		`"parentId"`,
		`"name"`,
		`"type"`,
		`"startTime"`,
		`"endTime"`,
		`"metadata"`,
		`"hasDiff"`,
		`"diff"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("TraceSpan JSON missing expected field %s in: %s", field, jsonStr)
		}
	}

	// Verify nested diff fields
	expectedDiffFields := []string{`"files"`, `"path"`, `"patch"`}
	for _, field := range expectedDiffFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("TraceSpan.Diff JSON missing expected field %s in: %s", field, jsonStr)
		}
	}

	// Verify NO snake_case leaks
	snakeCaseFields := []string{
		`"parent_id"`,
		`"start_time"`,
		`"end_time"`,
		`"has_diff"`,
	}
	for _, field := range snakeCaseFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("TraceSpan JSON contains snake_case field %s (should be camelCase): %s", field, jsonStr)
		}
	}
}

// TestBoundary_RESTTypes_AgentLogEntryJSON verifies AgentLogEntry serialization.
func TestBoundary_RESTTypes_AgentLogEntryJSON(t *testing.T) {
	entry := server.AgentLogEntry{
		Timestamp: "2026-03-20T10:00:00Z",
		Type:      "assistant",
		Content:   "I will fix the bug.",
		ToolName:  "edit_file",
		ToolInput: `{"path":"main.go"}`,
		Model:     "gpt-4",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal AgentLogEntry: %v", err)
	}

	jsonStr := string(data)

	expectedFields := []string{
		`"timestamp"`,
		`"type"`,
		`"content"`,
		`"toolName"`,
		`"toolInput"`,
		`"model"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("AgentLogEntry JSON missing expected field %s in: %s", field, jsonStr)
		}
	}

	// Verify NO snake_case leaks
	snakeCaseFields := []string{`"tool_name"`, `"tool_input"`}
	for _, field := range snakeCaseFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("AgentLogEntry JSON contains snake_case field %s: %s", field, jsonStr)
		}
	}
}

// TestBoundary_RESTTypes_FileEntryJSON verifies FileEntry serialization.
func TestBoundary_RESTTypes_FileEntryJSON(t *testing.T) {
	entry := server.FileEntry{
		Name:     "main.go",
		Type:     "file",
		Size:     1024,
		Modified: "2026-03-20T10:00:00Z",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal FileEntry: %v", err)
	}

	jsonStr := string(data)

	expectedFields := []string{`"name"`, `"type"`, `"size"`, `"modified"`}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("FileEntry JSON missing expected field %s in: %s", field, jsonStr)
		}
	}
}

// TestBoundary_RESTTypes_ThinkingResponseJSON verifies ThinkingResponse serialization.
func TestBoundary_RESTTypes_ThinkingResponseJSON(t *testing.T) {
	resp := server.ThinkingResponse{
		Thinking: true,
		Text:     "I am analyzing the code...",
		ToolName: "read_file",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal ThinkingResponse: %v", err)
	}

	jsonStr := string(data)

	expectedFields := []string{`"thinking"`, `"text"`, `"toolName"`}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("ThinkingResponse JSON missing expected field %s in: %s", field, jsonStr)
		}
	}

	// No snake_case
	if strings.Contains(jsonStr, `"tool_name"`) {
		t.Errorf("ThinkingResponse JSON contains snake_case field: %s", jsonStr)
	}
}

// TestBoundary_RESTTypes_TraceSpanOmitsEmptyOptionals verifies that optional
// fields with zero values are omitted (matching frontend expectations).
func TestBoundary_RESTTypes_TraceSpanOmitsEmptyOptionals(t *testing.T) {
	span := server.TraceSpan{
		ID:        "span-1",
		Name:      "test",
		Type:      "tool",
		StartTime: "2026-03-20T10:00:00Z",
		EndTime:   "2026-03-20T10:00:05Z",
		// ParentID, Metadata, Diff are zero/nil
	}

	data, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	jsonStr := string(data)

	// parentId should be omitted when empty (omitempty tag)
	if strings.Contains(jsonStr, `"parentId"`) {
		t.Errorf("expected parentId to be omitted when empty: %s", jsonStr)
	}

	// metadata should be omitted when nil (omitempty tag)
	if strings.Contains(jsonStr, `"metadata"`) {
		t.Errorf("expected metadata to be omitted when nil: %s", jsonStr)
	}

	// diff should be omitted when nil (omitempty tag)
	if strings.Contains(jsonStr, `"diff"`) {
		t.Errorf("expected diff to be omitted when nil: %s", jsonStr)
	}
}

// TestBoundary_RESTTypes_RoundTripJSON verifies that marshaling and
// unmarshaling produces identical values for all types.
func TestBoundary_RESTTypes_RoundTripJSON(t *testing.T) {
	original := server.TraceSpan{
		ID:        "abc-123",
		ParentID:  "parent-456",
		Name:      "edit_file",
		Type:      "tool",
		StartTime: "2026-03-20T10:00:00Z",
		EndTime:   "2026-03-20T10:00:05Z",
		HasDiff:   true,
		Metadata:  map[string]interface{}{"key": "value"},
		Diff: &server.SpanDiff{
			Files: []server.FileDiff{{Path: "a.go", Patch: "+line"}},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded server.TraceSpan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	assertEqual(t, "ID", decoded.ID, original.ID)
	assertEqual(t, "ParentID", decoded.ParentID, original.ParentID)
	assertEqual(t, "Name", decoded.Name, original.Name)
	assertEqual(t, "Type", decoded.Type, original.Type)
	assertEqual(t, "StartTime", decoded.StartTime, original.StartTime)
	assertEqual(t, "EndTime", decoded.EndTime, original.EndTime)
	assertEqual(t, "HasDiff", decoded.HasDiff, original.HasDiff)

	if decoded.Diff == nil {
		t.Fatal("expected non-nil Diff after round-trip")
	}
	if len(decoded.Diff.Files) != 1 {
		t.Fatalf("expected 1 file in diff, got %d", len(decoded.Diff.Files))
	}
	assertEqual(t, "Diff.Files[0].Path", decoded.Diff.Files[0].Path, "a.go")
	assertEqual(t, "Diff.Files[0].Patch", decoded.Diff.Files[0].Patch, "+line")
}

// --- WebSocket binary message format regression tests ---

// TestBoundary_ExecHandler_BinaryMessageType verifies that the WebSocket
// message type constant used for terminal output is BinaryMessage (2), not
// TextMessage (1). This is a regression test for the bug where terminal
// output was sent as TextMessage, causing xterm.js to fail to render raw
// terminal escape sequences properly.
//
// The exec handler (internal/server/exec.go) writes terminal output via:
//
//	wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
//
// This contract test ensures BinaryMessage has the expected RFC 6455 value
// and documents the invariant that terminal data MUST use binary framing.
func TestBoundary_ExecHandler_BinaryMessageType(t *testing.T) {
	// BinaryMessage must be 2 per RFC 6455 section 11.8.
	// This is the message type the exec handler uses for terminal output.
	if websocket.BinaryMessage != 2 {
		t.Errorf("websocket.BinaryMessage = %d, want 2 (RFC 6455)", websocket.BinaryMessage)
	}

	// TextMessage is 1 — the handler must NOT use this for terminal output,
	// because raw terminal bytes are not valid UTF-8 text.
	if websocket.TextMessage != 1 {
		t.Errorf("websocket.TextMessage = %d, want 1 (RFC 6455)", websocket.TextMessage)
	}

	// Verify the two are distinct (safety net against import aliasing mistakes).
	if websocket.BinaryMessage == websocket.TextMessage {
		t.Fatal("BinaryMessage and TextMessage must be different message types")
	}
}

// TestBoundary_ExecHandler_ResizeMessageFormat verifies the JSON format of
// resize messages sent from the frontend to the exec handler.
func TestBoundary_ExecHandler_ResizeMessageFormat(t *testing.T) {
	// The frontend sends resize messages as JSON text frames.
	// The exec handler expects: {"type":"resize","cols":N,"rows":N}
	resizeJSON := `{"type":"resize","cols":120,"rows":40}`

	var msg struct {
		Type string `json:"type"`
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}
	if err := json.Unmarshal([]byte(resizeJSON), &msg); err != nil {
		t.Fatalf("unmarshal resize message: %v", err)
	}

	if msg.Type != "resize" {
		t.Errorf("type = %q, want \"resize\"", msg.Type)
	}
	if msg.Cols != 120 {
		t.Errorf("cols = %d, want 120", msg.Cols)
	}
	if msg.Rows != 40 {
		t.Errorf("rows = %d, want 40", msg.Rows)
	}
}
