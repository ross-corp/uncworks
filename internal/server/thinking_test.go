package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseThinkingFromLines_InProgressMessage(t *testing.T) {
	lines := []string{
		`{"type":"session","timestamp":"2025-01-01T00:00:00Z"}`,
		`{"type":"agent_start"}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Hello "}}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"world"}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.True(t, result.Thinking)
	assert.Equal(t, "Hello world", result.Text)
	assert.Empty(t, result.ToolName)
}

func TestParseThinkingFromLines_CompletedMessage(t *testing.T) {
	lines := []string{
		`{"type":"session","timestamp":"2025-01-01T00:00:00Z"}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Hello world"}}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Hello world"}]}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.False(t, result.Thinking)
}

func TestParseThinkingFromLines_EmptyFile(t *testing.T) {
	result := parseThinkingFromLines(nil)
	assert.False(t, result.Thinking)

	result = parseThinkingFromLines([]string{})
	assert.False(t, result.Thinking)
}

func TestParseThinkingFromLines_ToolUseInProgress(t *testing.T) {
	lines := []string{
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Let me read that file."}}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"tool_use","tool":{"name":"Read","input":{"path":"/foo"}}}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.True(t, result.Thinking)
	assert.Equal(t, "Let me read that file.", result.Text)
	assert.Equal(t, "Read", result.ToolName)
}

func TestParseThinkingFromLines_MultipleMessages_LastInProgress(t *testing.T) {
	// First message completed, second in progress.
	lines := []string{
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"First message"}}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"First message"}]}}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Second "}}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"message"}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.True(t, result.Thinking)
	assert.Equal(t, "Second message", result.Text)
}

func TestParseThinkingFromLines_MultipleMessages_AllCompleted(t *testing.T) {
	lines := []string{
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"First"}}`,
		`{"type":"message_end","message":{"role":"assistant"}}`,
		`{"type":"message_start"}`,
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Second"}}`,
		`{"type":"message_end","message":{"role":"assistant"}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.False(t, result.Thinking)
}

func TestParseThinkingFromLines_MessageStartOnly(t *testing.T) {
	// message_start with no text_delta yet — still "thinking" but empty text.
	lines := []string{
		`{"type":"message_start"}`,
	}

	result := parseThinkingFromLines(lines)
	assert.True(t, result.Thinking)
	assert.Empty(t, result.Text)
}

func TestParseThinkingFromLines_InvalidJSON(t *testing.T) {
	lines := []string{
		"not valid json",
		"also not json",
	}

	result := parseThinkingFromLines(lines)
	assert.False(t, result.Thinking)
}

func TestParseThinkingFromLines_MixedValidInvalid(t *testing.T) {
	lines := []string{
		`{"type":"message_start"}`,
		"invalid line",
		`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"thinking..."}}`,
	}

	result := parseThinkingFromLines(lines)
	assert.True(t, result.Thinking)
	assert.Equal(t, "thinking...", result.Text)
}
