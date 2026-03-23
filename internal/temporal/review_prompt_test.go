package temporal

import (
	"strings"
	"testing"
)

func TestBuildManageReviewPrompt_FirstAttempt(t *testing.T) {
	prompt := buildManageReviewPrompt(
		"add-auth",
		"+func NewAuthMiddleware() {}\n-// old code",
		"### Requirement: Auth middleware\nThe system SHALL...",
		"[assistant] I added the auth middleware.\n[tool:write] completed",
		"", // no previous feedback
	)

	if !strings.Contains(prompt, "add-auth") {
		t.Error("prompt should contain change name")
	}
	if !strings.Contains(prompt, "NewAuthMiddleware") {
		t.Error("prompt should contain git diff")
	}
	if !strings.Contains(prompt, "Auth middleware") {
		t.Error("prompt should contain spec content")
	}
	if !strings.Contains(prompt, "I added the auth middleware") {
		t.Error("prompt should contain implement log")
	}
	if strings.Contains(prompt, "Previous Review Feedback") {
		t.Error("first attempt should NOT have previous feedback section")
	}
}

func TestBuildManageReviewPrompt_RetryAttempt(t *testing.T) {
	prompt := buildManageReviewPrompt(
		"add-auth",
		"+func NewAuthMiddleware() {}",
		"### Requirement: Auth middleware",
		"[assistant] Fixed the issues",
		"The middleware was missing error handling. Add proper error returns.",
	)

	if !strings.Contains(prompt, "Previous Review Feedback") {
		t.Error("retry attempt should have previous feedback section")
	}
	if !strings.Contains(prompt, "missing error handling") {
		t.Error("previous feedback should be included")
	}
}

func TestBuildManageReviewPrompt_TruncatesLongDiff(t *testing.T) {
	longDiff := strings.Repeat("+ added line\n", 1000)
	prompt := buildManageReviewPrompt("test", longDiff, "spec", "log", "")

	if len(prompt) > 20000 {
		t.Errorf("prompt too long: %d chars (diff should be truncated)", len(prompt))
	}
	if !strings.Contains(prompt, "truncated") {
		t.Error("long diff should show truncation marker")
	}
}

func TestBuildManageReviewPrompt_ContainsInstructions(t *testing.T) {
	prompt := buildManageReviewPrompt("test", "diff", "spec", "log", "")

	if !strings.Contains(prompt, "ask_user") {
		t.Error("prompt should mention ask_user for escalation")
	}
	if !strings.Contains(prompt, "pass") && !strings.Contains(prompt, "criteria") {
		t.Error("prompt should describe the expected JSON output format")
	}
}
