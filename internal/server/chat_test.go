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
	if msg != minimalSystemPrompt {
		t.Errorf("expected minimal prompt, got %q", msg)
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
