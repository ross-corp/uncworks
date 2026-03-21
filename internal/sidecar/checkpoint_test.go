package sidecar

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temp dir with a git repo and an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", dir)
	run("git", "config", "user.name", "test")
	run("git", "config", "user.email", "test@test")

	if err := os.WriteFile(filepath.Join(dir, "initial.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "initial")

	return dir
}

func TestCreateGitCheckpoint_CreatesCommit(t *testing.T) {
	dir := initTestRepo(t)

	// Reset checkpoint state
	checkpointMu.Lock()
	lastCheckpointSHA = ""
	checkpointMu.Unlock()

	// Write a new file
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create checkpoint
	sha, diff := createGitCheckpoint(dir, "write")

	// Verify commit created
	if sha == "" {
		t.Fatal("expected non-empty SHA")
	}
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if len(diff.Files) == 0 {
		t.Fatal("expected files in diff")
	}

	// Verify file is in diff
	found := false
	for _, f := range diff.Files {
		if strings.Contains(f.Path, "new.txt") {
			found = true
		}
	}
	if !found {
		t.Error("expected new.txt in diff")
	}

	// Verify the commit message contains the tool name
	logCmd := exec.Command("git", "log", "-1", "--pretty=%s")
	logCmd.Dir = dir
	out, err := logCmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(string(out), "aot-checkpoint: write") {
		t.Errorf("expected commit message to contain 'aot-checkpoint: write', got %q", strings.TrimSpace(string(out)))
	}

	// Verify lastCheckpointSHA was updated
	checkpointMu.Lock()
	savedSHA := lastCheckpointSHA
	checkpointMu.Unlock()
	if savedSHA != sha {
		t.Errorf("expected lastCheckpointSHA=%s, got %s", sha, savedSHA)
	}
}

func TestCreateGitCheckpoint_NoChanges(t *testing.T) {
	dir := initTestRepo(t)

	// Reset checkpoint state
	checkpointMu.Lock()
	lastCheckpointSHA = ""
	checkpointMu.Unlock()

	// No new files — should return empty
	sha, diff := createGitCheckpoint(dir, "read")

	if sha != "" {
		t.Errorf("expected empty SHA for no changes, got %s", sha)
	}
	if diff != nil {
		t.Errorf("expected nil diff for no changes, got %+v", diff)
	}
}

func TestCreateGitCheckpoint_StateReset(t *testing.T) {
	dir := initTestRepo(t)

	// Set some checkpoint state
	checkpointMu.Lock()
	lastCheckpointSHA = "abc123fake"
	checkpointMu.Unlock()

	// Reset (simulates what StartAgent does)
	checkpointMu.Lock()
	lastCheckpointSHA = ""
	checkpointMu.Unlock()

	// Write a file and checkpoint
	if err := os.WriteFile(filepath.Join(dir, "after-reset.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	sha, diff := createGitCheckpoint(dir, "write")
	if sha == "" {
		t.Fatal("expected non-empty SHA after reset")
	}
	if diff == nil {
		t.Fatal("expected non-nil diff after reset")
	}

	// Verify lastCheckpointSHA is the new SHA, not the old fake one
	checkpointMu.Lock()
	savedSHA := lastCheckpointSHA
	checkpointMu.Unlock()
	if savedSHA == "abc123fake" {
		t.Error("lastCheckpointSHA was not properly reset")
	}
	if savedSHA != sha {
		t.Errorf("expected lastCheckpointSHA=%s, got %s", sha, savedSHA)
	}
}

func TestCreateGitCheckpoint_ConsecutiveCheckpoints(t *testing.T) {
	dir := initTestRepo(t)

	// Reset checkpoint state
	checkpointMu.Lock()
	lastCheckpointSHA = ""
	checkpointMu.Unlock()

	// First checkpoint: create file1
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	sha1, diff1 := createGitCheckpoint(dir, "write")
	if sha1 == "" {
		t.Fatal("expected non-empty SHA for first checkpoint")
	}
	if diff1 == nil {
		t.Fatal("expected non-nil diff for first checkpoint")
	}

	// Verify file1 is in first diff
	found1 := false
	for _, f := range diff1.Files {
		if strings.Contains(f.Path, "file1.txt") {
			found1 = true
		}
	}
	if !found1 {
		t.Error("expected file1.txt in first diff")
	}

	// Second checkpoint: create file2
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	sha2, diff2 := createGitCheckpoint(dir, "write")
	if sha2 == "" {
		t.Fatal("expected non-empty SHA for second checkpoint")
	}
	if sha2 == sha1 {
		t.Error("second SHA should differ from first")
	}
	if diff2 == nil {
		t.Fatal("expected non-nil diff for second checkpoint")
	}

	// Verify second diff only shows file2, NOT file1
	hasFile1 := false
	hasFile2 := false
	for _, f := range diff2.Files {
		if strings.Contains(f.Path, "file1.txt") {
			hasFile1 = true
		}
		if strings.Contains(f.Path, "file2.txt") {
			hasFile2 = true
		}
	}
	if hasFile1 {
		t.Error("second diff should NOT contain file1.txt (already committed in first checkpoint)")
	}
	if !hasFile2 {
		t.Error("second diff should contain file2.txt")
	}
}
