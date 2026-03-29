// test/stubs/litellm_test.go — unit tests for the LiteLLMStub helper.
package stubs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiteLLMStub_DefaultResponse(t *testing.T) {
	stub := NewLiteLLMStub(t)

	resp, err := http.Post(stub.URL()+"/chat/completions", "application/json",
		bytes.NewBufferString(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cr CompletionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cr))
	assert.Equal(t, "chatcmpl-test", cr.ID)
	require.Len(t, cr.Choices, 1)
	assert.Equal(t, "done", cr.Choices[0].Message.Content)
	assert.Equal(t, "assistant", cr.Choices[0].Message.Role)
	assert.Equal(t, "stop", cr.Choices[0].FinishReason)
}

func TestLiteLLMStub_RecordsRequests(t *testing.T) {
	stub := NewLiteLLMStub(t)

	body := `{"model":"test-model","messages":[{"role":"user","content":"what is 2+2?"}]}`
	resp, err := http.Post(stub.URL()+"/chat/completions", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, 1, stub.RequestCount())
	require.Len(t, stub.Requests, 1)
	assert.Equal(t, "test-model", stub.Requests[0].Model)
	require.Len(t, stub.Requests[0].Messages, 1)
	assert.Equal(t, "user", stub.Requests[0].Messages[0].Role)
	assert.Equal(t, "what is 2+2?", stub.Requests[0].Messages[0].Content)
}

func TestLiteLLMStub_SequencedResponses(t *testing.T) {
	stub := NewLiteLLMStub(t,
		DefaultCompletion("first"),
		DefaultCompletion("second"),
		DefaultCompletion("third"),
	)

	post := func() string {
		resp, err := http.Post(stub.URL()+"/chat/completions", "application/json",
			bytes.NewBufferString(`{"model":"m","messages":[]}`))
		require.NoError(t, err)
		defer resp.Body.Close()
		var cr CompletionResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&cr))
		require.Len(t, cr.Choices, 1)
		return cr.Choices[0].Message.Content
	}

	assert.Equal(t, "first", post())
	assert.Equal(t, "second", post())
	assert.Equal(t, "third", post())
	// exhausted — last response repeats
	assert.Equal(t, "third", post())
	assert.Equal(t, "third", post())

	assert.Equal(t, 5, stub.RequestCount())
}

func TestLiteLLMStub_FallsBackToLastWhenExhausted(t *testing.T) {
	stub := NewLiteLLMStub(t,
		DefaultCompletion("only"),
	)

	for i := 0; i < 3; i++ {
		resp, err := http.Post(stub.URL()+"/chat/completions", "application/json",
			bytes.NewBufferString(`{"model":"m","messages":[]}`))
		require.NoError(t, err)
		var cr CompletionResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&cr))
		resp.Body.Close()
		assert.Equal(t, "only", cr.Choices[0].Message.Content)
	}
	assert.Equal(t, 3, stub.RequestCount())
}

func TestDefaultCompletion_Shape(t *testing.T) {
	c := DefaultCompletion("hello world")

	assert.Equal(t, "chatcmpl-test", c.ID)
	assert.Equal(t, "chat.completion", c.Object)
	require.Len(t, c.Choices, 1)
	assert.Equal(t, 0, c.Choices[0].Index)
	assert.Equal(t, "assistant", c.Choices[0].Message.Role)
	assert.Equal(t, "hello world", c.Choices[0].Message.Content)
	assert.Equal(t, "stop", c.Choices[0].FinishReason)
}
