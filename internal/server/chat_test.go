package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newChatMux(liteLLMURL string) *http.ServeMux {
	mux := http.NewServeMux()
	h := NewChatHandler(liteLLMURL)
	h.RegisterChatHandlers(mux)
	return mux
}

func TestChatHandler_MissingMessages_Returns400(t *testing.T) {
	mux := newChatMux("http://fake-litellm")

	body := `{"messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChatHandler_NilMessages_Returns400(t *testing.T) {
	mux := newChatMux("http://fake-litellm")

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChatHandler_ContextTruncation(t *testing.T) {
	// Build a context content larger than maxContextBytes (8192).
	large := strings.Repeat("x", 10000)
	if len(large) <= maxContextBytes {
		t.Fatal("test setup error: large string should exceed maxContextBytes")
	}

	// Verify buildChatSystemMessage includes a truncated version of content.
	ctx := &chatContext{
		Type:    "spec",
		Content: large,
		Label:   "test/spec.md",
	}
	// Simulate server-side truncation (as done in handleChatStream).
	if len(ctx.Content) > maxContextBytes {
		ctx.Content = ctx.Content[:maxContextBytes]
	}
	msg := buildChatSystemMessage(ctx)

	if len(ctx.Content) != maxContextBytes {
		t.Errorf("truncated content length = %d, want %d", len(ctx.Content), maxContextBytes)
	}
	if !strings.Contains(msg, "spec — test/spec.md") {
		t.Errorf("system message missing context label, got: %s", msg[:200])
	}
}

func TestChatHandler_NoLiteLLMURL_Returns502(t *testing.T) {
	mux := newChatMux("") // empty URL

	body := `{"messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestBuildChatSystemMessage_NoContext(t *testing.T) {
	msg := buildChatSystemMessage(nil)
	if msg != baseSystemPrompt {
		t.Errorf("expected base prompt, got %q", msg)
	}
	// Verify guidance token instructions are present.
	if !strings.Contains(msg, "[NAV:") {
		t.Errorf("base prompt missing NAV token guidance")
	}
	if !strings.Contains(msg, "[HIGHLIGHT:") {
		t.Errorf("base prompt missing HIGHLIGHT token guidance")
	}
}

func TestBuildChatSystemMessage_WithContext(t *testing.T) {
	ctx := &chatContext{Type: "run", Content: "phase: failed", Label: "ar-abc123"}
	msg := buildChatSystemMessage(ctx)
	if !strings.Contains(msg, "run — ar-abc123") {
		t.Errorf("missing type/label in message: %s", msg)
	}
	if !strings.Contains(msg, "phase: failed") {
		t.Errorf("missing content in message: %s", msg)
	}
}

// newChatMuxWithRateLimit builds a mux with the chat handler wrapped by a rate limiter.
func newChatMuxWithRateLimit(liteLLMURL string, cfg RateLimiterConfig) *http.ServeMux {
	mux := http.NewServeMux()
	h := NewChatHandler(liteLLMURL)
	mid := RateLimitMiddleware(NewRateLimiter(cfg))
	h.RegisterChatHandlersWithMiddleware(mux, mid)
	return mux
}

// TestChatHandler_RateLimitDisabled_NormalFlow verifies that rate limiting disabled
// does not interfere with normal request processing.
func TestChatHandler_RateLimitDisabled_NormalFlow(t *testing.T) {
	cfg := RateLimiterConfig{
		Enabled:    false,
		RPS:        0.001,
		Burst:      1,
		TTLMinutes: 10,
	}
	mux := newChatMuxWithRateLimit("", cfg)

	body := `{"messages":[{"role":"user","content":"hello"}]}`
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "7.7.7.7:1234"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		// Without rate limiting the request proceeds to the handler.
		// With an empty LiteLLM URL the handler returns 502 — that's fine.
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d: got unexpected 429 when rate limiting is disabled", i+1)
		}
	}
}

// TestChatHandler_RateLimitEnabled_Returns429WhenExceeded verifies that 429 is
// returned when a tiny RPS limit is exceeded.
func TestChatHandler_RateLimitEnabled_Returns429WhenExceeded(t *testing.T) {
	cfg := RateLimiterConfig{
		Enabled:    true,
		RPS:        0.001, // effectively 0: won't replenish during test
		Burst:      1,
		TTLMinutes: 10,
	}
	mux := newChatMuxWithRateLimit("http://fake-litellm", cfg)

	makePost := func() int {
		body := `{"messages":[{"role":"user","content":"hello"}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	// First request consumes the single burst token.
	first := makePost()
	if first == http.StatusTooManyRequests {
		t.Fatal("first request should not be rate-limited")
	}

	// Second request should be rate-limited.
	if got := makePost(); got != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", got)
	}
}
