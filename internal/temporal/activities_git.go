package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	temporalsdk "go.temporal.io/sdk/temporal"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/activity"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
	aotgithub "github.com/uncworks/aot/internal/github"
)

// PushChangesInput contains parameters for committing and pushing changes.
type PushChangesInput struct {
	AgentRunName  string
	PodIP         string
	RepoPath      string
	BranchName    string // e.g., "aot/ar-xxxxx"
	CommitMessage string
	RepoURL       string // e.g., "https://github.com/org/repo.git" — used for authenticated push
	ChangeName    string // e.g., "git-push-and-pr" — used to locate proposal.md
}

// PushChangesOutput contains the result of the push operation.
type PushChangesOutput struct {
	BranchName      string
	CommitSHA       string
	DiffStat        string // output of `git diff --stat HEAD~1`
	ProposalContent string // contents of openspec/changes/{changeName}/proposal.md
}

// conventionalCommitRE validates the conventional commits specification.
// Format: type(optional-scope): description
var conventionalCommitRE = regexp.MustCompile(`^(feat|fix|refactor|docs|test|chore|perf|ci|build|revert)(\([^)]+\))?: .{1,72}$`)

// readAgentCommitMessage reads the commit message the agent wrote to /workspace/.aot/commit_message.txt.
// Returns the fallback message if the file is missing, empty, or not in conventional commits format.
func readAgentCommitMessage(ctx context.Context, sc agentv1connect.AgentSidecarServiceClient, runID, repoPath, fallback string) string {
	raw, err := gitExec(ctx, sc, runID, repoPath, "cat /workspace/.aot/commit_message.txt 2>/dev/null || echo ''")
	if err != nil {
		return fallback
	}
	msg := strings.TrimSpace(raw)
	if msg == "" || !conventionalCommitRE.MatchString(msg) {
		return fallback
	}
	return msg
}

