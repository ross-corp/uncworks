// Package hydration implements the init-container logic for provisioning
// git worktrees and devbox environments in agent pods.
package hydration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoConfig describes a single repository to hydrate.
type RepoConfig struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// Config holds the hydration configuration from environment variables.
type Config struct {
	Repos        []RepoConfig
	WorkspaceDir string
	DevboxConfig string
}

// ConfigFromEnv creates a Config from environment variables.
// Multi-repo: reads AOT_REPOS (JSON array of {url, branch, path}).
// Single-repo fallback: reads AOT_REPO_URL + AOT_BRANCH for backward compat.
func ConfigFromEnv() *Config {
	workspace := os.Getenv("AOT_WORKSPACE_DIR")
	if workspace == "" {
		workspace = "/workspace"
	}

	config := &Config{
		WorkspaceDir: workspace,
		DevboxConfig: os.Getenv("AOT_DEVBOX_CONFIG"),
	}

	// Try multi-repo env var first
	if reposJSON := os.Getenv("AOT_REPOS"); reposJSON != "" {
		var repos []RepoConfig
		if err := json.Unmarshal([]byte(reposJSON), &repos); err == nil && len(repos) > 0 {
			config.Repos = repos
			return config
		}
	}

	// Fallback to single-repo env vars
	if repoURL := os.Getenv("AOT_REPO_URL"); repoURL != "" {
		config.Repos = []RepoConfig{
			{URL: repoURL, Branch: os.Getenv("AOT_BRANCH")},
		}
	}

	return config
}

// Hydrator provisions the workspace for an agent run.
type Hydrator struct {
	config *Config
	runner CommandRunner
}

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

// ExecRunner runs real OS commands.
type ExecRunner struct{}

func (r *ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// NewHydrator creates a new Hydrator.
func NewHydrator(config *Config, runner CommandRunner) *Hydrator {
	if runner == nil {
		runner = &ExecRunner{}
	}
	return &Hydrator{config: config, runner: runner}
}

// Run executes the full hydration sequence for all repos.
func (h *Hydrator) Run(ctx context.Context) error {
	for i, repo := range h.config.Repos {
		repoPath := repo.Path
		if repoPath == "" {
			repoPath = repoNameFromURL(repo.URL)
		}

		bareDir := filepath.Join(h.config.WorkspaceDir, ".bare", repoPath)
		worktreeDir := filepath.Join(h.config.WorkspaceDir, "src", repoPath)

		if err := h.cloneRepo(ctx, repo.URL, bareDir); err != nil {
			return fmt.Errorf("clone repo %d (%s): %w", i, repo.URL, err)
		}

		if err := h.createWorktree(ctx, bareDir, worktreeDir, repo.Branch); err != nil {
			return fmt.Errorf("create worktree %d (%s): %w", i, repo.URL, err)
		}
	}

	if h.config.DevboxConfig != "" {
		if err := h.setupDevbox(ctx); err != nil {
			return fmt.Errorf("setup devbox: %w", err)
		}
	}

	return nil
}

func (h *Hydrator) cloneRepo(ctx context.Context, repoURL, bareDir string) error {
	if _, err := os.Stat(bareDir); err == nil {
		return nil // Already cloned
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check bare dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(bareDir), 0o755); err != nil {
		return fmt.Errorf("create bare parent dir: %w", err)
	}

	args := []string{"clone", "--bare", repoURL, bareDir}
	_, err := h.runner.Run(ctx, h.config.WorkspaceDir, "git", args...)
	return err
}

func (h *Hydrator) createWorktree(ctx context.Context, bareDir, worktreeDir, branch string) error {
	if branch == "" {
		// Detect default branch from the bare repo's HEAD
		out, err := h.runner.Run(ctx, bareDir, "git", "symbolic-ref", "--short", "HEAD")
		if err == nil && out != "" {
			branch = out
		} else {
			branch = "main"
		}
	}

	// Create a new worktree branch for the agent
	worktreeBranch := fmt.Sprintf("aot/%s", branch)
	_, err := h.runner.Run(ctx, bareDir, "git", "worktree", "add", "-b", worktreeBranch, worktreeDir, branch)
	return err
}

func (h *Hydrator) setupDevbox(ctx context.Context) error {
	worktreeDir := h.PrimaryWorktreePath()

	// Check if devbox.json exists in the worktree
	devboxPath := filepath.Join(worktreeDir, h.config.DevboxConfig)
	if _, err := os.Stat(devboxPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("devbox config not found: %s", devboxPath)
		}
		return fmt.Errorf("check devbox config: %w", err)
	}

	// Install devbox packages
	_, err := h.runner.Run(ctx, worktreeDir, "devbox", "install")
	return err
}

// PrimaryWorktreePath returns the path to the first repo's worktree.
func (h *Hydrator) PrimaryWorktreePath() string {
	if len(h.config.Repos) == 0 {
		return filepath.Join(h.config.WorkspaceDir, "src")
	}
	repoPath := h.config.Repos[0].Path
	if repoPath == "" {
		repoPath = repoNameFromURL(h.config.Repos[0].URL)
	}
	return filepath.Join(h.config.WorkspaceDir, "src", repoPath)
}

// WorktreePath returns the path to the created worktree (backward compat alias).
func (h *Hydrator) WorktreePath() string {
	return h.PrimaryWorktreePath()
}

// repoNameFromURL derives a directory name from a git URL.
// e.g. "https://github.com/org/foo.git" → "foo"
func repoNameFromURL(repoURL string) string {
	// Try parsing as URL
	if u, err := url.Parse(repoURL); err == nil && u.Path != "" {
		base := filepath.Base(u.Path)
		return strings.TrimSuffix(base, ".git")
	}
	// Fallback: strip .git suffix from whatever the string is
	base := filepath.Base(repoURL)
	return strings.TrimSuffix(base, ".git")
}
