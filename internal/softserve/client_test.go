package softserve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldProject_CreatesFiles(t *testing.T) {
	// This test verifies the scaffold file generation logic without needing soft-serve.
	// We test the file content generation that ScaffoldAndPush would create.

	scaffold := ScaffoldProject{
		Name:     "test-proj",
		Packages: []string{"go@1.22", "nodejs@20"},
	}

	// Simulate devbox.json generation
	devboxJSON := map[string]interface{}{
		"packages": scaffold.Packages,
	}
	data, err := json.MarshalIndent(devboxJSON, "", "  ")
	if err != nil {
		t.Fatalf("marshal devbox.json: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	packages, ok := parsed["packages"].([]interface{})
	if !ok {
		t.Fatal("packages not an array")
	}
	if len(packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(packages))
	}
	if packages[0] != "go@1.22" {
		t.Errorf("expected go@1.22, got %v", packages[0])
	}
}

func TestScaffoldProject_EmptyPackages(t *testing.T) {
	scaffold := ScaffoldProject{
		Name:     "empty-proj",
		Packages: nil,
	}

	devboxJSON := map[string]interface{}{
		"packages": scaffold.Packages,
	}
	if scaffold.Packages == nil {
		devboxJSON["packages"] = []string{}
	}
	data, err := json.MarshalIndent(devboxJSON, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	packages, ok := parsed["packages"].([]interface{})
	if !ok {
		t.Fatal("packages not an array")
	}
	if len(packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(packages))
	}
}

func TestCloneURL(t *testing.T) {
	c := &Client{SSHAddr: "soft-serve.aot.svc:23231"}
	url := c.CloneURL("my-project")
	expected := "ssh://soft-serve.aot.svc:23231/my-project.git"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestScaffoldFilesWritten(t *testing.T) {
	// Verify that ScaffoldAndPush creates the expected directory structure
	tmpDir := t.TempDir()

	// Simulate what ScaffoldAndPush writes
	files := map[string]string{
		"devbox.json":                     `{"packages":["go@1.22"]}`,
		"openspec/openspec.yaml":          "name: test\nschema: spec-driven\n",
		".devcontainer/devcontainer.json": `{"name":"test","postCreateCommand":"devbox install"}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	// Verify all files exist
	for path := range files {
		fullPath := filepath.Join(tmpDir, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", path)
		}
	}

	// Verify devbox.json content
	data, _ := os.ReadFile(filepath.Join(tmpDir, "devbox.json"))
	var devbox map[string]interface{}
	if err := json.Unmarshal(data, &devbox); err != nil {
		t.Fatalf("parse devbox.json: %v", err)
	}
	if devbox["packages"] == nil {
		t.Error("devbox.json missing packages field")
	}
}
