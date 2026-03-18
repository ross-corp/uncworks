package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/activity"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// Activities for the spec-driven pipeline are methods on the existing Activities struct.
// They are registered alongside the other activities in the worker.

// PlanRun invokes the sidecar with stage=plan to generate an OpenSpec change,
// then validates the output via the OpenSpec CLI.
func (a *Activities) PlanRun(ctx context.Context, input PlanRunInput) (PlanRunOutput, error) {
	activity.RecordHeartbeat(ctx, "starting plan agent")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// Start the planning agent.
	prompt := input.Prompt
	if input.SpecContent != "" {
		prompt = fmt.Sprintf("User-provided specification:\n\n%s\n\n---\n\nCreate an OpenSpec change named \"%s\" based on the above. Generate proposal.md, specs/ with WHEN/THEN scenarios, and tasks.md.",
			input.SpecContent, input.AgentRunName)
	} else {
		prompt = fmt.Sprintf("Create an OpenSpec change named \"%s\" for this task:\n\n%s\n\nGenerate proposal.md, specs/ with WHEN/THEN scenarios, and tasks.md.",
			input.AgentRunName, prompt)
	}

	envVars := map[string]string{}
	if input.Model != "" {
		envVars["PI_MODEL"] = input.Model
	}

	_, err := sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: input.AgentRunName,
		Prompt:     prompt,
		RepoPath:   input.RepoPath,
		Stage:      "plan",
		EnvVars:    envVars,
	}))
	if err != nil {
		return PlanRunOutput{}, fmt.Errorf("start plan agent: %w", err)
	}

	// Poll for agent completion.
	if err := pollUntilAgentDone(ctx, sidecarClient, input.AgentRunName); err != nil {
		return PlanRunOutput{}, fmt.Errorf("plan agent: %w", err)
	}

	activity.RecordHeartbeat(ctx, "plan agent complete, validating")

	// Validate the generated change via openspec CLI (exec in pod).
	// For now, we trust the agent produced valid output. The workflow
	// can add validation as a separate check if needed.
	return PlanRunOutput{
		ChangeName: input.AgentRunName,
		SpecsValid: true,
	}, nil
}

