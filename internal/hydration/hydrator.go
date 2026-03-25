// Package hydration implements the init-container logic for provisioning
// git worktrees and devbox environments in agent pods.
package hydration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
	SpecContent  string
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
		SpecContent:  os.Getenv("AOT_SPEC_CONTENT"),
	}

	// Try multi-repo env var first
	if reposJSON := os.Getenv("AOT_REPOS"); reposJSON != "" {
		var repos []RepoConfig
		if err := json.Unmarshal([]byte(reposJSON), &repos); err != nil {
			log.Printf("WARNING: failed to parse AOT_REPOS as JSON: %v, falling back to single-repo", err)
		} else if len(repos) > 0 {
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

// ManifestRepo describes a repo entry in uncspace.yaml.
type ManifestRepo struct {
	Path   string `yaml:"path"`
	URL    string `yaml:"url"`
	Branch string `yaml:"branch,omitempty"`
}

// DevboxSource describes a devbox.json location in uncspace.yaml.
type DevboxSource struct {
	Path string `yaml:"path"`
}

// Manifest represents the uncspace.yaml workspace manifest.
type Manifest struct {
	Repos  []ManifestRepo  `yaml:"repos,omitempty"`
	Devbox *DevboxManifest `yaml:"devbox,omitempty"`
}

// DevboxManifest describes devbox configuration in the manifest.
type DevboxManifest struct {
	Sources []DevboxSource `yaml:"sources,omitempty"`
}

// validateRepoPath rejects paths that could escape the workspace directory.
// It disallows absolute paths, paths containing "..", and paths that when
// cleaned resolve to strings starting with "..".
func validateRepoPath(path string) error {
	if path == "" {
		return nil // empty means "derive from URL"
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("repository path must be relative, got %q", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed in repository path %q", path)
	}
	if clean := filepath.Clean(path); strings.HasPrefix(clean, "..") {
		return fmt.Errorf("repository path would escape workspace: %q", path)
	}
	return nil
}

// Run executes the full hydration sequence for all repos.
func (h *Hydrator) Run(ctx context.Context) error {
	for i, repo := range h.config.Repos {
		if err := validateRepoPath(repo.Path); err != nil {
			return fmt.Errorf("repo %d: %w", i, err)
		}

		repoPath := repo.Path
		if repoPath == "" {
			repoPath = repoNameFromURL(repo.URL)
		}

		bareDir := filepath.Join(h.config.WorkspaceDir, ".bare", repoPath)
		worktreeDir := filepath.Join(h.config.WorkspaceDir, repoPath)

		if err := h.cloneRepo(ctx, repo.URL, bareDir); err != nil {
			return fmt.Errorf("clone repo %d (%s): %w", i, repo.URL, err)
		}

		if err := h.createWorktree(ctx, bareDir, worktreeDir, repo.Branch); err != nil {
			return fmt.Errorf("create worktree %d (%s): %w", i, repo.URL, err)
		}
	}

	if h.config.SpecContent != "" {
		if err := h.writeSpec(); err != nil {
			return fmt.Errorf("write spec: %w", err)
		}
	}

	// Generate workspace manifest after cloning, before devbox setup
	if err := h.generateManifest(); err != nil {
		return fmt.Errorf("generate manifest: %w", err)
	}

	// Generate devcontainer.json for VS Code Remote Containers (6.1)
	if err := h.writeDevcontainer(); err != nil {
		return fmt.Errorf("write devcontainer: %w", err)
	}

	// Create .aot directory structure for traces and logs (6.2, 6.3)
	for _, dir := range []string{
		filepath.Join(h.config.WorkspaceDir, ".aot", "traces"),
		filepath.Join(h.config.WorkspaceDir, ".aot", "logs"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Write run metadata (6.4)
	if err := h.writeMetadata(); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	// Devbox setup: use explicit config if set, otherwise auto-compose
	if h.config.DevboxConfig != "" {
		if err := h.setupDevbox(ctx); err != nil {
			return fmt.Errorf("setup devbox: %w", err)
		}
	} else if len(h.config.Repos) > 0 {
		if err := h.composeDevbox(ctx); err != nil {
			return fmt.Errorf("compose devbox: %w", err)
		}
	}

	return nil
}

// generateManifest writes /workspace/uncspace.yaml describing the workspace layout.
func (h *Hydrator) generateManifest() error {
	manifest := Manifest{}

	for _, repo := range h.config.Repos {
		repoPath := repo.Path
		if repoPath == "" {
			repoPath = repoNameFromURL(repo.URL)
		}
		relPath := repoPath

		manifest.Repos = append(manifest.Repos, ManifestRepo{
			Path:   relPath,
			URL:    repo.URL,
			Branch: repo.Branch,
		})

		// Check for devbox.json in the worktree
		worktreeDir := filepath.Join(h.config.WorkspaceDir, relPath)
		devboxPath := filepath.Join(worktreeDir, "devbox.json")
		if _, err := os.Stat(devboxPath); err == nil {
			if manifest.Devbox == nil {
				manifest.Devbox = &DevboxManifest{}
			}
			manifest.Devbox.Sources = append(manifest.Devbox.Sources, DevboxSource{
				Path: filepath.Join(relPath, "devbox.json"),
			})
		}
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(h.config.WorkspaceDir, "uncspace.yaml")
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("write uncspace.yaml: %w", err)
	}

	return nil
}

// DevboxInclude represents a root devbox.json with include directives.
type DevboxInclude struct {
	Include []string `json:"include"`
}

// composeDevbox scans repo worktrees for devbox.json files and generates a
// root /workspace/devbox.json with include directives, then runs devbox install.
func (h *Hydrator) composeDevbox(ctx context.Context) error {
	var includes []string

	for _, repo := range h.config.Repos {
		repoPath := repo.Path
		if repoPath == "" {
			repoPath = repoNameFromURL(repo.URL)
		}

		worktreeDir := filepath.Join(h.config.WorkspaceDir, repoPath)
		devboxPath := filepath.Join(worktreeDir, "devbox.json")
		if _, err := os.Stat(devboxPath); err == nil {
			includes = append(includes, filepath.Join(repoPath, "devbox.json"))
		}
	}

	if len(includes) == 0 {
		return nil
	}

	// Write root devbox.json with include directives
	devboxConfig := DevboxInclude{Include: includes}
	data, err := json.MarshalIndent(devboxConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devbox config: %w", err)
	}
	data = append(data, '\n')

	rootDevbox := filepath.Join(h.config.WorkspaceDir, "devbox.json")
	if err := os.WriteFile(rootDevbox, data, 0o644); err != nil {
		return fmt.Errorf("write root devbox.json: %w", err)
	}

	// Run devbox install from workspace root.
	// If devbox/nix aren't fully available, log warning but don't fail —
	// the agent can still work, it just won't have devbox-managed deps.
	_, err = h.runner.Run(ctx, h.config.WorkspaceDir, "devbox", "install")
	if err != nil {
		log.Printf("WARNING: devbox install failed: %v (agent will work without devbox deps)", err)
	}

	return nil
}

// writeSpec writes the CodeSpeak spec file and codespeak.json to the workspace.
func (h *Hydrator) writeSpec() error {
	specDir := filepath.Join(h.config.WorkspaceDir, "spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return fmt.Errorf("create spec dir: %w", err)
	}

	specPath := filepath.Join(specDir, "main.cs.md")
	if err := os.WriteFile(specPath, []byte(h.config.SpecContent), 0o644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}

	configJSON := `{"specs": ["spec/main.cs.md"]}` + "\n"
	configPath := filepath.Join(h.config.WorkspaceDir, "codespeak.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		return fmt.Errorf("write codespeak.json: %w", err)
	}

	return nil
}

// DevcontainerConfig represents a .devcontainer/devcontainer.json file.
type DevcontainerConfig struct {
	Name             string `json:"name"`
	Image            string `json:"image"`
	WorkspaceFolder  string `json:"workspaceFolder"`
	PostStartCommand string `json:"postStartCommand"`
	RemoteUser       string `json:"remoteUser"`
}

// writeDevcontainer generates /workspace/.devcontainer/devcontainer.json (6.1).
func (h *Hydrator) writeDevcontainer() error {
	dir := filepath.Join(h.config.WorkspaceDir, ".devcontainer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .devcontainer dir: %w", err)
	}

	config := DevcontainerConfig{
		Name:             "aot-run",
		Image:            "aot-agent:local",
		WorkspaceFolder:  "/workspace",
		PostStartCommand: "devbox install || true",
		RemoteUser:       "root",
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	data = append(data, '\n')

	return os.WriteFile(filepath.Join(dir, "devcontainer.json"), data, 0o644)
}

// RunMetadata captures run configuration for debugging and audit (6.4).
type RunMetadata struct {
	AgentRunID string `json:"agentRunId,omitempty"`
	Repos      string `json:"repos,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
	ModelTier  string `json:"modelTier,omitempty"`
}

// writeMetadata writes /workspace/.aot/metadata.json from environment variables (6.4).
func (h *Hydrator) writeMetadata() error {
	meta := RunMetadata{
		AgentRunID: os.Getenv("AOT_AGENT_RUN_ID"),
		Repos:      os.Getenv("AOT_REPOS"),
		Prompt:     os.Getenv("AOT_PROMPT"),
		ModelTier:  os.Getenv("AOT_MODEL_TIER"),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	data = append(data, '\n')

	metaDir := filepath.Join(h.config.WorkspaceDir, ".aot")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return fmt.Errorf("create .aot dir: %w", err)
	}

	return os.WriteFile(filepath.Join(metaDir, "metadata.json"), data, 0o644)
}

func (h *Hydrator) cloneRepo(ctx context.Context, repoURL, bareDir string) error {
	if _, err := os.Stat(bareDir); err == nil {
		// Directory exists — validate it's actually a git repo
		if _, gitErr := h.runner.Run(ctx, bareDir, "git", "rev-parse", "--git-dir"); gitErr != nil {
			// Broken or partial clone; remove and re-clone
			log.Printf("WARNING: removing broken bare clone at %s: %v", bareDir, gitErr)
			if rmErr := os.RemoveAll(bareDir); rmErr != nil {
				return fmt.Errorf("remove broken bare dir: %w", rmErr)
			}
		} else {
			return nil // Valid existing clone
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check bare dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(bareDir), 0o755); err != nil {
		return fmt.Errorf("create bare parent dir: %w", err)
	}

	// Inject GITHUB_TOKEN into clone URL for private repo authentication.
	// The init container receives GITHUB_TOKEN from a k8s Secret (scoped to init only).
	cloneURL := repoURL
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneURL = injectTokenInURL(repoURL, token)
	}

	args := []string{"clone", "--bare", cloneURL, bareDir}
	if _, err := h.runner.Run(ctx, h.config.WorkspaceDir, "git", args...); err != nil {
		// Clean up partial clone so retries start fresh
		_ = os.RemoveAll(bareDir)
		return err
	}
	return nil
}

// injectTokenInURL embeds a token into an HTTPS git URL for authentication.
// Only injects into github.com HTTPS URLs; returns the original URL unchanged
// for SSH URLs or any host not in the allowlist.
func injectTokenInURL(repoURL, token string) string {
	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme != "https" {
		return repoURL // SSH or unparseable — leave unchanged
	}
	if u.Host != "github.com" {
		return repoURL // Not in allowlist — refuse to embed token
	}
	u.User = url.UserPassword("x-access-token", token)
	return u.String()
}

func (h *Hydrator) createWorktree(ctx context.Context, bareDir, worktreeDir, branch string) error {
	// Idempotent: skip if worktree directory already exists (e.g., debug pod restart)
	if _, err := os.Stat(filepath.Join(worktreeDir, ".git")); err == nil {
		return nil
	}

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
		return h.config.WorkspaceDir
	}
	repoPath := h.config.Repos[0].Path
	if repoPath == "" {
		repoPath = repoNameFromURL(h.config.Repos[0].URL)
	}
	return filepath.Join(h.config.WorkspaceDir, repoPath)
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
