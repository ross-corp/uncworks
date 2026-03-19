package temporal

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseOpenSpecJSON extracts JSON from OpenSpec CLI output.
// The CLI often prefixes JSON with text like "- Loading change status..."
// This finds the first '{' and parses from there.
func parseOpenSpecJSON(raw string) (json.RawMessage, error) {
	idx := strings.Index(raw, "{")
	if idx < 0 {
		return nil, fmt.Errorf("no JSON found in output: %q", truncate(raw, 200))
	}
	jsonStr := raw[idx:]
	// Validate it's actual JSON
	var check json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &check); err != nil {
		return nil, fmt.Errorf("invalid JSON in output: %w (raw: %q)", err, truncate(jsonStr, 200))
	}
	return check, nil
}

// OpenSpecListResponse is the parsed output of `openspec list --json`.
type OpenSpecListResponse struct {
	Changes []OpenSpecChangeInfo `json:"changes"`
}

// OpenSpecChangeInfo is a single change in the list response.
type OpenSpecChangeInfo struct {
	Name           string `json:"name"`
	CompletedTasks int    `json:"completedTasks"`
	TotalTasks     int    `json:"totalTasks"`
	Status         string `json:"status"`
	LastModified   string `json:"lastModified"`
}

// parseOpenSpecListResponse parses `openspec list --json` output.
func parseOpenSpecListResponse(raw string) (*OpenSpecListResponse, error) {
	jsonBytes, err := parseOpenSpecJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	var resp OpenSpecListResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal list response: %w", err)
	}
	return &resp, nil
}

// FindChange looks up a change by name in the list response.
func (r *OpenSpecListResponse) FindChange(name string) *OpenSpecChangeInfo {
	for i := range r.Changes {
		if r.Changes[i].Name == name {
			return &r.Changes[i]
		}
	}
	return nil
}

// OpenSpecValidateResponse is the parsed output of `openspec validate --json`.
type OpenSpecValidateResponse struct {
	Items []OpenSpecValidateItem `json:"items"`
}

// OpenSpecValidateItem is a single item validation result.
type OpenSpecValidateItem struct {
	ID     string                  `json:"id"`
	Type   string                  `json:"type"`
	Valid  bool                    `json:"valid"`
	Issues []OpenSpecValidateIssue `json:"issues"`
}

// OpenSpecValidateIssue is a single validation issue.
type OpenSpecValidateIssue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// parseOpenSpecValidateResponse parses `openspec validate --json` output.
func parseOpenSpecValidateResponse(raw string) (*OpenSpecValidateResponse, error) {
	jsonBytes, err := parseOpenSpecJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parse validate response: %w", err)
	}
	var resp OpenSpecValidateResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal validate response: %w", err)
	}
	return &resp, nil
}

// OpenSpecStatusResponse is the parsed output of `openspec status --json`.
type OpenSpecStatusResponse struct {
	ChangeName    string                   `json:"changeName"`
	SchemaName    string                   `json:"schemaName"`
	IsComplete    bool                     `json:"isComplete"`
	ApplyRequires []string                 `json:"applyRequires"`
	Artifacts     []OpenSpecStatusArtifact `json:"artifacts"`
}

// OpenSpecStatusArtifact is a single artifact in the status response.
type OpenSpecStatusArtifact struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// parseOpenSpecStatusResponse parses `openspec status --json` output.
func parseOpenSpecStatusResponse(raw string) (*OpenSpecStatusResponse, error) {
	jsonBytes, err := parseOpenSpecJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parse status response: %w", err)
	}
	var resp OpenSpecStatusResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal status response: %w", err)
	}
	return &resp, nil
}

// AllArtifactsDone checks if all applyRequires artifacts have status "done".
func (r *OpenSpecStatusResponse) AllArtifactsDone() bool {
	required := make(map[string]bool)
	for _, id := range r.ApplyRequires {
		required[id] = true
	}
	for _, a := range r.Artifacts {
		if required[a.ID] && a.Status != "done" {
			return false
		}
	}
	return true
}

// MissingArtifacts returns the IDs of required artifacts that aren't done.
func (r *OpenSpecStatusResponse) MissingArtifacts() []string {
	required := make(map[string]bool)
	for _, id := range r.ApplyRequires {
		required[id] = true
	}
	var missing []string
	for _, a := range r.Artifacts {
		if required[a.ID] && a.Status != "done" {
			missing = append(missing, a.ID)
		}
	}
	return missing
}

// OpenSpecInstructionsResponse is the parsed output of `openspec instructions <artifact> --json`.
type OpenSpecInstructionsResponse struct {
	Template string `json:"template"`
}

// parseOpenSpecInstructionsResponse parses `openspec instructions <artifact> --json` output
// and extracts the template field.
func parseOpenSpecInstructionsResponse(raw string) (string, error) {
	jsonBytes, err := parseOpenSpecJSON(raw)
	if err != nil {
		return "", fmt.Errorf("parse instructions response: %w", err)
	}
	var resp OpenSpecInstructionsResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return "", fmt.Errorf("unmarshal instructions response: %w", err)
	}
	return resp.Template, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
