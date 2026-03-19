package temporal

import (
	"strings"
	"testing"
)

func TestParseOpenSpecJSON_Valid(t *testing.T) {
	raw := `{"key": "value"}`
	j, err := parseOpenSpecJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(j) != raw {
		t.Errorf("got %q, want %q", string(j), raw)
	}
}

func TestParseOpenSpecJSON_TextPrefix(t *testing.T) {
	raw := "- Loading change status...\n{\"changeName\": \"test\"}"
	j, err := parseOpenSpecJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(j) != `{"changeName": "test"}` {
		t.Errorf("got %q", string(j))
	}
}

func TestParseOpenSpecJSON_NoJSON(t *testing.T) {
	_, err := parseOpenSpecJSON("no json here")
	if err == nil {
		t.Fatal("expected error for no JSON")
	}
}

func TestParseOpenSpecJSON_Empty(t *testing.T) {
	_, err := parseOpenSpecJSON("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseOpenSpecJSON_MalformedJSON(t *testing.T) {
	_, err := parseOpenSpecJSON("{broken")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseOpenSpecListResponse(t *testing.T) {
	raw := `- Loading...
{
  "changes": [
    {"name": "my-change", "completedTasks": 5, "totalTasks": 10, "status": "in-progress"},
    {"name": "other", "completedTasks": 3, "totalTasks": 3, "status": "complete"}
  ]
}`
	resp, err := parseOpenSpecListResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(resp.Changes))
	}
	c := resp.FindChange("my-change")
	if c == nil {
		t.Fatal("expected to find my-change")
	}
	if c.CompletedTasks != 5 || c.TotalTasks != 10 {
		t.Errorf("tasks: %d/%d, want 5/10", c.CompletedTasks, c.TotalTasks)
	}
	if resp.FindChange("nonexistent") != nil {
		t.Error("expected nil for nonexistent change")
	}
}

func TestParseOpenSpecValidateResponse(t *testing.T) {
	raw := `{
  "items": [
    {"id": "my-change", "type": "change", "valid": true, "issues": []}
  ]
}`
	resp, err := parseOpenSpecValidateResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if !resp.Items[0].Valid {
		t.Error("expected valid: true")
	}
}

func TestParseOpenSpecValidateResponse_Invalid(t *testing.T) {
	raw := `{
  "items": [
    {
      "id": "bad-change",
      "type": "change",
      "valid": false,
      "issues": [
        {"level": "ERROR", "path": "file", "message": "Missing Purpose section"}
      ]
    }
  ]
}`
	resp, err := parseOpenSpecValidateResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items[0].Valid {
		t.Error("expected valid: false")
	}
	if len(resp.Items[0].Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Items[0].Issues))
	}
	if resp.Items[0].Issues[0].Message != "Missing Purpose section" {
		t.Errorf("unexpected issue message: %q", resp.Items[0].Issues[0].Message)
	}
}

func TestParseOpenSpecStatusResponse(t *testing.T) {
	raw := `- Loading change status...
{
  "changeName": "test",
  "schemaName": "spec-driven",
  "isComplete": true,
  "applyRequires": ["tasks"],
  "artifacts": [
    {"id": "proposal", "status": "done"},
    {"id": "design", "status": "done"},
    {"id": "specs", "status": "done"},
    {"id": "tasks", "status": "done"}
  ]
}`
	resp, err := parseOpenSpecStatusResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.AllArtifactsDone() {
		t.Error("expected all artifacts done")
	}
	if len(resp.MissingArtifacts()) != 0 {
		t.Errorf("expected no missing artifacts, got %v", resp.MissingArtifacts())
	}
}

func TestParseOpenSpecInstructionsResponse_Valid(t *testing.T) {
	raw := `- Loading instructions...
{
  "template": "## Why\n\nDescribe the problem.\n\n## What Changes\n\nDescribe the solution."
}`
	tmpl, err := parseOpenSpecInstructionsResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl == "" {
		t.Fatal("expected non-empty template")
	}
	if !strings.Contains(tmpl, "## Why") {
		t.Errorf("expected template to contain '## Why', got %q", tmpl)
	}
}

func TestParseOpenSpecInstructionsResponse_EmptyTemplate(t *testing.T) {
	raw := `{"template": ""}`
	tmpl, err := parseOpenSpecInstructionsResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl != "" {
		t.Errorf("expected empty template, got %q", tmpl)
	}
}

func TestParseOpenSpecInstructionsResponse_NoJSON(t *testing.T) {
	_, err := parseOpenSpecInstructionsResponse("no json here at all")
	if err == nil {
		t.Fatal("expected error for input without JSON")
	}
}

func TestParseOpenSpecInstructionsResponse_MalformedJSON(t *testing.T) {
	_, err := parseOpenSpecInstructionsResponse("{broken json")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseOpenSpecInstructionsResponse_MissingTemplate(t *testing.T) {
	raw := `{"other_field": "value"}`
	tmpl, err := parseOpenSpecInstructionsResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// template field not present → defaults to empty string
	if tmpl != "" {
		t.Errorf("expected empty template for missing field, got %q", tmpl)
	}
}

func TestParseOpenSpecStatusResponse_Incomplete(t *testing.T) {
	raw := `{
  "changeName": "test",
  "applyRequires": ["tasks"],
  "artifacts": [
    {"id": "proposal", "status": "done"},
    {"id": "tasks", "status": "blocked"}
  ]
}`
	resp, err := parseOpenSpecStatusResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AllArtifactsDone() {
		t.Error("expected not all done")
	}
	missing := resp.MissingArtifacts()
	if len(missing) != 1 || missing[0] != "tasks" {
		t.Errorf("expected [tasks] missing, got %v", missing)
	}
}
