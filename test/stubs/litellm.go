// test/stubs/litellm.go — httptest stub for LiteLLM /chat/completions endpoint.
package stubs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// CompletionRequest is the OpenAI-compatible request body.
type CompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse is a minimal OpenAI-compatible completion response.
type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
}

// Choice is a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// LiteLLMStub is a fake LiteLLM server for tests.
type LiteLLMStub struct {
	Server    *httptest.Server
	Requests  []CompletionRequest
	mu        sync.Mutex
	responses []CompletionResponse
	idx       int
}

// NewLiteLLMStub creates a new LiteLLM stub with the given canned responses.
// If all responses are exhausted, the last one is repeated.
func NewLiteLLMStub(t *testing.T, responses ...CompletionResponse) *LiteLLMStub {
	t.Helper()
	if len(responses) == 0 {
		responses = []CompletionResponse{DefaultCompletion("done")}
	}
	s := &LiteLLMStub{responses: responses}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Server.Close)
	return s
}

// DefaultCompletion returns a minimal completion response with the given content.
func DefaultCompletion(content string) CompletionResponse {
	return CompletionResponse{
		ID:     "chatcmpl-test",
		Object: "chat.completion",
		Choices: []Choice{{
			Index:        0,
			Message:      Message{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
	}
}

// URL returns the stub server's base URL.
func (s *LiteLLMStub) URL() string { return s.Server.URL }

func (s *LiteLLMStub) handle(w http.ResponseWriter, r *http.Request) {
	var req CompletionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	s.Requests = append(s.Requests, req)
	idx := s.idx
	if s.idx < len(s.responses)-1 {
		s.idx++
	}
	resp := s.responses[idx]
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// RequestCount returns the number of requests received.
func (s *LiteLLMStub) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Requests)
}
