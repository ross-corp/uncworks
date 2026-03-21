package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/uncworks/aot/internal/sidecar"
)

func TestResolveWorkDir_SingleRepo(t *testing.T) {
	// Single repo: .git at workspace root → return workspace root.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", base, base, got, base)
	}
}

func TestResolveWorkDir_RepoInSubdir(t *testing.T) {
	// Repo in subdirectory: /ws/neph.nvim/.git → return /ws/neph.nvim.
	base := t.TempDir()
	repoDir := filepath.Join(base, "neph.nvim")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", base, base, got, repoDir)
	}
}

func TestResolveWorkDir_LegacySrcLayout(t *testing.T) {
	// Legacy src layout: /ws/src/neph.nvim/.git
	// The function only scans immediate subdirs of the base. Since "src" itself
	// has no .git, but src/neph.nvim does, the function checks src/.git which
	// does not exist. So it falls through and returns base.
	// However, if we place .git inside src/ itself, it would match src.
	//
	// To test the "src contains a repo" case: create src/.git → returns /ws/src.
	base := t.TempDir()
	srcDir := filepath.Join(base, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != srcDir {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", base, base, got, srcDir)
	}
}

func TestResolveWorkDir_ExplicitPath(t *testing.T) {
	// When repoPath differs from defaultBase, return it unchanged.
	base := t.TempDir()
	custom := filepath.Join(base, "custom")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(custom, base)
	if got != custom {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", custom, base, got, custom)
	}
}

func TestResolveWorkDir_SkipBareAndAotDirs(t *testing.T) {
	// .bare and .aot directories should be skipped even if they contain .git.
	// Only the actual repo directory should be returned.
	base := t.TempDir()

	// Create .bare/foo/.git — should be skipped (.bare is excluded)
	if err := os.MkdirAll(filepath.Join(base, ".bare", "foo", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create .aot/.git — should be skipped (.aot is excluded)
	if err := os.MkdirAll(filepath.Join(base, ".aot", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create the real repo
	repoDir := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", base, base, got, repoDir)
	}
}

// --- Debug pod regression tests ---

// TestResolveWorkDir_DebugPodBareWorktreeLayout verifies that resolveWorkDir
// correctly finds the worktree directory in a debug pod scenario where both
// .bare/<repo> (bare repo) and <repo>/ (worktree with .git file) exist.
// This is a regression test for the bug where debug pods failed to resolve
// the working directory because .bare was not being skipped.
func TestResolveWorkDir_DebugPodBareWorktreeLayout(t *testing.T) {
	base := t.TempDir()

	// Simulate the hydration layout: .bare/myrepo (bare repo) + myrepo/ (worktree)
	bareDir := filepath.Join(base, ".bare", "myrepo")
	if err := os.MkdirAll(bareDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// The worktree has a .git *file* (not directory) pointing to the bare repo.
	// Git worktrees use a .git file with "gitdir: <path>" content.
	repoDir := filepath.Join(base, "myrepo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(repoDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: "+bareDir), 0o644); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q (should find worktree, not .bare)", base, base, got, repoDir)
	}
}

// TestResolveWorkDir_DebugPodMultipleRepos verifies that when multiple repo
// worktrees exist alongside .bare, the first non-hidden repo is returned.
func TestResolveWorkDir_DebugPodMultipleRepos(t *testing.T) {
	base := t.TempDir()

	// .bare directory (should be skipped)
	if err := os.MkdirAll(filepath.Join(base, ".bare", "repo1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".bare", "repo2"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Two worktrees
	for _, name := range []string{"repo1", "repo2"} {
		repoDir := filepath.Join(base, name)
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, ".git"), []byte("gitdir: "+filepath.Join(base, ".bare", name)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	// Should return the first repo found (directory listing order), not .bare
	if got == base || got == filepath.Join(base, ".bare") {
		t.Errorf("ResolveWorkDirAt should not return base or .bare, got %q", got)
	}
	// It should be one of the repo worktrees
	if got != filepath.Join(base, "repo1") && got != filepath.Join(base, "repo2") {
		t.Errorf("ResolveWorkDirAt should return a repo worktree, got %q", got)
	}
}

// TestResolveWorkDir_GitFileNotJustGitDir verifies that ResolveWorkDirAt
// detects .git as a file (worktree pointer), not just as a directory.
// Git worktrees create a .git file with "gitdir: <path>" content.
func TestResolveWorkDir_GitFileNotJustGitDir(t *testing.T) {
	base := t.TempDir()

	repoDir := filepath.Join(base, "myrepo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// .git as a *file* (worktree style), not a directory
	gitFile := filepath.Join(repoDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /workspace/.bare/myrepo"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != repoDir {
		t.Errorf("ResolveWorkDirAt should detect .git file (worktree), got %q, want %q", got, repoDir)
	}
}

func TestResolveWorkDir_NoGitAnywhere(t *testing.T) {
	// No .git anywhere → return the base directory unchanged.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "somedir"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := sidecar.ResolveWorkDirAt(base, base)
	if got != base {
		t.Errorf("ResolveWorkDirAt(%q, %q) = %q, want %q", base, base, got, base)
	}
}
