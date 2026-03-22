package hydration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// MockRunner records and replays commands for testing.
type MockRunner struct {
	commands []RecordedCommand
	results  map[string]MockResult
}

type RecordedCommand struct {
	Dir  string
	Name string
	Args []string
}

type MockResult struct {
	Output string
	Err    error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		results: make(map[string]MockResult),
	}
}

func (m *MockRunner) On(name string, result MockResult) {
	m.results[name] = result
}

func (m *MockRunner) Run(_ context.Context, dir string, name string, args ...string) (string, error) {
	m.commands = append(m.commands, RecordedCommand{Dir: dir, Name: name, Args: args})
	key := name + " " + strings.Join(args, " ")

	// Check for exact match first
	if r, ok := m.results[key]; ok {
		return r.Output, r.Err
	}
	// Check for command name only
	if r, ok := m.results[name]; ok {
		return r.Output, r.Err
	}
	return "", nil
}

func (m *MockRunner) CommandCount() int {
	return len(m.commands)
}

func (m *MockRunner) LastCommand() RecordedCommand {
	if len(m.commands) == 0 {
		return RecordedCommand{}
	}
	return m.commands[len(m.commands)-1]
}

func TestHydrator_CloneAndWorktree(t *testing.T) {
	runner := NewMockRunner()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		WorkspaceDir: t.TempDir(),
	}

	h := NewHydrator(config, runner)
	ctx := context.Background()

	if err := h.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if runner.CommandCount() != 2 {
		t.Fatalf("expected 2 commands, got %d", runner.CommandCount())
	}

	// Verify clone command
	clone := runner.commands[0]
	if clone.Name != "git" {
		t.Errorf("expected git, got %s", clone.Name)
	}
	if clone.Args[0] != "clone" || clone.Args[1] != "--bare" {
		t.Errorf("expected bare clone, got %v", clone.Args)
	}

	// Verify worktree command
	wt := runner.commands[1]
	if wt.Args[0] != "worktree" || wt.Args[1] != "add" {
		t.Errorf("expected worktree add, got %v", wt.Args)
	}
	if wt.Args[2] != "-b" || wt.Args[3] != "aot/main" {
		t.Errorf("expected branch aot/main, got %v", wt.Args)
	}
}

func TestHydrator_DefaultBranch(t *testing.T) {
	runner := NewMockRunner()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git"}}, // Empty branch should detect from HEAD or fall back to "main"
		WorkspaceDir: t.TempDir(),
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Commands: clone, symbolic-ref (detect default branch), worktree add
	if runner.CommandCount() != 3 {
		t.Fatalf("expected 3 commands, got %d", runner.CommandCount())
	}

	// symbolic-ref returns "" from mock → falls back to "main"
	wt := runner.commands[2]
	if wt.Args[5] != "main" {
		t.Errorf("expected default branch main, got %s", wt.Args[5])
	}
}

func TestHydrator_DefaultBranchFromHEAD(t *testing.T) {
	runner := NewMockRunner()
	// Mock git symbolic-ref to return "master"
	runner.On("git symbolic-ref --short HEAD", MockResult{Output: "master"})
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git"}},
		WorkspaceDir: t.TempDir(),
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wt := runner.commands[2]
	if wt.Args[5] != "master" {
		t.Errorf("expected branch master from HEAD, got %s", wt.Args[5])
	}
	if wt.Args[3] != "aot/master" {
		t.Errorf("expected worktree branch aot/master, got %s", wt.Args[3])
	}
}

func TestHydrator_CloneFailure(t *testing.T) {
	runner := NewMockRunner()
	runner.On("git", MockResult{Err: fmt.Errorf("clone failed")})

	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		WorkspaceDir: t.TempDir(),
	}

	h := NewHydrator(config, runner)
	err := h.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "clone repo") {
		t.Errorf("expected clone error, got: %v", err)
	}
}

