package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/activity"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// PushChangesInput contains parameters for committing and pushing changes.
type PushChangesInput struct {
	AgentRunName  string
	PodIP         string
	RepoPath      string
	BranchName    string // e.g., "aot/ar-xxxxx"
	CommitMessage string
}

// PushChangesOutput contains the result of the push operation.
type PushChangesOutput struct {
	BranchName string
	CommitSHA  string
}

// PushChanges commits all workspace changes and pushes to a feature branch via the sidecar.
func (a *Activities) PushChanges(ctx context.Context, input PushChangesInput) (*PushChangesOutput, error) {
	activity.RecordHeartbeat(ctx, "pushing changes to feature branch")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(a.httpClient(), sidecarURL)

	// Configure git user (needed for commit)
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		`git config user.email "aot@uncworks.io" && git config user.name "AOT Pipeline"`); err != nil {
		return nil, fmt.Errorf("configure git user: %w", err)
	}

	// Create and checkout feature branch
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		fmt.Sprintf("git checkout -b %s", input.BranchName)); err != nil {
		return nil, fmt.Errorf("create branch %s: %w", input.BranchName, err)
	}

	// Stage all changes
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		"git add -A"); err != nil {
		return nil, fmt.Errorf("git add: %w", err)
	}

	// Check if there are changes to commit
	statusOut, _ := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		"git status --porcelain")
	if strings.TrimSpace(statusOut) == "" {
		return nil, fmt.Errorf("no changes to commit")
	}

	// Commit
	commitCmd := fmt.Sprintf("git commit -m %q", input.CommitMessage)
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, commitCmd); err != nil {
		return nil, fmt.Errorf("git commit: %w", err)
	}

	// Get commit SHA
	sha, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		"git rev-parse HEAD")
	if err != nil {
		return nil, fmt.Errorf("get commit sha: %w", err)
	}

	// Push to remote
	pushCmd := fmt.Sprintf("git push origin %s", input.BranchName)
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, pushCmd); err != nil {
		return nil, fmt.Errorf("git push: %w", err)
	}

	return &PushChangesOutput{
		BranchName: input.BranchName,
		CommitSHA:  strings.TrimSpace(sha),
	}, nil
}

// CreatePRInput contains parameters for creating a GitHub PR.
type CreatePRInput struct {
	RepoOwner    string // e.g., "uncworks"
	RepoName     string // e.g., "aot"
	BranchName   string
	BaseBranch   string
	Title        string
	Body         string
	AgentRunName string
}

// CreatePROutput contains the result of creating a PR.
type CreatePROutput struct {
	PRUrl    string
	PRNumber int
}

// CreatePR creates a GitHub pull request using the GitHub REST API.
// Requires GITHUB_TOKEN to be set in the worker environment.
func (a *Activities) CreatePR(ctx context.Context, input CreatePRInput) (*CreatePROutput, error) {
	activity.RecordHeartbeat(ctx, "creating GitHub PR")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not set in worker environment")
	}

	baseBranch := input.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Build the PR payload
	payload := map[string]string{
		"title": input.Title,
		"head":  input.BranchName,
		"base":  baseBranch,
		"body":  input.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal PR payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", input.RepoOwner, input.RepoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create PR request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	httpClient := a.httpClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read GitHub response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var prResp struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.Unmarshal(respBody, &prResp); err != nil {
		return nil, fmt.Errorf("parse PR response: %w", err)
	}

	return &CreatePROutput{
		PRUrl:    prResp.HTMLURL,
		PRNumber: prResp.Number,
	}, nil
}

// httpClient returns the Activities HTTP client, falling back to the default.
func (a *Activities) httpClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

// gitExec runs a git command via the sidecar's ExecCommand RPC with a 120s timeout.
func gitExec(ctx context.Context, client agentv1connect.AgentSidecarServiceClient, runID, repoPath, command string) (string, error) {
	resp, err := client.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        command,
		WorkingDir:     repoPath,
		TimeoutSeconds: 120,
	}))
	if err != nil {
		return "", fmt.Errorf("exec git command: %w", err)
	}
	if resp.Msg.ExitCode != 0 {
		return resp.Msg.Stdout, fmt.Errorf("git command exited with code %d: %s", resp.Msg.ExitCode, resp.Msg.Stderr)
	}
	return resp.Msg.Stdout, nil
}
