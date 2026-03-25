package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aotgithub "github.com/uncworks/aot/internal/github"
)

// WebhookHandler handles incoming GitHub webhook events.
type WebhookHandler struct {
	secret         string
	allowedRepos   []string
	githubProvider aotgithub.TokenProvider
	k8sClient      client.Client
	namespace      string
	// httpClient is used for fetching file content from the GitHub API.
	// Defaults to http.DefaultClient if nil.
	httpClient *http.Client
	// ciAutofix handles CI failure auto-fix (nil if disabled)
	ciAutofix *CIAutofix
}

// NewWebhookHandler creates a new WebhookHandler reading configuration from
// environment variables:
//   - GITHUB_WEBHOOK_SECRET: shared secret for HMAC-SHA256 signature validation
//   - GITHUB_WEBHOOK_REPOS: comma-separated allowlist of "owner/repo" strings
//
// ctx should be a server-lifetime context so that CIAutofix timer callbacks
// respect shutdown. The GitHub token for fetching file content is provided via
// the TokenProvider.
func NewWebhookHandler(ctx context.Context, k8sClient client.Client, namespace string, provider aotgithub.TokenProvider) *WebhookHandler {
	var repos []string
	if raw := os.Getenv("GITHUB_WEBHOOK_REPOS"); raw != "" {
		for _, r := range strings.Split(raw, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				repos = append(repos, r)
			}
		}
	}

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		slog.Warn("GITHUB_WEBHOOK_SECRET not set — webhook signature validation is disabled")
	}

	maxRetries := 3
	if v := os.Getenv("CI_AUTOFIX_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRetries = n
		}
	}

	return &WebhookHandler{
		secret:         secret,
		allowedRepos:   repos,
		githubProvider: provider,
		k8sClient:      k8sClient,
		namespace:      namespace,
		ciAutofix:      NewCIAutofix(ctx, k8sClient, namespace, provider, maxRetries),
	}
}

// httpDo is a convenience that uses the configured HTTP client or the default.
func (wh *WebhookHandler) httpDo(req *http.Request) (*http.Response, error) {
	c := wh.httpClient
	if c == nil {
		c = http.DefaultClient
	}
	return c.Do(req)
}

// ServeHTTP implements http.Handler for the GitHub webhook endpoint.
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit body to 10 MB to prevent abuse.
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Require HMAC signature validation. If no secret is configured the
	// endpoint is fail-closed (401) so that unconfigured deployments don't
	// accept unauthenticated webhook triggers.
	if wh.secret == "" {
		http.Error(w, "webhook secret not configured", http.StatusUnauthorized)
		return
	}
	sig := r.Header.Get("X-Hub-Signature-256")
	if !validateSignature(body, sig, wh.secret) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")

	// Handle check_run events for CI autofix
	if eventType == "check_run" && wh.ciAutofix != nil {
		triggered, err := wh.ciAutofix.HandleCheckRunEvent(r.Context(), body)
		if err != nil {
			slog.Error("CI autofix handler error", "err", err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "ci_autofix_triggered": triggered})
		return
	}

	// Only handle push events beyond this point.
	if eventType != "push" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "message": "ignored event type"})
		return
	}

	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	repo := payload.Repository.FullName
	if !wh.isRepoAllowed(repo) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "message": "repo not in allowlist"})
		return
	}

	// Collect unique .cs.md file paths from all commits.
	specFiles := wh.collectSpecFiles(payload.Commits)

	ref := payload.Ref
	// Extract branch name from "refs/heads/<branch>".
	branch := ref
	if strings.HasPrefix(ref, "refs/heads/") {
		branch = strings.TrimPrefix(ref, "refs/heads/")
	}

	created := 0
	for _, path := range specFiles {
		content, err := wh.fetchFileContent(r.Context(), repo, path, payload.After)
		if err != nil {
			slog.Error("webhook: failed to fetch file", "repo", repo, "path", path, "sha", payload.After, "err", err)
			continue
		}

		if err := wh.createAgentRun(r.Context(), repo, path, branch, content); err != nil {
			slog.Error("webhook: failed to create AgentRun", "repo", repo, "path", path, "err", err)
			continue
		}
		created++
	}

	w.WriteHeader(http.StatusOK)
	resp := map[string]interface{}{
		"ok":      true,
		"created": created,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// validateSignature checks the HMAC-SHA256 signature from the X-Hub-Signature-256 header.
func validateSignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	// GitHub sends "sha256=<hex>".
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigHex := strings.TrimPrefix(signature, "sha256=")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(sigBytes, expected)
}

// isRepoAllowed returns true if the given "owner/repo" is in the allowlist,
// or if the allowlist is empty (all repos allowed).
func (wh *WebhookHandler) isRepoAllowed(repo string) bool {
	if len(wh.allowedRepos) == 0 {
		return true
	}
	for _, allowed := range wh.allowedRepos {
		if strings.EqualFold(allowed, repo) {
			return true
		}
	}
	return false
}

// collectSpecFiles extracts unique .cs.md file paths from commit added/modified lists.
func (wh *WebhookHandler) collectSpecFiles(commits []commitInfo) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, c := range commits {
		for _, f := range append(c.Added, c.Modified...) {
			if strings.HasSuffix(f, ".cs.md") {
				if _, ok := seen[f]; !ok {
					seen[f] = struct{}{}
					result = append(result, f)
				}
			}
		}
	}
	return result
}

// fetchFileContent retrieves a file's raw content from the GitHub API at the given SHA.
func (wh *WebhookHandler) fetchFileContent(ctx context.Context, repo, path, sha string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", repo, path, sha)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	if wh.githubProvider != nil {
		if token, tokenErr := wh.githubProvider.Token(ctx); tokenErr == nil && token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := wh.httpDo(req)
	if err != nil {
		return "", fmt.Errorf("github api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	return string(data), nil
}

// createAgentRun creates an AgentRun CRD with the given spec content.
func (wh *WebhookHandler) createAgentRun(ctx context.Context, repo, path, branch, content string) error {
	name, err := generateRunName()
	if err != nil {
		return fmt.Errorf("generate name: %w", err)
	}

	crd := &aotv1alpha1.AgentRun{}
	crd.Name = name
	crd.Namespace = wh.namespace
	crd.Spec = aotv1alpha1.AgentRunSpec{
		SpecContent: content,
		SpecSource:  fmt.Sprintf("webhook:github:%s/%s", repo, path),
		Prompt:      fmt.Sprintf("Execute the CodeSpeak spec from %s/%s", repo, path),
		Repos: []aotv1alpha1.Repository{
			{
				URL:    fmt.Sprintf("https://github.com/%s.git", repo),
				Branch: branch,
			},
		},
	}
	crd.Status.Phase = aotv1alpha1.AgentRunPhasePending
	crd.Status.Message = "Queued via GitHub webhook"

	if err := wh.k8sClient.Create(ctx, crd); err != nil {
		return fmt.Errorf("create agentrun CRD: %w", err)
	}
	slog.Info("webhook: created AgentRun", "run", name, "repo", repo, "path", path)
	return nil
}

// --- GitHub push event payload types ---

type pushPayload struct {
	Ref        string       `json:"ref"`
	After      string       `json:"after"`
	Repository repoInfo     `json:"repository"`
	Commits    []commitInfo `json:"commits"`
}

type repoInfo struct {
	FullName string `json:"full_name"`
}

type commitInfo struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Removed  []string `json:"removed"`
}