func TestHydrator_WorktreePath(t *testing.T) {
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/myrepo.git"}},
		WorkspaceDir: "/workspace",
	}
	h := NewHydrator(config, NewMockRunner())
	if h.WorktreePath() != "/workspace/myrepo" {
		t.Errorf("expected /workspace/myrepo, got %s", h.WorktreePath())
	}

	// With explicit path
	config2 := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/myrepo.git", Path: "custom"}},
		WorkspaceDir: "/workspace",
	}
	h2 := NewHydrator(config2, NewMockRunner())
	if h2.WorktreePath() != "/workspace/custom" {
		t.Errorf("expected /workspace/custom, got %s", h2.WorktreePath())
	}

	// With no repos — fallback
	config3 := &Config{WorkspaceDir: "/workspace"}
	h3 := NewHydrator(config3, NewMockRunner())
	if h3.WorktreePath() != "/workspace" {
		t.Errorf("expected /workspace, got %s", h3.WorktreePath())
	}
}

func TestHydrator_WriteSpec(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
		SpecContent:  "# MyConverter\n\nConverts CSV to JSON.",
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify spec file was written
	specPath := filepath.Join(tmpDir, "spec", "main.cs.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec file: %v", err)
	}
	if string(data) != "# MyConverter\n\nConverts CSV to JSON." {
		t.Errorf("unexpected spec content: %q", string(data))
	}

	// Verify codespeak.json was written
	configPath := filepath.Join(tmpDir, "codespeak.json")
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read codespeak.json: %v", err)
	}
	if !strings.Contains(string(data), `"specs": ["spec/main.cs.md"]`) {
		t.Errorf("unexpected codespeak.json: %q", string(data))
	}
}

func TestHydrator_NoSpec(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify no spec files were written
	specPath := filepath.Join(tmpDir, "spec", "main.cs.md")
	if _, err := os.Stat(specPath); !os.IsNotExist(err) {
		t.Error("spec file should not exist when SpecContent is empty")
	}
}

func TestHydrator_SpecOnlyRun(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		WorkspaceDir: tmpDir,
		SpecContent:  "# TestSpec",
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// No git commands should have been run
	if runner.CommandCount() != 0 {
		t.Errorf("expected 0 commands for spec-only run, got %d", runner.CommandCount())
	}

	// Spec file should still be written
	specPath := filepath.Join(tmpDir, "spec", "main.cs.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec file: %v", err)
	}
	if string(data) != "# TestSpec" {
		t.Errorf("unexpected spec content: %q", string(data))
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AOT_REPO_URL", "https://github.com/test/repo.git")
	t.Setenv("AOT_BRANCH", "develop")
	t.Setenv("AOT_WORKSPACE_DIR", "/custom/workspace")
	t.Setenv("AOT_DEVBOX_CONFIG", "devbox.json")

	config := ConfigFromEnv()
	if len(config.Repos) != 1 || config.Repos[0].URL != "https://github.com/test/repo.git" {
		t.Errorf("got Repos %+v", config.Repos)
	}
	if config.Repos[0].Branch != "develop" {
		t.Errorf("got Branch %q", config.Repos[0].Branch)
	}
	if config.WorkspaceDir != "/custom/workspace" {
		t.Errorf("got WorkspaceDir %q", config.WorkspaceDir)
	}
}

// --- Manifest generation tests ---

