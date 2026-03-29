package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	aotgithub "github.com/uncworks/aot/internal/github"
)

// maxSpecBodyBytes caps the request body for spec push operations at 1 MB.
const maxSpecBodyBytes = 1 << 20

// GitHubClient communicates with the GitHub Contents API.
type GitHubClient struct {
	provider   aotgithub.TokenProvider
	httpClient *http.Client
}

// NewGitHubClient creates a GitHubClient using the given TokenProvider.
func NewGitHubClient(provider aotgithub.TokenProvider) *GitHubClient {
	return &GitHubClient{
		provider:   provider,
		httpClient: &http.Client{},
	}
}

// getToken retrieves the current token from the provider.
func (g *GitHubClient) getToken(ctx context.Context) (string, error) {
	if g.provider == nil {
		return "", fmt.Errorf("GITHUB_TOKEN not configured")
	}
	return g.provider.Token(ctx)
}

// --- request / response types ---

type pushRequest struct {
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Message string `json:"message"`
}

type pushResponse struct {
	SHA string `json:"sha"`
}

type pullResponse struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- GitHub API types ---

type ghContentsResponse struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

type ghPutRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
}

type ghPutResponse struct {
	Content struct {
		SHA string `json:"sha"`
	} `json:"content"`
}

// RegisterHandlers registers the GitHub integration REST endpoints on the given mux.
func (g *GitHubClient) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/specs/push", g.handlePush)
	mux.HandleFunc("GET /api/v1/specs/pull", g.handlePull)
}

func (g *GitHubClient) handlePush(w http.ResponseWriter, r *http.Request) {
	token, err := g.getToken(r.Context())
	if err != nil {
		slog.Error("github token unavailable", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "GitHub integration not configured"})
		return
	}

	var req pushRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body: " + err.Error()})
		return
	}
	if req.Repo == "" || req.Path == "" || req.Content == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "repo, path, content, and message are required"})
		return
	}

	owner, repo, err := splitRepo(req.Repo)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, req.Path)

	// Check if file already exists so we can supply the sha for an update.
	existingSHA := ""
	getReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, apiURL, nil)
	setAuthHeaders(getReq, token)
	getResp, err := g.httpClient.Do(getReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to reach GitHub API: " + err.Error()})
		return
	}
	defer func() { _ = getResp.Body.Close() }()

	if getResp.StatusCode == http.StatusOK {
		var existing ghContentsResponse
		if err := json.NewDecoder(getResp.Body).Decode(&existing); err == nil {
			existingSHA = existing.SHA
		}
	} else {
		// Drain body so the connection can be reused.
		_, _ = io.Copy(io.Discard, getResp.Body)
	}

	// Build the PUT payload.
	putBody := ghPutRequest{
		Message: req.Message,
		Content: base64.StdEncoding.EncodeToString([]byte(req.Content)),
		SHA:     existingSHA,
	}
	bodyBytes, err := json.Marshal(putBody)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "marshal request: " + err.Error()})
		return
	}

	putReq, err := http.NewRequestWithContext(r.Context(), http.MethodPut, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create request: " + err.Error()})
		return
	}
	setAuthHeaders(putReq, token)
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := g.httpClient.Do(putReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to reach GitHub API: " + err.Error()})
		return
	}
	defer func() { _ = putResp.Body.Close() }()

	if err := checkGitHubError(putResp); err != nil {
		writeJSON(w, err.statusCode, errorResponse{Error: err.message})
		return
	}

	var ghResp ghPutResponse
	if err := json.NewDecoder(putResp.Body).Decode(&ghResp); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to decode GitHub response: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, pushResponse{SHA: ghResp.Content.SHA})
}

func (g *GitHubClient) handlePull(w http.ResponseWriter, r *http.Request) {
	token, err := g.getToken(r.Context())
	if err != nil {
		slog.Error("github token unavailable", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "GitHub integration not configured"})
		return
	}

	repoParam := r.URL.Query().Get("repo")
	pathParam := r.URL.Query().Get("path")
	if repoParam == "" || pathParam == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "repo and path query params are required"})
		return
	}

	owner, repo, err := splitRepo(repoParam)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, pathParam)

	getReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, apiURL, nil)
	setAuthHeaders(getReq, token)

	getResp, err := g.httpClient.Do(getReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to reach GitHub API: " + err.Error()})
		return
	}
	defer func() { _ = getResp.Body.Close() }()

	if err := checkGitHubError(getResp); err != nil {
		writeJSON(w, err.statusCode, errorResponse{Error: err.message})
		return
	}

	var ghResp ghContentsResponse
	if err := json.NewDecoder(getResp.Body).Decode(&ghResp); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to decode GitHub response: " + err.Error()})
		return
	}

	// GitHub returns base64-encoded content with newlines; decode it.
	cleaned := strings.ReplaceAll(ghResp.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to decode file content: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, pullResponse{Content: string(decoded), SHA: ghResp.SHA})
}

// --- helpers ---

func setAuthHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// splitRepo parses "owner/repo" into its two parts.
func splitRepo(fullRepo string) (string, string, error) {
	parts := strings.SplitN(fullRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be in owner/repo format, got %q", fullRepo)
	}
	return parts[0], parts[1], nil
}

type ghError struct {
	statusCode int
	message    string
}

func (e *ghError) Error() string { return e.message }

// checkGitHubError inspects a GitHub API response and returns a typed error
// for non-2xx status codes.
func checkGitHubError(resp *http.Response) *ghError {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return &ghError{statusCode: http.StatusNotFound, message: "repository or file not found"}
	case http.StatusUnauthorized:
		return &ghError{statusCode: http.StatusUnauthorized, message: "GitHub authentication failed: invalid or expired token"}
	case http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return &ghError{statusCode: http.StatusTooManyRequests, message: "GitHub API rate limit exceeded"}
		}
		return &ghError{statusCode: http.StatusForbidden, message: "GitHub access forbidden: " + string(body)}
	default:
		return &ghError{statusCode: resp.StatusCode, message: fmt.Sprintf("GitHub API error (%d): %s", resp.StatusCode, string(body))}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
