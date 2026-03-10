package hydration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHydrator_DevboxSetup(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Create a fake workspace with devbox.json
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, "devbox.json"), []byte(`{"packages":["bun@latest"]}`), 0o644)

	// Pre-create the bare dir so clone is skipped
	os.MkdirAll(filepath.Join(tmpDir, ".bare"), 0o755)

	config := &Config{
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		WorkspaceDir: tmpDir,
		DevboxConfig: "devbox.json",
	}

	h := NewHydrator(config, runner)
	err := h.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have run: worktree add + devbox install
	found := false
	for _, cmd := range runner.commands {
		if cmd.Name == "devbox" && len(cmd.Args) > 0 && cmd.Args[0] == "install" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected devbox install command")
	}
}

func TestHydrator_DevboxConfigNotFound(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Create workspace but no devbox.json in src
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".bare"), 0o755)

	config := &Config{
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		WorkspaceDir: tmpDir,
		DevboxConfig: "devbox.json",
	}

	h := NewHydrator(config, runner)
	err := h.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing devbox config")
	}
	if !strings.Contains(err.Error(), "devbox config not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHydrator_NoDevboxSkipped(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".bare"), 0o755)

	config := &Config{
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		WorkspaceDir: tmpDir,
		DevboxConfig: "", // No devbox config
	}

	h := NewHydrator(config, runner)
	err := h.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should not have run devbox install
	for _, cmd := range runner.commands {
		if cmd.Name == "devbox" {
			t.Error("devbox should not be called when config is empty")
		}
	}
}

func TestHydrator_DevboxInstallFailure(t *testing.T) {
	runner := NewMockRunner()
	runner.On("devbox", MockResult{Err: os.ErrPermission})
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, "devbox.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(tmpDir, ".bare"), 0o755)

	config := &Config{
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		WorkspaceDir: tmpDir,
		DevboxConfig: "devbox.json",
	}

	h := NewHydrator(config, runner)
	err := h.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from devbox install failure")
	}
	if !strings.Contains(err.Error(), "setup devbox") {
		t.Errorf("unexpected error: %v", err)
	}
}
