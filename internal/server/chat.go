package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatHandler handles the POST /api/v1/chat/stream endpoint.
// NOTE: This endpoint has no authentication — same posture as /api/v1/classify.
// Flag for future auth layer when user authentication is added to the platform.
type ChatHandler struct {
	LiteLLMBaseURL string
	// HTTPClient has a connect timeout but no response timeout — streaming
	// responses may take arbitrarily long.
	HTTPClient *http.Client
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(liteLLMBaseURL string) *ChatHandler {
	return &ChatHandler{
		LiteLLMBaseURL: liteLLMBaseURL,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 60 * time.Second,
			},
			// No overall timeout — streaming responses run until completion
			// or client disconnect (handled via request context).
		},
	}
}

// RegisterChatHandlers registers chat routes on the given mux.
func (h *ChatHandler) RegisterChatHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/chat/stream", h.handleChatStream)
}

// chatMessage is a single turn in the conversation.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatContext is optional page-level context injected as a system message.
type chatContext struct {
	Type    string `json:"type"`    // "spec" | "run" | "project" | "general"
	Content string `json:"content"` // raw text, truncated server-side to 8 KB
	Label   string `json:"label"`   // human-readable identifier
}

// chatRequest is the JSON body for POST /api/v1/chat/stream.
type chatRequest struct {
	Messages []chatMessage `json:"messages"`
	Context  *chatContext  `json:"context,omitempty"`
}

const (
	maxChatBodyBytes = 64 * 1024 // 64 KB request body limit
	maxContextBytes  = 8 * 1024  // 8 KB context content limit

	// baseSystemPrompt is always included. The guidance section teaches the
	// copilot how to emit navigation and highlight actions that the UI interprets.
	baseSystemPrompt = `You are a helpful assistant for the uncworks AI agent platform.

You can guide the user through the UI using special action tokens that the app interprets:
- To navigate: include [NAV:/path] in your response (e.g. [NAV:/run/ar-abc123])
- To highlight a UI element: include [HIGHLIGHT:css-selector] (e.g. [HIGHLIGHT:.run-status-badge])

Only emit these tokens when it genuinely helps the user. They are stripped from the displayed text.
Be concise and specific.`
)

func (h *ChatHandler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	// Parse request.
	var req chatRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxChatBodyBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "messages must be non-empty"})
		return
	}

	// Truncate context content if too large.
	if req.Context != nil && len(req.Context.Content) > maxContextBytes {
		req.Context.Content = req.Context.Content[:maxContextBytes]
	}

	// Build system message.
	systemContent := buildChatSystemMessage(req.Context)

	// Assemble LiteLLM messages: system first, then conversation history.
	llmMessages := make([]map[string]string, 0, len(req.Messages)+1)
	llmMessages = append(llmMessages, map[string]string{
		"role":    "system",
		"content": systemContent,
	})
	for _, m := range req.Messages {
		llmMessages = append(llmMessages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	if h.LiteLLMBaseURL == "" {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "LITELLM_BASE_URL not configured"})
		return
	}

	// Build LiteLLM request with streaming enabled.
	llmReqBody := map[string]interface{}{
		"model":    "default",
		"messages": llmMessages,
		"stream":   true,
	}
	bodyBytes, err := json.Marshal(llmReqBody)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to build request"})
		return
	}

	url := strings.TrimRight(h.LiteLLMBaseURL, "/") + "/v1/chat/completions"
	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create upstream request"})
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	if masterKey := os.Getenv("LITELLM_MASTER_KEY"); masterKey != "" {
		upstreamReq.Header.Set("Authorization", "Bearer "+masterKey)
	}

	resp, err := h.HTTPClient.Do(upstreamReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: fmt.Sprintf("LLM request failed: %v", err)})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		// Write SSE error so the client can handle it gracefully.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = fmt.Fprintf(w, "data: {\"error\": \"LLM returned status %d: %s\"}\n\n", resp.StatusCode, string(body))
		return
	}

	// Stream SSE response.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		_, _ = fmt.Fprintf(w, "%s\n\n", line)
		if canFlush {
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(w, "data: {\"error\": \"stream read error\"}\n\n")
		if canFlush {
			flusher.Flush()
		}
		return
	}

	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	if canFlush {
		flusher.Flush()
	}
}

// buildChatSystemMessage constructs the system message, injecting context when present.
func buildChatSystemMessage(ctx *chatContext) string {
	if ctx == nil || ctx.Content == "" {
		return baseSystemPrompt
	}
	return fmt.Sprintf(
		"%s\n\nCurrent context (%s — %s):\n\n%s",
		baseSystemPrompt,
		ctx.Type,
		ctx.Label,
		ctx.Content,
	)
}
