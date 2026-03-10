package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Init a bare repo and create a worktree
	commands := [][]string{
		{"git", "init", "--bare", filepath.Join(dir, "bare.git")},
		{"git", "clone", filepath.Join(dir, "bare.git"), filepath.Join(dir, "main")},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup command %v failed: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit so we can create worktrees
	mainDir := filepath.Join(dir, "main")
	os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("test"), 0o644)
	for _, args := range [][]string{
		{"git", "-C", mainDir, "add", "."},
		{"git", "-C", mainDir, "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup command %v failed: %v\n%s", args, err, out)
		}
	}

	return mainDir
}

func TestFindWorktrees(t *testing.T) {
	repoDir := setupTestRepo(t)

	worktrees, err := FindWorktrees(repoDir)
	if err != nil {
		t.Fatalf("FindWorktrees: %v", err)
	}

	// Should have at least the main worktree
	if len(worktrees) < 1 {
		t.Fatalf("expected at least 1 worktree, got %d", len(worktrees))
	}
}

func TestFindAOTWorktrees(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create an AOT worktree
	wtDir := filepath.Join(t.TempDir(), "aot-wt")
	cmd := exec.Command("git", "worktree", "add", "-b", "aot/test-branch", wtDir, "HEAD")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	worktrees, err := FindAOTWorktrees(repoDir)
	if err != nil {
		t.Fatalf("FindAOTWorktrees: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 AOT worktree, got %d", len(worktrees))
	}

	if worktrees[0] != wtDir {
		t.Errorf("expected worktree at %s, got %s", wtDir, worktrees[0])
	}
}

func TestFindAOTWorktrees_None(t *testing.T) {
	repoDir := setupTestRepo(t)

	worktrees, err := FindAOTWorktrees(repoDir)
	if err != nil {
		t.Fatalf("FindAOTWorktrees: %v", err)
	}

	if len(worktrees) != 0 {
		t.Fatalf("expected 0 AOT worktrees, got %d", len(worktrees))
	}
}