func TestHydrator_GenerateManifest_MultiRepo(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/frontend.git", Branch: "main"},
			{URL: "https://github.com/org/backend.git", Branch: "develop", Path: "api"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Read and parse uncspace.yaml
	data, err := os.ReadFile(filepath.Join(tmpDir, "uncspace.yaml"))
	if err != nil {
		t.Fatalf("read uncspace.yaml: %v", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse uncspace.yaml: %v", err)
	}

	if len(manifest.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(manifest.Repos))
	}

	// First repo: path derived from URL
	if manifest.Repos[0].Path != "frontend" {
		t.Errorf("repo[0].Path = %q, want frontend", manifest.Repos[0].Path)
	}
	if manifest.Repos[0].URL != "https://github.com/org/frontend.git" {
		t.Errorf("repo[0].URL = %q", manifest.Repos[0].URL)
	}
	if manifest.Repos[0].Branch != "main" {
		t.Errorf("repo[0].Branch = %q", manifest.Repos[0].Branch)
	}

	// Second repo: explicit path
	if manifest.Repos[1].Path != "api" {
		t.Errorf("repo[1].Path = %q, want api", manifest.Repos[1].Path)
	}
}

func TestHydrator_GenerateManifest_SingleRepo(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/mono.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "uncspace.yaml"))
	if err != nil {
		t.Fatalf("read uncspace.yaml: %v", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(manifest.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(manifest.Repos))
	}
	if manifest.Repos[0].Path != "mono" {
		t.Errorf("repo.Path = %q, want mono", manifest.Repos[0].Path)
	}
}

func TestHydrator_GenerateManifest_ZeroRepos(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "uncspace.yaml"))
	if err != nil {
		t.Fatalf("read uncspace.yaml: %v", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(manifest.Repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(manifest.Repos))
	}
}

