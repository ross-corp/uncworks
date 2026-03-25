// Package server implements the AOT API server handlers.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// ClassifyRunHandler handles the POST /api/v1/classify endpoint.
type ClassifyRunHandler struct {
	K8sClient      client.Client
	Namespace      string
	LiteLLMBaseURL string
	HTTPClient     *http.Client
}

// NewClassifyRunHandler creates a new ClassifyRunHandler.
func NewClassifyRunHandler(k8sClient client.Client, namespace, liteLLMBaseURL string) *ClassifyRunHandler {
	return &ClassifyRunHandler{
		K8sClient:      k8sClient,
		Namespace:      namespace,
		LiteLLMBaseURL: liteLLMBaseURL,
		HTTPClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// classifyRequest is the JSON body for the classify endpoint.
type classifyRequest struct {
	Prompt string   `json:"prompt"`
	Repos  []string `json:"repos"`
}

// classifyResponse is the JSON response for the classify endpoint.
type classifyResponse struct {
	Project      string   `json:"project"`
	Feature      string   `json:"feature"`
	FeatureIsNew bool     `json:"featureIsNew"`
	Tags         []string `json:"tags"`
}

// RegisterClassifyHandlers registers the classify and improve endpoints on the given mux.
func (h *ClassifyRunHandler) RegisterClassifyHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/classify", h.handleClassify)
	mux.HandleFunc("POST /api/v1/improve-text", h.handleImproveText)
}

func (h *ClassifyRunHandler) handleClassify(w http.ResponseWriter, r *http.Request) {
	var req classifyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if req.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "prompt is required"})
		return
	}

	// List all AgentRun CRDs to extract distinct project and feature labels.
	projects, features, err := h.extractExistingLabels(r.Context())
	if err != nil {
		slog.Warn("failed to list agent runs for classification", "err", err)
		// Continue with empty lists — the LLM can still suggest new values.
	}

	// Build the classification prompt.
	llmPrompt := buildClassificationPrompt(req.Prompt, req.Repos, projects, features)

	// Call LiteLLM for classification.
	resp, err := h.callLiteLLM(r.Context(), llmPrompt)
	if err != nil {
		slog.Error("classify LLM call failed", "err", err)
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: fmt.Sprintf("LLM classification failed: %s", err.Error())})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode classify response", "err", err)
	}
}

// extractExistingLabels lists all AgentRun CRDs and returns distinct project and feature label values.
func (h *ClassifyRunHandler) extractExistingLabels(ctx context.Context) (projects, features []string, err error) {
	var list aotv1alpha1.AgentRunList
	if err := h.K8sClient.List(ctx, &list, client.InNamespace(h.Namespace)); err != nil {
		return nil, nil, fmt.Errorf("list agentruns: %w", err)
	}

	projectSet := make(map[string]struct{})
	featureSet := make(map[string]struct{})

	for _, item := range list.Items {
		if item.Labels == nil {
			continue
		}
		if p, ok := item.Labels["aot.uncworks.io/project"]; ok && p != "" {
			projectSet[p] = struct{}{}
		}
		if f, ok := item.Labels["aot.uncworks.io/feature"]; ok && f != "" {
			featureSet[f] = struct{}{}
		}
	}

	for p := range projectSet {
		projects = append(projects, p)
	}
	for f := range featureSet {
		features = append(features, f)
	}
	return projects, features, nil
}

// buildClassificationPrompt constructs the LLM prompt for classification.
func buildClassificationPrompt(prompt string, repos, projects, features []string) string {
	return fmt.Sprintf(`Given this agent run request, suggest classification.
Prompt: "%s"
Repos: [%s]
Existing projects: [%s]
Existing features: [%s]

Return JSON only, no explanation:
{"project": "suggested-project", "feature": "feature-name-kebab-case", "featureIsNew": true/false, "tags": ["tag1", "tag2"]}`,
		prompt,
		strings.Join(repos, ", "),
		strings.Join(projects, ", "),
		strings.Join(features, ", "),
	)
}

// callLiteLLM calls the LiteLLM chat completion endpoint and parses a classifyResponse.
func (h *ClassifyRunHandler) callLiteLLM(ctx context.Context, prompt string) (*classifyResponse, error) {
	if h.LiteLLMBaseURL == "" {
		return nil, fmt.Errorf("LITELLM_BASE_URL not configured")
	}

	reqBody := map[string]interface{}{
		"model": "default",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 200,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	llmCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	url := strings.TrimRight(h.LiteLLMBaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(llmCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Include the LiteLLM master key if available.
	if masterKey := os.Getenv("LITELLM_MASTER_KEY"); masterKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+masterKey)
	}

	resp, err := h.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Parse the LiteLLM chat completion response.
	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &llmResp); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}
	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	content := strings.TrimSpace(llmResp.Choices[0].Message.Content)

	// Strip markdown code fences if present.
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result classifyResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse classification JSON %q: %w", content, err)
	}

	return &result, nil
}

// handleImproveText takes text (prompt or spec) and returns an AI-improved version.
func (h *ClassifyRunHandler) handleImproveText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
		Kind string `json:"kind"` // "prompt" or "spec"
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "text is required"})
		return
	}
	if req.Kind == "" {
		req.Kind = "prompt"
	}

	var systemPrompt string
	if req.Kind == "spec" {
		systemPrompt = `You are a spec editor. Improve the given specification to be clearer, more structured, and more actionable for an AI coding agent. Keep the same intent but:
- Add clear acceptance criteria
- Break vague requirements into specific, testable items
- Use markdown formatting with headers and bullet points
- Add edge cases the author may have missed
Return ONLY the improved spec text, no explanation.`
	} else {
		systemPrompt = `You are a prompt editor. Improve the given prompt to be clearer and more effective for an AI coding agent. Keep the same intent but:
- Be more specific about what to do
- Clarify success criteria
- Add relevant constraints or context hints
- Keep it concise
Return ONLY the improved prompt text, no explanation.`
	}

	if h.LiteLLMBaseURL == "" {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "LITELLM_BASE_URL not configured"})
		return
	}

	reqBody := map[string]interface{}{
		"model": "default",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": req.Text},
		},
		"max_tokens": 2000,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "marshal request"})
		return
	}

	llmCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	url := strings.TrimRight(h.LiteLLMBaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(llmCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if masterKey := os.Getenv("LITELLM_MASTER_KEY"); masterKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+masterKey)
	}

	resp, err := h.HTTPClient.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: fmt.Sprintf("LLM request failed: %v", err)})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: fmt.Sprintf("LLM error %d: %s", resp.StatusCode, string(body))})
		return
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 32768))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read LLM response"})
		return
	}

	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &llmResp); err != nil || len(llmResp.Choices) == 0 {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "invalid LLM response"})
		return
	}

	improved := strings.TrimSpace(llmResp.Choices[0].Message.Content)
	writeJSON(w, http.StatusOK, map[string]string{"improved": improved})
}
