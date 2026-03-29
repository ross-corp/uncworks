// Package softserve provides a client for interacting with a soft-serve Git server.
package softserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client manages repositories on a soft-serve instance via SSH.
type Client struct {
	// SSHAddr is the SSH address of the soft-serve server (e.g., "soft-serve.aot.svc:23231").
	SSHAddr string
	// KeyPath is the path to the SSH private key for admin access.
	KeyPath string
}

// sshCmd runs an SSH command against soft-serve and returns stdout.
func (c *Client) sshCmd(args ...string) (string, error) {
	parts := strings.SplitN(c.SSHAddr, ":", 2)
	host := parts[0]
	port := "23231"
	if len(parts) == 2 {
		port = parts[1]
	}

	// When running as a nonroot UID in a Kubernetes pod, the SSH key mounted
	// from a Secret is owned by root (UID 0). The ssh binary refuses to use
	// keys that aren't owned by the current user. Copy the key to a temp file
	// owned by the current process so the permission check passes.
	keyPath, cleanup, err := ensureKeyOwnership(c.KeyPath)
	if err != nil {
		return "", fmt.Errorf("prepare SSH key: %w", err)
	}
	defer cleanup()

	cmdArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", port,
		host,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("ssh", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh command failed: %w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ensureKeyOwnership copies the SSH key to a temp file with correct ownership
// so the ssh binary accepts it. Returns the path to use and a cleanup function.
// Falls back to the original key path on any failure (best-effort); unexpected
// failures are logged so that permission issues remain visible.
func ensureKeyOwnership(keyPath string) (string, func(), error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		// Key may already be accessible (correct owner); fall through silently.
		return keyPath, func() {}, nil
	}
	tmp, err := os.CreateTemp("", "ss-key-*")
	if err != nil {
		slog.Warn("ensureKeyOwnership: create temp file failed, using original key", "err", err)
		return keyPath, func() {}, nil
	}
	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		slog.Warn("ensureKeyOwnership: chmod temp file failed, using original key", "err", err)
		_ = os.Remove(tmp.Name())
		return keyPath, func() {}, nil
	}
	if _, err := tmp.Write(data); err != nil {
		slog.Warn("ensureKeyOwnership: write temp file failed, using original key", "err", err)
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return keyPath, func() {}, nil
	}
	_ = tmp.Close()
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

// CreateRepo creates a new repository in soft-serve.
func (c *Client) CreateRepo(name string) error {
	if _, err := c.sshCmd("repo", "create", name); err != nil {
		return fmt.Errorf("create repo %q: %w", name, err)
	}
	return nil
}

// DeleteRepo deletes a repository from soft-serve.
func (c *Client) DeleteRepo(name string) error {
	if _, err := c.sshCmd("repo", "delete", name); err != nil {
		return fmt.Errorf("delete repo %q: %w", name, err)
	}
	return nil
}

// ListRepos lists all repositories in soft-serve.
func (c *Client) ListRepos() ([]string, error) {
	out, err := c.sshCmd("repo", "list")
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// RepoExists checks if a repository exists in soft-serve.
func (c *Client) RepoExists(name string) (bool, error) {
	repos, err := c.ListRepos()
	if err != nil {
		return false, fmt.Errorf("check repo exists %q: %w", name, err)
	}
	for _, r := range repos {
		if r == name {
			return true, nil
		}
	}
	return false, nil
}

// CloneURL returns the SSH clone URL for a repository.
func (c *Client) CloneURL(name string) string {
	return fmt.Sprintf("ssh://%s/%s.git", c.SSHAddr, name)
}

// ScaffoldProject represents the initial files for a new project repo.
type ScaffoldProject struct {
	Name     string
	Packages []string // devbox packages
}

// ScaffoldAndPush creates a temp repo, adds scaffold files, and pushes to soft-serve.
func (c *Client) ScaffoldAndPush(scaffold ScaffoldProject) error {
	tmpDir, err := os.MkdirTemp("", "aot-scaffold-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// git init
	if err := c.gitExec(tmpDir, "init", "-b", "main"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if err := c.gitExec(tmpDir, "config", "user.email", "aot@uncworks.io"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}
	if err := c.gitExec(tmpDir, "config", "user.name", "AOT"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}

	// Write devbox.json
	devboxJSON := map[string]interface{}{
		"packages": scaffold.Packages,
	}
	if scaffold.Packages == nil {
		devboxJSON["packages"] = []string{}
	}
	devboxBytes, _ := json.MarshalIndent(devboxJSON, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.json"), devboxBytes, 0644); err != nil {
		return fmt.Errorf("write devbox.json: %w", err)
	}

	// Write openspec config
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "specs"), 0755); err != nil {
		return fmt.Errorf("mkdir openspec/specs: %w", err)
	}
	openspecYAML := fmt.Sprintf("name: %s\nschema: spec-driven\n", scaffold.Name)
	if err := os.WriteFile(filepath.Join(tmpDir, "openspec", "openspec.yaml"), []byte(openspecYAML), 0644); err != nil {
		return fmt.Errorf("write openspec.yaml: %w", err)
	}

	// Write embedded scaffold files (openspec skills, .pi/ directory, etc.)
	if err := writeScaffoldFiles(tmpDir); err != nil {
		return fmt.Errorf("write scaffold files: %w", err)
	}

	// Write .devcontainer/devcontainer.json
	if err := os.MkdirAll(filepath.Join(tmpDir, ".devcontainer"), 0755); err != nil {
		return fmt.Errorf("mkdir .devcontainer: %w", err)
	}
	devcontainer := map[string]interface{}{
		"name":              scaffold.Name,
		"postCreateCommand": "devbox install",
	}
	dcBytes, _ := json.MarshalIndent(devcontainer, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, ".devcontainer", "devcontainer.json"), dcBytes, 0644); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}

	// git add + commit
	if err := c.gitExec(tmpDir, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := c.gitExec(tmpDir, "commit", "-m", "scaffold project"); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// git push
	remoteURL := c.CloneURL(scaffold.Name)
	if err := c.gitExec(tmpDir, "remote", "add", "origin", remoteURL); err != nil {
		return fmt.Errorf("git remote add: %w", err)
	}
	if err := c.gitExec(tmpDir, "push", "-u", "origin", "main"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// ReadFile reads a file from a repo at HEAD.
func (c *Client) ReadFile(repoName, filePath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "aot-read-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	remoteURL := c.CloneURL(repoName)
	if err := c.gitExec(tmpDir, "clone", "--depth", "1", remoteURL, "."); err != nil {
		return "", fmt.Errorf("clone: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, filePath))
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filePath, err)
	}
	return string(data), nil
}

// WriteFile writes a file to a repo, commits, and pushes.
func (c *Client) WriteFile(repoName, filePath, content, commitMsg string) error {
	tmpDir, err := os.MkdirTemp("", "aot-write-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	remoteURL := c.CloneURL(repoName)
	if err := c.gitExec(tmpDir, "clone", remoteURL, "."); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	if err := c.gitExec(tmpDir, "config", "user.email", "aot@uncworks.io"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}
	if err := c.gitExec(tmpDir, "config", "user.name", "AOT"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}

	fullPath := filepath.Join(tmpDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("mkdir parent dir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", filePath, err)
	}

	if err := c.gitExec(tmpDir, "add", filePath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := c.gitExec(tmpDir, "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if err := c.gitExec(tmpDir, "push"); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

// ListFiles lists files in a repo at HEAD.
func (c *Client) ListFiles(repoName string) ([]string, error) {
	tmpDir, err := os.MkdirTemp("", "aot-list-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	remoteURL := c.CloneURL(repoName)
	if err := c.gitExec(tmpDir, "clone", "--depth", "1", remoteURL, "."); err != nil {
		return nil, fmt.Errorf("clone: %w", err)
	}

	cmd := exec.Command("git", "ls-files")
	cmd.Dir = tmpDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

func (c *Client) gitExec(dir string, args ...string) error {
	keyPath, cleanup, _ := ensureKeyOwnership(c.KeyPath)
	defer cleanup()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR", keyPath),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}
