// Package hydration implements the init-container logic for provisioning
// git worktrees and devbox environments in agent pods.
package hydration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds the hydration configuration from environment variables.
type Config struct {
	RepoURL      string
	Branch       string
	WorkspaceDir string
	DevboxConfig string
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() *Config {
	workspace := os.Getenv("AOT_WORKSPACE_DIR")
	if workspace == "" {
		workspace = "/workspace"
	}
	return &Config{
		RepoURL:      os.Getenv("AOT_REPO_URL"),
		Branch:       os.Getenv("AOT_BRANCH"),
		WorkspaceDir: workspace,
		DevboxConfig: os.Getenv("AOT_DEVBOX_CONFIG"),
	}
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

// Run executes the full hydration sequence.
func (h *Hydrator) Run(ctx context.Context) error {
	if err := h.cloneRepo(ctx); err != nil {
		return fmt.Errorf("clone repo: %w", err)
	}

	if err := h.createWorktree(ctx); err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	if h.config.DevboxConfig != "" {
		if err := h.setupDevbox(ctx); err != nil {
			return fmt.Errorf("setup devbox: %w", err)
		}
	}

	return nil
}

func (h *Hydrator) cloneRepo(ctx context.Context) error {
	bareDir := filepath.Join(h.config.WorkspaceDir, ".bare")

	if _, err := os.Stat(bareDir); err == nil {
		return nil // Already cloned
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check bare dir: %w", err)
	}

	args := []string{"clone", "--bare", h.config.RepoURL, bareDir}
	_, err := h.runner.Run(ctx, h.config.WorkspaceDir, "git", args...)
	return err
}

func (h *Hydrator) createWorktree(ctx context.Context) error {
	bareDir := filepath.Join(h.config.WorkspaceDir, ".bare")
	worktreeDir := filepath.Join(h.config.WorkspaceDir, "src")

	branch := h.config.Branch
	if branch == "" {
		branch = "main"
	}

	// Create a new worktree branch for the agent
	worktreeBranch := fmt.Sprintf("aot/%s", branch)
	_, err := h.runner.Run(ctx, bareDir, "git", "worktree", "add", "-b", worktreeBranch, worktreeDir, branch)
	return err
}

func (h *Hydrator) setupDevbox(ctx context.Context) error {
	worktreeDir := filepath.Join(h.config.WorkspaceDir, "src")

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

// WorktreePath returns the path to the created worktree.
func (h *Hydrator) WorktreePath() string {
	return filepath.Join(h.config.WorkspaceDir, "src")
}
