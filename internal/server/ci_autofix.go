package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aotgithub "github.com/uncworks/aot/internal/github"
)

// checkRunPayload represents the GitHub check_run webhook payload.
type checkRunPayload struct {
	Action     string   `json:"action"`
	CheckRun   checkRun `json:"check_run"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type checkRun struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Conclusion string     `json:"conclusion"` // "success", "failure", "cancelled", etc.
	HeadSHA    string     `json:"head_sha"`
	CheckSuite checkSuite `json:"check_suite"`
	HTMLURL    string     `json:"html_url"`
}

type checkSuite struct {
	ID         int64  `json:"id"`
	HeadBranch string `json:"head_branch"`
}

// CIAutofix handles CI failure detection and auto-fix run creation.
type CIAutofix struct {
	K8sClient      client.Client
	Namespace      string
	GitHubProvider aotgithub.TokenProvider
	MaxRetries     int
	HTTPClient     *http.Client

	ctx          context.Context
	mu           sync.Mutex
	pendingFixes map[string]*time.Timer // SHA → debounce timer
}

// NewCIAutofix creates a new CI autofix handler.
// ctx should be a server-lifetime context so that timer callbacks respect shutdown.
func NewCIAutofix(ctx context.Context, k8s client.Client, ns string, ghProvider aotgithub.TokenProvider, maxRetries int) *CIAutofix {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &CIAutofix{
		K8sClient:      k8s,
		Namespace:      ns,
		GitHubProvider: ghProvider,
		MaxRetries:     maxRetries,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
		ctx:            ctx,
		pendingFixes:   make(map[string]*time.Timer),
	}
}

// HandleCheckRunEvent processes a check_run webhook payload.
// Returns true if a fix run was triggered or scheduled.
func (ci *CIAutofix) HandleCheckRunEvent(ctx context.Context, body []byte) (bool, error) {
	var payload checkRunPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("unmarshal check_run: %w", err)
	}

	// Only act on completed failures
	if payload.Action != "completed" {
		return false, nil
	}
	if payload.CheckRun.Conclusion != "failure" {
		// On success, update CI status for the branch
		if payload.CheckRun.Conclusion == "success" {
			ci.updateCIStatus(ctx, payload.CheckRun.CheckSuite.HeadBranch, "success")
		}
		return false, nil
	}

	branch := payload.CheckRun.CheckSuite.HeadBranch
	if !strings.HasPrefix(branch, "aot/") {
		return false, nil // not our branch
	}

	repo := payload.Repository.FullName
	sha := payload.CheckRun.HeadSHA
	checkSuiteID := payload.CheckRun.CheckSuite.ID

	slog.Info("detected CI failure", "check", payload.CheckRun.Name, "repo", repo, "branch", branch, "sha", sha[:min(8, len(sha))])

	// Check retry count
	attempts, err := ci.getFixAttemptCount(ctx, branch)
	if err != nil {
		slog.Error("failed to count fix attempts", "branch", branch, slog.Any("error", err))
	}
	if attempts >= ci.MaxRetries {
		slog.Warn("CI autofix: max retries reached", "maxRetries", ci.MaxRetries, "branch", branch)
		ci.postCircuitBreakerComment(ctx, repo, branch, attempts)
		return false, nil
	}

	// Debounce: coalesce multiple check_run failures for the same SHA
	ci.mu.Lock()
	if timer, exists := ci.pendingFixes[sha]; exists {
		timer.Stop()
	}
	ci.pendingFixes[sha] = time.AfterFunc(30*time.Second, func() {
		if ci.ctx.Err() != nil {
			return
		}
		ci.mu.Lock()
		delete(ci.pendingFixes, sha)
		ci.mu.Unlock()

		if err := ci.createFixRun(ci.ctx, repo, branch, sha, checkSuiteID, attempts+1); err != nil {
			slog.Error("failed to create fix run", "branch", branch, slog.Any("error", err))
		}
	})
	ci.mu.Unlock()

	return true, nil
}

// createFixRun spawns an AgentRun to fix CI failures on the given branch.
func (ci *CIAutofix) createFixRun(ctx context.Context, repoFullName, branch, sha string, checkSuiteID int64, attempt int) error {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo name: %s", repoFullName)
	}
	owner, repo := parts[0], parts[1]

	// Fetch CI logs
	ciLogs, err := ci.fetchAndCondenseCILogs(ctx, owner, repo, checkSuiteID)
	if err != nil {
		slog.Warn("failed to fetch CI logs, proceeding without", "branch", branch, slog.Any("error", err))
		ciLogs = "(CI logs unavailable)"
	}

	// Build the fix prompt
	prompt := fmt.Sprintf(`The CI checks failed on branch %s. Fix the failing checks.

## CI Error Output
%s

## Instructions
- Read the error messages carefully
- Fix only the issues identified in the CI output
- Do not make unrelated changes
- Run any test commands locally to verify before completing`, branch, ciLogs)

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Create the AgentRun CRD
	run := &aotv1alpha1.AgentRun{}
	run.GenerateName = "ar-"
	run.Namespace = ci.Namespace
	run.Spec = aotv1alpha1.AgentRunSpec{
		Backend:           aotv1alpha1.BackendPod,
		Repos:             []aotv1alpha1.Repository{{URL: repoURL, Branch: branch}},
		Prompt:            prompt,
		TTLSeconds:        1800,
		ModelTier:         "deepseek-v3.1",
		OrchestrationMode: aotv1alpha1.OrchestrationModeSpecDriven,
		SpecSource:        fmt.Sprintf("ci-autofix:%s/%s#%s", owner, repo, sha[:min(8, len(sha))]),
		AutoPush:          true,
		AutoPR:            false, // push to existing branch, don't create new PR
		Feature:           "ci-autofix",
	}

	// Set annotations for tracking
	run.Annotations = map[string]string{
		"aot.uncworks.io/pr-branch":      branch,
		"aot.uncworks.io/ci-fix-sha":     sha,
		"aot.uncworks.io/ci-fix-attempt": fmt.Sprintf("%d", attempt),
	}

	if err := ci.K8sClient.Create(ctx, run); err != nil {
		return fmt.Errorf("create fix AgentRun: %w", err)
	}

	slog.Info("CI autofix: created fix run", "run", run.Name, "branch", branch, "attempt", attempt, "maxRetries", ci.MaxRetries)
	return nil
}

// fetchAndCondenseCILogs fetches CI logs from GitHub Actions and condenses them.
func (ci *CIAutofix) fetchAndCondenseCILogs(ctx context.Context, owner, repo string, checkSuiteID int64) (string, error) {
	if ci.GitHubProvider == nil {
		return "", fmt.Errorf("no GitHub token provider")
	}
	token, err := ci.GitHubProvider.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	// Resolve check_suite to Actions run ID
	runID, err := ci.resolveActionsRunID(ctx, owner, repo, checkSuiteID, token)
	if err != nil {
		return "", fmt.Errorf("resolve run ID: %w", err)
	}

	// Fetch logs
	logsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d/logs", owner, repo, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := ci.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch logs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for logs", resp.StatusCode)
	}

	// Read zip (capped at 50MB)
	zipData, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return "", fmt.Errorf("read log zip: %w", err)
	}

	// Extract text from zip
	raw, err := extractTextFromZip(zipData)
	if err != nil {
		return "", fmt.Errorf("extract zip: %w", err)
	}

	return condenseCIErrors(raw), nil
}

// resolveActionsRunID maps a check_suite ID to a GitHub Actions workflow run ID.
func (ci *CIAutofix) resolveActionsRunID(ctx context.Context, owner, repo string, checkSuiteID int64, token string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs?check_suite_id=%d", owner, repo, checkSuiteID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := ci.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		WorkflowRuns []struct {
			ID int64 `json:"id"`
		} `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.WorkflowRuns) == 0 {
		return 0, fmt.Errorf("no workflow runs found for check_suite %d", checkSuiteID)
	}
	return result.WorkflowRuns[0].ID, nil
}