func TestHydrator_GenerateManifest_DevboxDetection(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Create a fake devbox.json in the worktree path that would exist after clone
	worktree1 := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(worktree1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree1, "devbox.json"), []byte(`{"packages":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// repo2 has no devbox.json
	worktree2 := filepath.Join(tmpDir, "repo2")
	if err := os.MkdirAll(worktree2, 0o755); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
			{URL: "https://github.com/org/repo2.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	// Call generateManifest directly (Run would try to clone)
	if err := h.generateManifest(); err != nil {
		t.Fatalf("generateManifest: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "uncspace.yaml"))
	if err != nil {
		t.Fatalf("read uncspace.yaml: %v", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if manifest.Devbox == nil {
		t.Fatal("expected devbox section in manifest")
	}
	if len(manifest.Devbox.Sources) != 1 {
		t.Fatalf("expected 1 devbox source, got %d", len(manifest.Devbox.Sources))
	}
	if manifest.Devbox.Sources[0].Path != "repo1/devbox.json" {
		t.Errorf("devbox source path = %q, want repo1/devbox.json", manifest.Devbox.Sources[0].Path)
	}
}

// --- Devbox composition tests ---

func TestHydrator_ComposeDevbox_MultipleConfigs(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Create fake devbox.json files in worktree paths
	for _, name := range []string{"repo1", "repo2"} {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(`{"packages":[]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
			{URL: "https://github.com/org/repo2.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.composeDevbox(context.Background()); err != nil {
		t.Fatalf("composeDevbox: %v", err)
	}

	// Read root devbox.json
	data, err := os.ReadFile(filepath.Join(tmpDir, "devbox.json"))
	if err != nil {
		t.Fatalf("read devbox.json: %v", err)
	}

	var devbox DevboxInclude
	if err := json.Unmarshal(data, &devbox); err != nil {
		t.Fatalf("parse devbox.json: %v", err)
	}

	if len(devbox.Include) != 2 {
		t.Fatalf("expected 2 includes, got %d", len(devbox.Include))
	}
	if devbox.Include[0] != "repo1/devbox.json" {
		t.Errorf("include[0] = %q", devbox.Include[0])
	}
	if devbox.Include[1] != "repo2/devbox.json" {
		t.Errorf("include[1] = %q", devbox.Include[1])
	}

	// Verify devbox install was called from workspace root
	found := false
	for _, cmd := range runner.commands {
		if cmd.Name == "devbox" && len(cmd.Args) > 0 && cmd.Args[0] == "install" {
			if cmd.Dir != tmpDir {
				t.Errorf("devbox install dir = %q, want %q", cmd.Dir, tmpDir)
			}
			found = true
		}
	}
	if !found {
		t.Error("devbox install was not called")
	}
}

func TestHydrator_ComposeDevbox_SingleConfig(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(`{"packages":["go"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
			{URL: "https://github.com/org/repo2.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.composeDevbox(context.Background()); err != nil {
		t.Fatalf("composeDevbox: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "devbox.json"))
	if err != nil {
		t.Fatalf("read devbox.json: %v", err)
	}

	var devbox DevboxInclude
	if err := json.Unmarshal(data, &devbox); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(devbox.Include) != 1 {
		t.Fatalf("expected 1 include, got %d", len(devbox.Include))
	}
	if devbox.Include[0] != "repo1/devbox.json" {
		t.Errorf("include[0] = %q", devbox.Include[0])
	}
}

func TestHydrator_ComposeDevbox_NoConfigs(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.composeDevbox(context.Background()); err != nil {
		t.Fatalf("composeDevbox: %v", err)
	}

	// No root devbox.json should be created
	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.json")); !os.IsNotExist(err) {
		t.Error("devbox.json should not exist when no repos have devbox configs")
	}

	// No devbox install should have been called
	for _, cmd := range runner.commands {
		if cmd.Name == "devbox" {
			t.Error("devbox should not have been called")
		}
	}
}

func TestHydrator_ComposeDevbox_ExplicitOverride(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Create devbox.json in worktree
	dir := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(`{"packages":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
		DevboxConfig: "devbox.json", // Explicit config — should use setupDevbox, not composeDevbox
	}

	h := NewHydrator(config, runner)
	// Run will use setupDevbox (which checks in worktree), not composeDevbox
	// setupDevbox will look for devbox.json at PrimaryWorktreePath/devbox.json
	// which exists at repo1/devbox.json — should succeed
	err := h.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should NOT have a root /workspace/devbox.json with include directives
	rootDevbox := filepath.Join(tmpDir, "devbox.json")
	if _, err := os.Stat(rootDevbox); err == nil {
		t.Error("root devbox.json should not be created when DevboxConfig is explicitly set")
	}
}

// --- Idempotent worktree creation regression test ---

// TestHydrator_CreateWorktree_Idempotent verifies that running hydration twice
// (simulating a debug pod restart) succeeds without error. This is a regression
// test for the bug where createWorktree failed on "worktree already exists"
// when the pod restarted and the PVC still had the worktree from the first run.
func TestHydrator_CreateWorktree_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	bareDir := filepath.Join(tmpDir, ".bare", "myrepo")
	worktreeDir := filepath.Join(tmpDir, "myrepo")

	// Set up a real bare repo so worktree operations work.
	runner := &ExecRunner{}
	ctx := context.Background()

	// Create a bare repository with an initial commit.
	if err := os.MkdirAll(bareDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, bareDir, "git", "init", "--bare"); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	// Detect the default branch name (may be "master" or "main" depending on git config).
	defaultBranch, err := runner.Run(ctx, bareDir, "git", "symbolic-ref", "--short", "HEAD")
	if err != nil || defaultBranch == "" {
		defaultBranch = "master"
	}

	// We need at least one commit for worktree add to work.
	// Create a temp clone, make a commit, then push back to bare.
	tmpClone := filepath.Join(tmpDir, "_clone")
	if _, err := runner.Run(ctx, tmpDir, "git", "clone", bareDir, tmpClone); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	if _, err := runner.Run(ctx, tmpClone, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if _, err := runner.Run(ctx, tmpClone, "git", "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpClone, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, tmpClone, "git", "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runner.Run(ctx, tmpClone, "git", "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if _, err := runner.Run(ctx, tmpClone, "git", "push"); err != nil {
		t.Fatalf("git push: %v", err)
	}

	// Use the detected default branch so the test works regardless of git version.
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/myrepo.git", Branch: defaultBranch}},
		WorkspaceDir: tmpDir,
	}

	// First run: should create the worktree.
	h := NewHydrator(config, runner)
	if err := h.Run(ctx); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Verify worktree was created.
	if _, err := os.Stat(filepath.Join(worktreeDir, ".git")); err != nil {
		t.Fatalf("worktree .git should exist after first run: %v", err)
	}

	// Second run: should succeed without error (idempotent).
	h2 := NewHydrator(config, runner)
	if err := h2.Run(ctx); err != nil {
		t.Fatalf("second Run (idempotent) should succeed, got: %v", err)
	}

	// Verify the worktree still works (file is still there).
	if _, err := os.Stat(filepath.Join(worktreeDir, "README.md")); err != nil {
		t.Errorf("README.md should still exist after idempotent re-run: %v", err)
	}
}

// TestHydrator_CloneRepo_Idempotent verifies that cloneRepo is idempotent:
// if the bare dir already exists, it skips cloning.
func TestHydrator_CloneRepo_Idempotent(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()

	// Pre-create the bare dir to simulate a restart scenario.
	bareDir := filepath.Join(tmpDir, ".bare", "repo")
	if err := os.MkdirAll(bareDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Also pre-create the worktree dir with .git to skip worktree creation.
	worktreeDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: ../bare/repo"), 0o644); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/repo.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run should succeed when bare dir already exists: %v", err)
	}

	// No git commands should have been run — both clone and worktree were skipped.
	for _, cmd := range runner.commands {
		if cmd.Name == "git" {
			t.Errorf("expected no git commands (idempotent skip), got: git %v", cmd.Args)
		}
	}
}

// --- Token injection in clone URL ---

func TestInjectTokenInURL_GitHub(t *testing.T) {
	got := injectTokenInURL("https://github.com/org/repo.git", "ghp_test123")
	want := "https://x-access-token:ghp_test123@github.com/org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInjectTokenInURL_NonHTTPS(t *testing.T) {
	// SSH URLs should be returned unchanged
	got := injectTokenInURL("git@github.com:org/repo.git", "ghp_test123")
	if got != "git@github.com:org/repo.git" {
		t.Errorf("SSH URL should be unchanged, got %q", got)
	}
}

func TestHydrator_CloneRepo_WithToken(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/private-repo.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
	}

	// Set GITHUB_TOKEN for this test
	t.Setenv("GITHUB_TOKEN", "ghp_test_token")

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify clone command uses the token-injected URL
	clone := runner.commands[0]
	if clone.Name != "git" || clone.Args[0] != "clone" {
		t.Fatalf("expected git clone, got %s %v", clone.Name, clone.Args)
	}
	cloneURL := clone.Args[2] // args are: clone --bare <url> <dir>
	if !strings.Contains(cloneURL, "x-access-token:ghp_test_token@") {
		t.Errorf("clone URL should contain token, got %q", cloneURL)
	}
}

func TestHydrator_CloneRepo_WithoutToken(t *testing.T) {
	runner := NewMockRunner()
	tmpDir := t.TempDir()
	config := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/public-repo.git", Branch: "main"}},
		WorkspaceDir: tmpDir,
	}

	// Ensure GITHUB_TOKEN is not set
	t.Setenv("GITHUB_TOKEN", "")

	h := NewHydrator(config, runner)
	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify clone command uses the original URL (no token injection)
	clone := runner.commands[0]
	cloneURL := clone.Args[2]
	if cloneURL != "https://github.com/org/public-repo.git" {
		t.Errorf("clone URL should be unchanged, got %q", cloneURL)
	}
}

func TestHydrator_ComposeDevbox_InstallFailure(t *testing.T) {
	runner := NewMockRunner()
	runner.On("devbox", MockResult{Err: fmt.Errorf("devbox install failed")})
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(`{"packages":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Repos: []RepoConfig{
			{URL: "https://github.com/org/repo1.git", Branch: "main"},
		},
		WorkspaceDir: tmpDir,
	}

	h := NewHydrator(config, runner)
	err := h.composeDevbox(context.Background())
	if err == nil {
		t.Fatal("expected error from devbox install failure")
	}
	if !strings.Contains(err.Error(), "devbox install") {
		t.Errorf("expected devbox install error, got: %v", err)
	}
}
