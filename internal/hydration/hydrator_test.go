package hydration

import (
	"context"
	"fmt"
	"strings"
	"testing"
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
	if h.WorktreePath() != "/workspace/src/myrepo" {
		t.Errorf("expected /workspace/src/myrepo, got %s", h.WorktreePath())
	}

	// With explicit path
	config2 := &Config{
		Repos:        []RepoConfig{{URL: "https://github.com/org/myrepo.git", Path: "custom"}},
		WorkspaceDir: "/workspace",
	}
	h2 := NewHydrator(config2, NewMockRunner())
	if h2.WorktreePath() != "/workspace/src/custom" {
		t.Errorf("expected /workspace/src/custom, got %s", h2.WorktreePath())
	}

	// With no repos — fallback
	config3 := &Config{WorkspaceDir: "/workspace"}
	h3 := NewHydrator(config3, NewMockRunner())
	if h3.WorktreePath() != "/workspace/src" {
		t.Errorf("expected /workspace/src, got %s", h3.WorktreePath())
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