// extractTextFromZip extracts all text content from a zip archive.
func extractTextFromZip(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(io.LimitReader(rc, 1<<20)) // 1MB per file
		_ = rc.Close()
		if err != nil {
			continue
		}
		_, _ = fmt.Fprintf(&sb, "=== %s ===\n", f.Name)
		sb.Write(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// condenseCIErrors filters a raw CI log to extract error-relevant lines.
func condenseCIErrors(raw string) string {
	const maxLen = 8000
	errorIndicators := []string{
		"error", "Error", "ERROR",
		"FAIL", "fail", "Failed",
		"panic", "Panic",
		"undefined", "not found",
		"cannot", "Cannot",
		"expected", "unexpected",
		"warning", "Warning",
	}

	var relevant []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for _, indicator := range errorIndicators {
			if strings.Contains(trimmed, indicator) {
				relevant = append(relevant, trimmed)
				break
			}
		}
	}

	result := strings.Join(relevant, "\n")
	if len(result) > maxLen {
		// Middle-out truncation
		half := maxLen / 2
		result = result[:half] + "\n\n... (truncated) ...\n\n" + result[len(result)-half:]
	}
	if result == "" {
		// If no error lines found, return the last 4000 chars of raw output
		if len(raw) > 4000 {
			result = "... (showing last 4000 chars)\n" + raw[len(raw)-4000:]
		} else {
			result = raw
		}
	}
	return result
}

// getFixAttemptCount counts how many fix runs exist for a given branch.
func (ci *CIAutofix) getFixAttemptCount(ctx context.Context, branch string) (int, error) {
	var list aotv1alpha1.AgentRunList
	if err := ci.K8sClient.List(ctx, &list, client.InNamespace(ci.Namespace)); err != nil {
		return 0, err
	}

	count := 0
	for _, run := range list.Items {
		if run.Annotations != nil && run.Annotations["aot.uncworks.io/pr-branch"] == branch {
			if strings.HasPrefix(run.Spec.SpecSource, "ci-autofix:") {
				count++
			}
		}
	}
	return count, nil
}

// updateCIStatus updates the lastCIStatus on the most recent run for a branch.
func (ci *CIAutofix) updateCIStatus(ctx context.Context, branch string, status string) {
	if !strings.HasPrefix(branch, "aot/") {
		return
	}

	var list aotv1alpha1.AgentRunList
	if err := ci.K8sClient.List(ctx, &list, client.InNamespace(ci.Namespace)); err != nil {
		return
	}

	// Find the most recent run for this branch
	for i := len(list.Items) - 1; i >= 0; i-- {
		run := &list.Items[i]
		if run.Annotations != nil && run.Annotations["aot.uncworks.io/pr-branch"] == branch {
			run.Status.LastCIStatus = status
			if err := ci.K8sClient.Status().Update(ctx, run); err != nil {
				slog.Warn("failed to update CI status", "run", run.Name, slog.Any("error", err))
			}
			return
		}
	}
}

// postCircuitBreakerComment posts a comment on the PR when max retries are exhausted.
func (ci *CIAutofix) postCircuitBreakerComment(ctx context.Context, repoFullName, branch string, attempts int) {
	if ci.GitHubProvider == nil {
		return
	}
	token, tokenErr := ci.GitHubProvider.Token(ctx)
	if tokenErr != nil {
		return
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return
	}
	owner, repo := parts[0], parts[1]

	// Find PR number for this branch
	prNumber, err := ci.resolvePRNumber(ctx, owner, repo, branch, token)
	if err != nil {
		slog.Warn("failed to resolve PR number", "branch", branch, slog.Any("error", err))
		return
	}

	comment := fmt.Sprintf(
		"UNCWORKS CI Autofix has exhausted %d fix attempts for this PR. "+
			"The CI checks are still failing. Manual intervention is required.\n\n"+
			"The autofix agent attempted to resolve the failures but was unable to "+
			"produce changes that pass all CI checks within the retry limit.",
		attempts,
	)

	body, _ := json.Marshal(map[string]string{"body": comment})
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", owner, repo, prNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ci.HTTPClient.Do(req)
	if err != nil {
		slog.Warn("failed to post circuit breaker comment", slog.Any("error", err))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusCreated {
		slog.Info("CI autofix: posted circuit breaker comment", "prNumber", prNumber)
	}
}

// resolvePRNumber finds the PR number for a branch via GitHub API.
func (ci *CIAutofix) resolvePRNumber(ctx context.Context, owner, repo, branch, token string) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?head=%s:%s&state=open", owner, repo, owner, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := ci.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return 0, err
	}
	if len(prs) == 0 {
		return 0, fmt.Errorf("no open PR found for branch %s", branch)
	}
	return prs[0].Number, nil
}
