package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// LLMJudgeInput contains parameters for the LLM judge activity.
type LLMJudgeInput struct {
	AgentRunName   string
	PodIP          string
	RepoPath       string
	Prompt         string // original task prompt
	LiteLLMBaseURL string
	LLMKey         string
	ModelTier      string // model to use for judging; defaults to "default-cloud"
}

// LLMJudgeOutput contains the result of the LLM judge activity.
type LLMJudgeOutput struct {
	Approved bool
	Reason   string
	GitDiff  string
}

// LLMJudgeChanges calls an LLM to review the agent's changes relative to the
// original prompt and returns an approval decision.
func (a *Activities) LLMJudgeChanges(ctx context.Context, input LLMJudgeInput) (LLMJudgeOutput, error) {
	activity.RecordHeartbeat(ctx, "starting LLM judge review")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// Collect git diff from workspace
	gitDiff, _ := execInSidecar(ctx, sc, input.AgentRunName, input.RepoPath,
		"git diff HEAD~1 2>/dev/null || git diff HEAD 2>/dev/null || echo 'no git diff available'")
	if gitDiff == "" {
		gitDiff = "no git diff available"
	}
	// Truncate large diffs to stay within LLM context limits
	if len(gitDiff) > 12000 {
		gitDiff = gitDiff[:12000] + "\n... (truncated)"
	}

	activity.RecordHeartbeat(ctx, "running LLM judge")

	// Build the review prompt
	reviewPrompt := buildLLMJudgePrompt(input.Prompt, gitDiff)

	model := input.ModelTier
	if model == "" || model == "default" {
		model = "default-cloud"
	}

	verdict, err := callChatCompletion(ctx, input.LiteLLMBaseURL, input.LLMKey, model, reviewPrompt)
	if err != nil {
		return LLMJudgeOutput{Approved: false, Reason: fmt.Sprintf("LLM judge error: %v", err), GitDiff: gitDiff}, nil
	}

	approved := verdict.Approved
	reason := verdict.Reason
	if reason == "" {
		if approved {
			reason = "LLM judge: changes look good"
		} else {
			reason = "LLM judge: changes do not meet the requirements"
		}
	}

	return LLMJudgeOutput{Approved: approved, Reason: reason, GitDiff: gitDiff}, nil
}

// llmJudgeVerdict is the structured verdict returned by the LLM.
type llmJudgeVerdict struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

// buildLLMJudgePrompt constructs the review prompt sent to the LLM.
func buildLLMJudgePrompt(originalPrompt, gitDiff string) string {
	return fmt.Sprintf(`You are a code review judge. An AI coding agent was given the following task:

<task>
%s
</task>

The agent produced the following git diff:

<git_diff>
%s
</git_diff>

Review the diff and decide whether it adequately addresses the task.
Respond with ONLY a JSON object in this exact format (no other text):

{
  "approved": true,
  "reason": "brief explanation of your decision"
}

Set "approved" to true if the changes reasonably address the task requirements.
Set "approved" to false if the changes are empty, clearly wrong, incomplete, or harmful.
Keep "reason" under 200 characters.`, originalPrompt, gitDiff)
}

type chatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// callChatCompletion sends a single chat completion request to the LiteLLM proxy.
func callChatCompletion(ctx context.Context, baseURL, apiKey, model, prompt string) (*llmJudgeVerdict, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("LiteLLM base URL not configured")
	}

	reqBody, err := json.Marshal(chatCompletionRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat completion returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no choices in completion response")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)

	// Extract JSON from the response (handle markdown code fences)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart : jsonEnd+1]
	}

	var verdict llmJudgeVerdict
	if err := json.Unmarshal([]byte(content), &verdict); err != nil {
		// If we can't parse, check if the content contains clear approval/rejection signals
		lower := strings.ToLower(content)
		if strings.Contains(lower, `"approved": true`) || strings.Contains(lower, `"approved":true`) {
			return &llmJudgeVerdict{Approved: true, Reason: "LLM approved (unparseable response)"}, nil
		}
		return &llmJudgeVerdict{Approved: false, Reason: fmt.Sprintf("LLM response not parseable: %s", truncate(content, 100))}, nil
	}

	return &verdict, nil
}
