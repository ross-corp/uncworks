package temporal

import (
	"encoding/json"
	"testing"
)

func TestTraceSpanData_MarshalJSON_AllFields(t *testing.T) {
	span := TraceSpanData{
		ID:        "span-001",
		TraceID:   "trace-abc",
		ParentID:  "parent-xyz",
		Name:      "execute.thought",
		Type:      "llm",
		StartTime: "2026-03-20T12:00:00Z",
		EndTime:   "2026-03-20T12:01:00Z",
		Status:    "ok",
		Metadata: map[string]interface{}{
			"model":        "deepseek-v3.1",
			"inputTokens":  float64(1500),
			"outputTokens": float64(500),
		},
		HasDiff: true,
	}

	data, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("json.Marshal(TraceSpanData): %v", err)
	}

	// Verify it's valid JSON by unmarshalling into a generic map
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal round-trip to map: %v", err)
	}

	// Check all expected keys are present
	expectedKeys := []string{"id", "traceId", "parentId", "name", "type", "startTime", "endTime", "status", "metadata", "hasDiff"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in marshalled JSON", key)
		}
	}

	// Verify specific values
	if raw["id"] != "span-001" {
		t.Errorf("id = %v, want span-001", raw["id"])
	}
	if raw["traceId"] != "trace-abc" {
		t.Errorf("traceId = %v, want trace-abc", raw["traceId"])
	}
	if raw["hasDiff"] != true {
		t.Errorf("hasDiff = %v, want true", raw["hasDiff"])
	}
}

func TestTraceSpanData_RoundTrip(t *testing.T) {
	original := TraceSpanData{
		ID:        "span-002",
		TraceID:   "trace-def",
		ParentID:  "parent-456",
		Name:      "plan.bash",
		Type:      "tool",
		StartTime: "2026-03-20T10:00:00Z",
		EndTime:   "2026-03-20T10:00:05Z",
		Status:    "error",
		Metadata: map[string]interface{}{
			"command":  "go test ./...",
			"exitCode": float64(1),
		},
		HasDiff: false,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TraceSpanData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.TraceID != original.TraceID {
		t.Errorf("TraceID: got %q, want %q", decoded.TraceID, original.TraceID)
	}
	if decoded.ParentID != original.ParentID {
		t.Errorf("ParentID: got %q, want %q", decoded.ParentID, original.ParentID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.StartTime != original.StartTime {
		t.Errorf("StartTime: got %q, want %q", decoded.StartTime, original.StartTime)
	}
	if decoded.EndTime != original.EndTime {
		t.Errorf("EndTime: got %q, want %q", decoded.EndTime, original.EndTime)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
	if decoded.HasDiff != original.HasDiff {
		t.Errorf("HasDiff: got %v, want %v", decoded.HasDiff, original.HasDiff)
	}
	if decoded.Metadata["command"] != original.Metadata["command"] {
		t.Errorf("Metadata[command]: got %v, want %v", decoded.Metadata["command"], original.Metadata["command"])
	}
}

func TestTraceSpanData_OmitsEmpty(t *testing.T) {
	// A minimal span should omit optional fields via omitempty
	span := TraceSpanData{
		ID:        "span-min",
		Name:      "minimal",
		Type:      "llm",
		StartTime: "2026-03-20T10:00:00Z",
	}

	data, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// These fields have omitempty and should NOT be in output
	for _, key := range []string{"traceId", "parentId", "endTime", "status", "metadata"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected key %q to be omitted when empty, but it was present", key)
		}
	}

	// hasDiff should always be present (no omitempty, bool defaults to false)
	if _, ok := raw["hasDiff"]; !ok {
		t.Error("hasDiff should always be present even when false")
	}
}