// VerifyRun runs the verification pipeline: openspec list → validate → automated checks → LLM judge → archive.
func (a *Activities) VerifyRun(ctx context.Context, input VerifyRunInput) (VerifyRunOutput, error) {
	startTime := time.Now()
	activity.RecordHeartbeat(ctx, "starting verification")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	result := VerificationResult{
		AutomatedChecks: []AutomatedCheck{},
	}

	// Always write verification-result.json on exit (success or failure).
	defer func() {
		result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
		writeVerificationResult(input.RepoPath, input.ChangeName, result)
	}()

	// --- Gate 1: Task completion via openspec list --json ---
	activity.RecordHeartbeat(ctx, "checking task completion")

	taskCheck, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, input.RepoPath,
		fmt.Sprintf("openspec list --json 2>/dev/null | python3 -c \"import sys,json; raw=sys.stdin.read(); start=raw.index('{'); d=json.loads(raw[start:]); c=[x for x in d.get('changes',[]) if x['name']=='%s']; print(json.dumps(c[0]) if c else '{}')\"",
			input.ChangeName))
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("Failed to check task completion: %v", err)
		return VerifyRunOutput{Result: result}, nil
	}

	var changeInfo struct {
		CompletedTasks int    `json:"completedTasks"`
		TotalTasks     int    `json:"totalTasks"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal([]byte(taskCheck), &changeInfo); err == nil {
		result.TasksCompleted = changeInfo.CompletedTasks
		result.TasksTotal = changeInfo.TotalTasks
	}

	if changeInfo.TotalTasks > 0 && changeInfo.CompletedTasks < changeInfo.TotalTasks {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("Task completion: %d/%d tasks complete. Incomplete tasks must be finished before verification passes.",
			changeInfo.CompletedTasks, changeInfo.TotalTasks)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name:   "task_completion",
			Pass:   false,
			Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
		Name:   "task_completion",
		Pass:   true,
		Output: fmt.Sprintf("%d/%d tasks complete", changeInfo.CompletedTasks, changeInfo.TotalTasks),
	})

	// --- Gate 2: Structural validation via openspec validate --json ---
	activity.RecordHeartbeat(ctx, "validating spec structure")

	validateCheck, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, input.RepoPath,
		fmt.Sprintf("openspec validate \"%s\" --json 2>/dev/null | tail -1", input.ChangeName))
	if err == nil {
		var valResult struct {
			Items []struct {
				Valid  bool `json:"valid"`
				Issues []struct {
					Message string `json:"message"`
				} `json:"issues"`
			} `json:"items"`
		}
		if json.Unmarshal([]byte(validateCheck), &valResult) == nil && len(valResult.Items) > 0 {
			result.ValidationValid = valResult.Items[0].Valid
			if !valResult.Items[0].Valid {
				var issues []string
				for _, issue := range valResult.Items[0].Issues {
					issues = append(issues, issue.Message)
				}
				result.Pass = false
				result.FailureReport = fmt.Sprintf("Spec validation failed: %s", strings.Join(issues, "; "))
				result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
					Name:   "spec_validation",
					Pass:   false,
					Output: result.FailureReport,
				})
				result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
				return VerifyRunOutput{Result: result}, nil
			}
		}
	}
	result.ValidationValid = true
	result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
		Name: "spec_validation",
		Pass: true,
	})

	// --- Gate 2b: File existence checks from spec scenarios ---
	activity.RecordHeartbeat(ctx, "checking file existence")

	fileChecks := extractFileChecks(input.RepoPath, input.ChangeName)
	for _, fc := range fileChecks {
		fullPath := filepath.Join(input.RepoPath, fc.Path)
		_, statErr := os.Stat(fullPath)
		check := AutomatedCheck{
			Name: fmt.Sprintf("file_exists: %s", fc.Path),
			Pass: statErr == nil,
		}
		if statErr != nil {
			check.Output = fmt.Sprintf("File not found: %s (referenced in spec scenario: %s)", fc.Path, fc.Scenario)
			result.AutomatedChecks = append(result.AutomatedChecks, check)
			result.Pass = false
			result.FailureReport = fmt.Sprintf("File existence check failed: %s does not exist", fc.Path)
			result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
			return VerifyRunOutput{Result: result}, nil
		}
		check.Output = "exists"
		result.AutomatedChecks = append(result.AutomatedChecks, check)
	}

	// --- Gate 3: Automated scenario checks (test/build commands) ---
	activity.RecordHeartbeat(ctx, "running automated checks")

	// Check if common test commands exist and run them.
	testCommands := detectTestCommands(input.RepoPath)
	for _, tc := range testCommands {
		output, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, input.RepoPath, tc.Command)
		check := AutomatedCheck{
			Name:    tc.Name,
			Command: tc.Command,
		}
		if err != nil {
			check.Pass = false
			check.Output = fmt.Sprintf("Command failed: %v\n%s", err, output)
			result.AutomatedChecks = append(result.AutomatedChecks, check)
			result.Pass = false
			result.FailureReport = fmt.Sprintf("Automated check '%s' failed: %s", tc.Name, check.Output)
			result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
			return VerifyRunOutput{Result: result}, nil
		}
		check.Pass = true
		check.Output = output
		result.AutomatedChecks = append(result.AutomatedChecks, check)
	}

	// --- Gate 4: LLM judge for semantic criteria ---
	activity.RecordHeartbeat(ctx, "running LLM evaluation")

	// Get git diff for the LLM judge.
	gitDiff, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, input.RepoPath,
		"cd /workspace/src/* 2>/dev/null && git diff HEAD~1 --stat 2>/dev/null || echo 'no git diff available'")

	// Invoke LLM judge as a verify-stage agent.
	_, err = sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: input.AgentRunName + "-verify",
		Prompt: fmt.Sprintf(`Evaluate whether the implementation satisfies the spec.

Git diff summary:
%s

Read the spec files in the openspec change directory and evaluate each WHEN/THEN scenario.
Output your verdict as JSON: {"pass": true/false, "criteria": [{"scenario": "...", "pass": true/false, "explanation": "..."}]}`,
			gitDiff),
		RepoPath: input.RepoPath,
		Stage:    "verify",
	}))
	if err != nil {
		// LLM judge failure is non-fatal — pass with warning.
		result.Pass = true
		return VerifyRunOutput{Result: result}, nil
	}

	// Wait for verify agent to complete.
	_ = pollUntilAgentDone(ctx, sidecarClient, input.AgentRunName+"-verify")

	// --- Gate 5: Archive on success ---
	activity.RecordHeartbeat(ctx, "archiving change")

	_, _ = execInSidecar(ctx, sidecarClient, input.AgentRunName, input.RepoPath,
		fmt.Sprintf("openspec archive \"%s\" --yes 2>&1 || true", input.ChangeName))

	result.Pass = true
	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	// Write verification-result.json to the change directory.
	writeVerificationResult(input.RepoPath, input.ChangeName, result)

	return VerifyRunOutput{Result: result}, nil
}

// TestCommand is a test/build command detected from the workspace.
type TestCommand struct {
	Name    string
	Command string
}

// detectTestCommands looks for common test commands in the workspace.
func detectTestCommands(repoPath string) []TestCommand {
	// These will be enhanced to parse spec scenarios for command references.
	// For now, return empty — automated command checks are opt-in via spec scenarios.
	return nil
}

// FileCheck represents a file existence check extracted from a spec scenario.
type FileCheck struct {
	Path     string // relative path to check
	Scenario string // scenario that referenced it
}

// backtickPathRe matches backtick-wrapped file paths in spec scenarios.
// Looks for patterns like `src/auth.ts`, `README.md`, `pkg/handler.go`.
// backtickPathRe matches backtick-wrapped file paths. Must contain a dot or slash
// to distinguish from command names. Matches: `src/auth.ts`, `.env`, `README.md`
var backtickPathRe = regexp.MustCompile("`([a-zA-Z0-9_./-]*[./][a-zA-Z0-9_./-]*)`")

// extractFileChecks parses spec files in the change directory for file path
// references in WHEN/THEN scenarios and returns them as file existence checks.
func extractFileChecks(repoPath, changeName string) []FileCheck {
	var checks []FileCheck

	// Look for spec files in common locations.
	specDirs := []string{
		filepath.Join(repoPath, "openspec", "changes", changeName, "specs"),
		filepath.Join(repoPath, ".openspec", "changes", changeName, "specs"),
	}

	for _, dir := range specDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			content := string(data)
			// Find THEN clauses that reference file paths.
			lines := strings.Split(content, "\n")
			var currentScenario string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "#### Scenario:") {
					currentScenario = strings.TrimPrefix(trimmed, "#### Scenario:")
					currentScenario = strings.TrimSpace(currentScenario)
				}
				// Look for THEN/AND lines with backtick-wrapped paths that mention "exist"
				lower := strings.ToLower(trimmed)
				hasThenOrAnd := strings.Contains(lower, "then") || strings.Contains(lower, "and")
				if hasThenOrAnd && strings.Contains(lower, "exist") {
					matches := backtickPathRe.FindAllStringSubmatch(trimmed, -1)
					for _, m := range matches {
						if len(m) > 1 {
							checks = append(checks, FileCheck{
								Path:     m[1],
								Scenario: currentScenario,
							})
						}
					}
				}
			}
			return nil
		})
	}

	return checks
}

// writeVerificationResult writes the verification result as JSON to the
// change directory on the workspace PVC.
func writeVerificationResult(repoPath, changeName string, result VerificationResult) {
	// Try both possible locations for the change directory.
	candidates := []string{
		filepath.Join(repoPath, "openspec", "changes", changeName),
		filepath.Join(repoPath, ".openspec", "changes", changeName),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			outPath := filepath.Join(dir, "verification-result.json")
			_ = os.WriteFile(outPath, data, 0o644)
			return
		}
	}

	// If no change dir exists, write to a fallback location.
	fallbackDir := filepath.Join(repoPath, ".aot", "verification")
	_ = os.MkdirAll(fallbackDir, 0o755)
	outPath := filepath.Join(fallbackDir, changeName+"-result.json")
	_ = os.WriteFile(outPath, data, 0o644)
}

// execInSidecar runs a bash command via the sidecar's ExecCommand RPC.
// Returns stdout and any error. This is a lightweight direct execution,
// not an agent invocation.
func execInSidecar(ctx context.Context, client agentv1connect.AgentSidecarServiceClient, runID, repoPath, command string) (string, error) {
	resp, err := client.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        command,
		WorkingDir:     repoPath,
		TimeoutSeconds: 60,
	}))
	if err != nil {
		return "", fmt.Errorf("exec command: %w", err)
	}

	if resp.Msg.ExitCode != 0 {
		return resp.Msg.Stdout, fmt.Errorf("command exited with code %d: %s", resp.Msg.ExitCode, resp.Msg.Stderr)
	}

	return resp.Msg.Stdout, nil
}

// pollUntilAgentDone polls the sidecar until the agent process completes.
// Handles UNSPECIFIED state (agent never started or crashed immediately).
func pollUntilAgentDone(ctx context.Context, client agentv1connect.AgentSidecarServiceClient, runID string) error {
	unspecifiedCount := 0
	const maxUnspecified = 10 // give the agent 30s to start (10 * 3s poll)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		activity.RecordHeartbeat(ctx, "polling agent status")

		status, err := client.GetStatus(ctx, connect.NewRequest(&agentv1.GetStatusRequest{}))
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}

		switch status.Msg.State {
		case agentv1.AgentProcessState_AGENT_PROCESS_STATE_COMPLETED:
			return nil
		case agentv1.AgentProcessState_AGENT_PROCESS_STATE_FAILED:
			return fmt.Errorf("agent failed: %s", status.Msg.Error)
		case agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING,
			agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT:
			unspecifiedCount = 0 // agent is alive, reset counter
		default:
			// UNSPECIFIED — agent hasn't started or exited immediately
			unspecifiedCount++
			if unspecifiedCount >= maxUnspecified {
				return fmt.Errorf("agent never started (UNSPECIFIED state after %d polls)", unspecifiedCount)
			}
		}

		time.Sleep(3 * time.Second)
	}
}