// PushChanges commits all workspace changes and pushes to a feature branch via the sidecar.
// It injects the GitHub token into the remote URL for authentication, then restores the
// original URL after the push to avoid persisting credentials.
func (a *Activities) PushChanges(ctx context.Context, input PushChangesInput) (*PushChangesOutput, error) {
	activity.RecordHeartbeat(ctx, "pushing changes to feature branch")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(a.httpClient(), sidecarURL)

	// Configure git user (needed for commit)
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		`git config user.email "aot@uncworks.io" && git config user.name "AOT Pipeline"`); err != nil {
		return nil, fmt.Errorf("configure git user: %w", err)
	}

	// Create feature branch from current HEAD (which has all checkpoint commits)
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		fmt.Sprintf("git checkout -B %s", input.BranchName)); err != nil {
		return nil, fmt.Errorf("create branch %s: %w", input.BranchName, err)
	}

	// Check if HEAD is ahead of origin/main (has commits to squash)
	logOut, logErr := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		"git log --oneline origin/main..HEAD")
	var hasSquashed bool
	if logErr == nil && strings.TrimSpace(logOut) != "" {
		// There are commits to squash
		// Soft reset to merge base, keeping changes staged
		if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
			"git reset --soft $(git merge-base HEAD origin/main)"); err != nil {
			return nil, fmt.Errorf("git reset --soft for squash: %w", err)
		}
		// Check if there are any changes (staged or unstaged) after reset
		statusOut, _ := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
			"git status --porcelain")
		if strings.TrimSpace(statusOut) != "" {
			// Stage any remaining unstaged changes
			if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
				"git add -A"); err != nil {
				return nil, fmt.Errorf("git add after squash: %w", err)
			}
			// Commit the squashed changes using the agent-provided message when valid.
			commitMsg := readAgentCommitMessage(ctx, sc, input.AgentRunName, input.RepoPath, input.CommitMessage)
			commitCmd := fmt.Sprintf("git commit -m %q", commitMsg)
			if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, commitCmd); err != nil {
				return nil, fmt.Errorf("git commit after squash: %w", err)
			}
			hasSquashed = true
		}
		// If status is empty after reset, tree is clean, no commit needed
	}

	// If we didn't squash, stage and commit any unstaged changes
	if !hasSquashed {
		if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
			"git add -A"); err != nil {
			return nil, fmt.Errorf("git add: %w", err)
		}

		statusOut, _ := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
			"git status --porcelain")
		if strings.TrimSpace(statusOut) != "" {
			// There are unstaged changes — commit them using the agent-provided message when valid.
			commitMsg := readAgentCommitMessage(ctx, sc, input.AgentRunName, input.RepoPath, input.CommitMessage)
			commitCmd := fmt.Sprintf("git commit -m %q", commitMsg)
			if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, commitCmd); err != nil {
				return nil, fmt.Errorf("git commit: %w", err)
			}
		}
	}

	// Get commit SHA
	sha, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath,
		"git rev-parse HEAD")
	if err != nil {
		return nil, fmt.Errorf("get commit sha: %w", err)
	}

	// Capture diff stats for PR body
	diffStat, _ := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, "git diff --stat HEAD~1")

	// Read proposal.md for PR body (best-effort, ignore errors)
	var proposalContent string
	if input.ChangeName != "" {
		catCmd := fmt.Sprintf("cat openspec/changes/%s/proposal.md 2>/dev/null || echo ''", input.ChangeName)
		proposalContent, _ = gitExec(ctx, sc, input.AgentRunName, input.RepoPath, catCmd)
	}

	// Push to remote using git's http.extraHeader to pass the token so it never
	// appears in the remote URL (and thus not in Temporal history or git error logs).
	pushCmd := fmt.Sprintf("git push --force origin %s", input.BranchName)
	if a.GitHubProvider != nil && input.RepoURL != "" {
		token, tokenErr := a.GitHubProvider.Token(ctx)
		if tokenErr == nil && token != "" {
			// Ensure the remote URL is the plain (unauthenticated) URL.
			setURLCmd := fmt.Sprintf("git remote set-url origin %s", input.RepoURL)
			if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, setURLCmd); err != nil {
				return nil, fmt.Errorf("set remote URL: %w", err)
			}
			// Pass the credential via git -c http.<url>.extraHeader so the token
			// is injected at the transport layer and never embedded in the URL.
			authHeader := aotgithub.BasicAuthHeader(token)
			pushCmd = fmt.Sprintf(
				"git -c http.https://github.com/.extraHeader=%q push --force origin %s",
				"Authorization: "+authHeader,
				input.BranchName,
			)
		}
	}
	if _, err := gitExec(ctx, sc, input.AgentRunName, input.RepoPath, pushCmd); err != nil {
		return nil, fmt.Errorf("git push: %w", err)
	}

	return &PushChangesOutput{
		BranchName:      input.BranchName,
		CommitSHA:       strings.TrimSpace(sha),
		DiffStat:        strings.TrimSpace(diffStat),
		ProposalContent: strings.TrimSpace(proposalContent),
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
// Requires a GitHubProvider to be configured on the Activities struct.
func (a *Activities) CreatePR(ctx context.Context, input CreatePRInput) (*CreatePROutput, error) {
	activity.RecordHeartbeat(ctx, "creating GitHub PR")

	if a.GitHubProvider == nil {
		return nil, fmt.Errorf("no GitHub token provider configured")
	}
	token, err := a.GitHubProvider.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("get GitHub token for PR creation: %w", err)
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
		msg := fmt.Sprintf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
		// 4xx errors (except 429 Too Many Requests) will not succeed on retry.
		// Return a non-retryable ApplicationError to avoid burning the retry budget.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return nil, temporalsdk.NewNonRetryableApplicationError(msg, "GitHubAPIError", nil)
		}
		return nil, temporalsdk.NewApplicationError(msg, "GitHubAPIError")
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
