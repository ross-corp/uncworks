package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// PlanRun runs the planning stage of a spec-driven pipeline: initializes OpenSpec,
// scaffolds the change, and runs the planning agent against the spec content.
func (a *Activities) PlanRun(ctx context.Context, input PlanRunInput) (PlanRunOutput, error) {
	activity.RecordHeartbeat(ctx, "starting plan stage")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// Step 1: Determine workspaces
	// repoDir = where the code lives (resolved by sidecar to /workspace/<repo>)
	// specDir = where OpenSpec artifacts live (/workspace — NOT inside the repo)
	workDir := input.RepoPath
	specDir := "/workspace"
	slog.Info("[PlanRun] workspace dirs", "run", input.AgentRunName, "repoDir", workDir, "specDir", specDir)

	// Step 2: OpenSpec init in workspace root (not inside repo)
	activity.RecordHeartbeat(ctx, "initializing openspec in workspace")
	initOut, initErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		"test -f openspec/config.yaml || openspec init --tools pi --force")
	if initErr != nil {
		slog.Warn("[PlanRun] openspec init warning (non-fatal)", "run", input.AgentRunName, "err", initErr, "stdout", initOut)
	} else {
		slog.Info("[PlanRun] openspec init OK", "run", input.AgentRunName, "stdout", truncate(initOut, 200))
	}

	// Step 3: Scaffold the change BEFORE starting the agent (idempotent — skip if already exists)
	activity.RecordHeartbeat(ctx, "scaffolding openspec change")
	checkCmd := fmt.Sprintf("test -d openspec/changes/%q && echo exists || echo missing", input.AgentRunName)
	checkOut, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, checkCmd)
	if strings.TrimSpace(checkOut) == "exists" {
		slog.Info("[PlanRun] change directory already exists, skipping scaffold", "run", input.AgentRunName)
	} else {
		newChangeCmd := fmt.Sprintf("openspec new change %q", input.AgentRunName)
		newOut, newErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, newChangeCmd)
		if newErr != nil {
			return PlanRunOutput{}, fmt.Errorf("scaffold openspec change: %w (output: %s)", newErr, newOut)
		}
		slog.Info("[PlanRun] scaffolded change", "run", input.AgentRunName, "output", truncate(newOut, 200))
	}

	// Step 4: Verify the change was created via status
	activity.RecordHeartbeat(ctx, "verifying scaffolded change")
	statusCmd := fmt.Sprintf("openspec status --change %q --json", input.AgentRunName)
	scaffoldStatusOut, scaffoldStatusErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, statusCmd)
	if scaffoldStatusErr != nil {
		return PlanRunOutput{}, fmt.Errorf("verify scaffolded change: %w (output: %s)", scaffoldStatusErr, scaffoldStatusOut)
	}
	slog.Info("[PlanRun] scaffolded change status", "run", input.AgentRunName, "status", truncate(scaffoldStatusOut, 300))

	scaffoldStatus, scaffoldParseErr := parseOpenSpecStatusResponse(scaffoldStatusOut)
	if scaffoldParseErr != nil {
		return PlanRunOutput{}, fmt.Errorf("parse scaffolded change status: %w", scaffoldParseErr)
	}

	// Step 4: Get templates from openspec instructions for each artifact
	activity.RecordHeartbeat(ctx, "fetching artifact templates")
	proposalTemplate := ""
	specsTemplate := ""
	tasksTemplate := ""

	proposalInstrOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec instructions proposal --change %q --json", input.AgentRunName))
	if err == nil {
		if t, parseErr := parseOpenSpecInstructionsResponse(proposalInstrOut); parseErr == nil {
			proposalTemplate = t
		} else {
			slog.Warn("[PlanRun] parse proposal instructions warning", "run", input.AgentRunName, "err", parseErr)
		}
	} else {
		slog.Warn("[PlanRun] openspec instructions proposal warning", "run", input.AgentRunName, "err", err)
	}

	specsInstrOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec instructions specs --change %q --json", input.AgentRunName))
	if err == nil {
		if t, parseErr := parseOpenSpecInstructionsResponse(specsInstrOut); parseErr == nil {
			specsTemplate = t
		} else {
			slog.Warn("[PlanRun] parse specs instructions warning", "run", input.AgentRunName, "err", parseErr)
		}
	} else {
		slog.Warn("[PlanRun] openspec instructions specs warning", "run", input.AgentRunName, "err", err)
	}

	tasksInstrOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec instructions tasks --change %q --json", input.AgentRunName))
	if err == nil {
		if t, parseErr := parseOpenSpecInstructionsResponse(tasksInstrOut); parseErr == nil {
			tasksTemplate = t
		} else {
			slog.Warn("[PlanRun] parse tasks instructions warning", "run", input.AgentRunName, "err", parseErr)
		}
	} else {
		slog.Warn("[PlanRun] openspec instructions tasks warning", "run", input.AgentRunName, "err", err)
	}

	// Step 5: Build structured agent prompt with exact paths and templates
	prompt := buildPlanAgentPrompt(input.Prompt, input.SpecContent, input.AgentRunName,
		scaffoldStatus, proposalTemplate, specsTemplate, tasksTemplate)

	envVars := map[string]string{}
	if input.Model != "" {
		envVars["PI_MODEL"] = input.Model
	}

	slog.Info("[PlanRun] starting plan agent", "run", input.AgentRunName, "workDir", workDir)
	_, err = sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId:   input.AgentRunName,
		Prompt:       prompt,
		RepoPath:     workDir,
		Stage:        "plan",
		EnvVars:      envVars,
		ParentSpanId: input.ParentSpanID,
		TraceId:      input.TraceID,
	}))
	if err != nil {
		return PlanRunOutput{}, fmt.Errorf("start plan agent: %w", err)
	}

	if err = pollUntilAgentDone(ctx, sidecarClient, input.AgentRunName); err != nil {
		return PlanRunOutput{}, fmt.Errorf("plan agent: %w", err)
	}

	// Fix 1: REAL validation via openspec CLI (was hardcoded SpecsValid: true)
	activity.RecordHeartbeat(ctx, "validating openspec change")

	output := PlanRunOutput{ChangeName: input.AgentRunName}

	// Validate the change structure (in specDir, not repoDir)
	validateOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec validate %q --json", input.AgentRunName))
	if err != nil {
		output.ValidationErrors = append(output.ValidationErrors, fmt.Sprintf("openspec validate failed: %v (stderr: %s)", err, validateOut))
		return output, nil
	}

	valResp, err := parseOpenSpecValidateResponse(validateOut)
	if err != nil {
		output.ValidationErrors = append(output.ValidationErrors, fmt.Sprintf("failed to parse validate response: %v", err))
		return output, nil
	}
	if len(valResp.Items) > 0 && !valResp.Items[0].Valid {
		for _, issue := range valResp.Items[0].Issues {
			output.ValidationErrors = append(output.ValidationErrors, issue.Message)
		}
		return output, nil
	}

	// Check artifact completion status (in specDir)
	statusOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec status --change %q --json", input.AgentRunName))
	if err != nil {
		output.ValidationErrors = append(output.ValidationErrors, fmt.Sprintf("openspec status failed: %v", err))
		return output, nil
	}

	statusResp, err := parseOpenSpecStatusResponse(statusOut)
	if err != nil {
		output.ValidationErrors = append(output.ValidationErrors, fmt.Sprintf("failed to parse status response: %v", err))
		return output, nil
	}
	if !statusResp.AllArtifactsDone() {
		missing := statusResp.MissingArtifacts()
		output.ValidationErrors = append(output.ValidationErrors, fmt.Sprintf("incomplete artifacts: %v", missing))
		return output, nil
	}

	output.SpecsValid = true
	output.TaskCount = len(statusResp.Artifacts)
	return output, nil
}

// VerifyRun runs the verification stage of a spec-driven pipeline: executes automated
// checks and optionally invokes an LLM judge to evaluate semantic acceptance criteria.
func (a *Activities) VerifyRun(ctx context.Context, input VerifyRunInput) (VerifyRunOutput, error) {
	startTime := time.Now()
	activity.RecordHeartbeat(ctx, "starting verification")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// workDir = repo dir (resolved by sidecar), specDir = /workspace for openspec
	workDir := input.RepoPath
	specDir := "/workspace"
	slog.Info("[VerifyRun] workspace dirs", "run", input.AgentRunName, "repoDir", workDir, "specDir", specDir)

	result := VerificationResult{
		AutomatedChecks: []AutomatedCheck{},
	}

	defer func() {
		result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
		writeVerificationResult(workDir, input.ChangeName, result)
	}()

	// ── Gate 1: Task completion (Fix 4: fail on errors, not pass) ──
	activity.RecordHeartbeat(ctx, "checking task completion")

	listOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		"openspec list --json")
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("openspec list failed: %v (output: %s)", err, listOut)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "task_completion", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	listResp, err := parseOpenSpecListResponse(listOut)
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("failed to parse openspec list: %v", err)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "task_completion", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	changeInfo := listResp.FindChange(input.ChangeName)
	if changeInfo == nil {
		// Fix 4: no change found = FAIL (was silently passing with TotalTasks=0)
		result.Pass = false
		result.FailureReport = fmt.Sprintf("change %q not found in openspec list output", input.ChangeName)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "task_completion", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	result.TasksCompleted = changeInfo.CompletedTasks
	result.TasksTotal = changeInfo.TotalTasks

	if changeInfo.TotalTasks == 0 {
		// Fix 4: TotalTasks=0 = FAIL (means no tasks were created)
		result.Pass = false
		result.FailureReport = "no tasks found in change — planning agent may not have created tasks.md"
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "task_completion", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	if changeInfo.CompletedTasks < changeInfo.TotalTasks {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("task completion: %d/%d tasks complete",
			changeInfo.CompletedTasks, changeInfo.TotalTasks)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "task_completion", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
		Name: "task_completion", Pass: true,
		Output: fmt.Sprintf("%d/%d tasks complete", changeInfo.CompletedTasks, changeInfo.TotalTasks),
	})

	// ── Gate 2: Structural validation (Fix 5: remove hardcoded true, Fix 6.2-6.5) ──
	activity.RecordHeartbeat(ctx, "validating spec structure")

	valOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec validate %q --json", input.ChangeName))
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("openspec validate failed: %v (output: %s)", err, valOut)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "spec_validation", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	valResp, err := parseOpenSpecValidateResponse(valOut)
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("failed to parse validate response: %v", err)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "spec_validation", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	if len(valResp.Items) > 0 && !valResp.Items[0].Valid {
		var issues []string
		for _, issue := range valResp.Items[0].Issues {
			issues = append(issues, issue.Message)
		}
		result.ValidationValid = false
		result.Pass = false
		result.FailureReport = fmt.Sprintf("spec validation failed: %s", strings.Join(issues, "; "))
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "spec_validation", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	result.ValidationValid = true
	result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
		Name: "spec_validation", Pass: true,
	})

	// ── Gate 2b: File existence checks ──
	activity.RecordHeartbeat(ctx, "checking file existence")

	fileChecks := extractFileChecks(workDir, input.ChangeName)
	for _, fc := range fileChecks {
		checkOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir,
			fmt.Sprintf("test -f %q && echo exists || echo missing", fc.Path))
		check := AutomatedCheck{
			Name: fmt.Sprintf("file_exists: %s", fc.Path),
		}
		if err != nil || strings.TrimSpace(checkOut) != "exists" {
			check.Pass = false
			check.Output = fmt.Sprintf("file not found: %s (spec scenario: %s)", fc.Path, fc.Scenario)
			result.AutomatedChecks = append(result.AutomatedChecks, check)
			result.Pass = false
			result.FailureReport = check.Output
			return VerifyRunOutput{Result: result}, nil
		}
		check.Pass = true
		check.Output = "exists"
		result.AutomatedChecks = append(result.AutomatedChecks, check)
	}

	// ── Gate 3: Test command extraction and execution ──
	activity.RecordHeartbeat(ctx, "running automated checks")

	testCommands := detectTestCommands(workDir, input.ChangeName)
	for _, tc := range testCommands {
		output, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir, tc.Command)
		check := AutomatedCheck{
			Name:    tc.Name,
			Command: tc.Command,
		}
		if err != nil {
			check.Pass = false
			check.Output = fmt.Sprintf("command failed: %v\n%s", err, output)
			result.AutomatedChecks = append(result.AutomatedChecks, check)
			result.Pass = false
			result.FailureReport = fmt.Sprintf("automated check '%s' failed: %s", tc.Name, check.Output)
			return VerifyRunOutput{Result: result}, nil
		}
		check.Pass = true
		check.Output = truncate(output, 500)
		result.AutomatedChecks = append(result.AutomatedChecks, check)
	}

	// ── Tier 2: Manage agent review (replaces LLM judge) ──
	activity.RecordHeartbeat(ctx, "structural checks passed, starting manage agent review")

	// Read the full git diff (not just --stat)
	gitDiff, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir,
		"git diff HEAD~1 2>/dev/null || echo 'no git diff available'")

	// Read spec content for the review prompt
	specContent, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("find openspec/changes/%s/specs -name 'spec.md' -exec cat {} + 2>/dev/null || echo 'no specs'", input.ChangeName))

	// Read implement agent's output log for question routing
	implementLog := ReadImplementAgentLog(ctx, sidecarClient, input.AgentRunName, workDir)

	// Build the manage agent review prompt
	reviewPrompt := buildManageReviewPrompt(
		input.ChangeName, gitDiff, specContent, implementLog, input.PreviousReviewFeedback)

	// Start the manage agent review session
	manageModel := input.ManageModel
	_, err = sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId:   input.AgentRunName + "-verify",
		Prompt:       reviewPrompt,
		RepoPath:     workDir,
		Stage:        "verify",
		ParentSpanId: input.ParentSpanID,
		TraceId:      input.TraceID,
		EnvVars:      map[string]string{"PI_MODEL": manageModel},
	}))
	if err != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("manage agent review failed to start: %v", err)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "llm_judge", Pass: false, Output: result.FailureReport,
		})
		slog.Warn("manage agent review failed to start", "err", err)
		return VerifyRunOutput{Result: result}, nil
	}

	if pollErr := pollUntilAgentDone(ctx, sidecarClient, input.AgentRunName+"-verify"); pollErr != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("manage agent review failed: %v", pollErr)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "llm_judge", Pass: false, Output: result.FailureReport,
		})
		slog.Warn("manage agent review failed", "err", pollErr)
		return VerifyRunOutput{Result: result}, nil
	}

	activity.RecordHeartbeat(ctx, "parsing manage agent verdict")
	verdictJSON, readErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir,
		"cat .aot/logs/agent.jsonl 2>/dev/null | tail -50")
	if readErr != nil {
		result.Pass = false
		result.FailureReport = fmt.Sprintf("failed to read manage agent log: %v", readErr)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "llm_judge", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	verdict := parseLLMVerdict(verdictJSON)
	if verdict == nil {
		result.Pass = false
		result.FailureReport = "manage agent produced no parseable verdict"
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "llm_judge", Pass: false, Output: result.FailureReport,
		})
		slog.Warn("manage agent produced no parseable verdict", "run", input.AgentRunName)
		return VerifyRunOutput{Result: result}, nil
	}

	result.LLMVerdict = verdict
	if !verdict.Pass {
		var failedCriteria []string
		for _, c := range verdict.Criteria {
			if !c.Pass {
				failedCriteria = append(failedCriteria, fmt.Sprintf("%s: %s", c.Scenario, c.Explanation))
			}
		}
		result.ReviewFeedback = strings.Join(failedCriteria, "\n")
		result.Pass = false
		result.FailureReport = fmt.Sprintf("Manage agent review failed: %s", strings.Join(failedCriteria, "; "))
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "llm_judge", Pass: false, Output: result.FailureReport,
		})
		return VerifyRunOutput{Result: result}, nil
	}

	result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
		Name: "llm_judge", Pass: true, Output: "manage agent review passed",
	})

	// ── Gate 5: Archive ──
	activity.RecordHeartbeat(ctx, "archiving change")

	archiveOut, archiveErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec archive %q --yes", input.ChangeName))
	if archiveErr != nil {
		// Fix 8: report archive errors (was swallowed with || true)
		// Archive failure is informational, not a gate blocker
		slog.Warn("openspec archive warning", "err", archiveErr, "output", archiveOut)
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "archive", Pass: false, Output: fmt.Sprintf("archive failed: %v", archiveErr),
		})
	} else {
		result.AutomatedChecks = append(result.AutomatedChecks, AutomatedCheck{
			Name: "archive", Pass: true, Output: "change archived successfully",
		})
	}

	result.Pass = true
	return VerifyRunOutput{Result: result}, nil
}

// ======================================================================
// parseLLMVerdict extracts a JSON verdict from JSONL agent log content.
// ======================================================================
func parseLLMVerdict(jsonlContent string) *LLMVerdict {
	// Look for the last assistant message containing a JSON verdict
	lines := strings.Split(jsonlContent, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var event struct {
			Type    string `json:"type"`
			Message *struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if event.Type != "message_end" || event.Message == nil || event.Message.Role != "assistant" {
			continue
		}

		// Try to find JSON verdict in content blocks
		var contentBlocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(event.Message.Content, &contentBlocks) != nil {
			continue
		}
		for _, block := range contentBlocks {
			if block.Type != "text" {
				continue
			}
			// Extract JSON from the text
			jsonBytes, err := parseOpenSpecJSON(block.Text)
			if err != nil {
				continue
			}
			var verdict LLMVerdict
			if json.Unmarshal(jsonBytes, &verdict) == nil && len(verdict.Criteria) > 0 {
				return &verdict
			}
		}
	}
	return nil
}

// TestCommand represents a shell command extracted from an OpenSpec verification criterion.
type TestCommand struct {
	Name    string
	Command string
}

// commandKeywords are words that indicate a WHEN/THEN line references a command.
var commandKeywords = []string{"run", "execute", "pass", "exit", "build", "test", "compile", "lint"}

// backtickCommandRe matches backtick-wrapped commands (contain spaces, indicating a command not a path).
var backtickCommandRe = regexp.MustCompile("`([^`]+\\s[^`]+)`")

func detectTestCommands(repoPath, changeName string) []TestCommand {
	var commands []TestCommand
	seen := make(map[string]bool)

	specDirs := []string{
		filepath.Join("/workspace", "openspec", "changes", changeName, "specs"),
		filepath.Join(repoPath, "openspec", "changes", changeName, "specs"),
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

			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				lower := strings.ToLower(trimmed)

				// Only look at WHEN/THEN/AND lines
				if !strings.Contains(lower, "when") && !strings.Contains(lower, "then") && !strings.Contains(lower, "and") {
					continue
				}

				// Check for command keywords
				hasKeyword := false
				for _, kw := range commandKeywords {
					if strings.Contains(lower, kw) {
						hasKeyword = true
						break
					}
				}
				if !hasKeyword {
					continue
				}

				// Extract backtick-wrapped commands (must contain spaces)
				matches := backtickCommandRe.FindAllStringSubmatch(trimmed, -1)
				for _, m := range matches {
					if len(m) < 2 {
						continue
					}
					cmd := strings.TrimSpace(m[1])
					if seen[cmd] {
						continue
					}
					seen[cmd] = true
					commands = append(commands, TestCommand{
						Name:    fmt.Sprintf("spec_command: %s", truncate(cmd, 40)),
						Command: cmd,
					})
				}
			}
			return nil
		})
	}

	return commands
}

// FileCheck represents a file existence check extracted from an OpenSpec verification criterion.
type FileCheck struct {
	Path     string
	Scenario string
}

var backtickPathRe = regexp.MustCompile("`([a-zA-Z0-9_./-]*[./][a-zA-Z0-9_./-]*)`")

func extractFileChecks(repoPath, changeName string) []FileCheck {
	var checks []FileCheck

	specDirs := []string{
		filepath.Join("/workspace", "openspec", "changes", changeName, "specs"),
		filepath.Join(repoPath, "openspec", "changes", changeName, "specs"),
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
			lines := strings.Split(content, "\n")
			var currentScenario string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "#### Scenario:") {
					currentScenario = strings.TrimSpace(strings.TrimPrefix(trimmed, "#### Scenario:"))
				}
				lower := strings.ToLower(trimmed)
				hasThenOrAnd := strings.Contains(lower, "then") || strings.Contains(lower, "and")
				if hasThenOrAnd && strings.Contains(lower, "exist") {
					matches := backtickPathRe.FindAllStringSubmatch(trimmed, -1)
					for _, m := range matches {
						if len(m) > 1 {
							checks = append(checks, FileCheck{Path: m[1], Scenario: currentScenario})
						}
					}
				}
			}
			return nil
		})
	}

	return checks
}

// ======================================================================
// Helpers
// ======================================================================

// buildPlanAgentPrompt constructs the planning agent prompt.
// Directs the agent to use openspec CLI commands for deterministic spec generation.
func buildPlanAgentPrompt(userPrompt, specContent, changeName string, status *OpenSpecStatusResponse, proposalTpl, specsTpl, tasksTpl string) string {
	var sb strings.Builder

	// User task first
	if specContent != "" {
		sb.WriteString(specContent)
	} else {
		sb.WriteString(userPrompt)
	}

	sb.WriteString("\n\n---\n\n")
	fmt.Fprintf(&sb, "## OpenSpec Change: %s\n\n", changeName)
	sb.WriteString("The change directory has been scaffolded at `/workspace/openspec/changes/")
	sb.WriteString(changeName)
	sb.WriteString("/`.\n")
	sb.WriteString("All openspec commands MUST run from `/workspace` (NOT from the repo dir).\n\n")

	sb.WriteString("### Steps\n\n")
	sb.WriteString("1. Read the codebase to understand what needs to change\n")
	fmt.Fprintf(&sb, "2. Run `cd /workspace && openspec instructions proposal --change %q` to see the proposal template\n", changeName)
	fmt.Fprintf(&sb, "3. Write `/workspace/openspec/changes/%s/proposal.md` following the template\n", changeName)
	fmt.Fprintf(&sb, "4. Run `cd /workspace && openspec instructions specs --change %q` to see the spec template\n", changeName)
	fmt.Fprintf(&sb, "5. Write specs to `/workspace/openspec/changes/%s/specs/<capability>/spec.md`\n", changeName)
	sb.WriteString("   - Each requirement MUST use SHALL or MUST (e.g., \"The system SHALL...\")\n")
	sb.WriteString("   - Each requirement MUST have at least one WHEN/THEN scenario\n")
	fmt.Fprintf(&sb, "6. Run `cd /workspace && openspec instructions tasks --change %q` to see the tasks template\n", changeName)
	fmt.Fprintf(&sb, "7. Write `/workspace/openspec/changes/%s/tasks.md` with numbered checkbox groups\n", changeName)
	fmt.Fprintf(&sb, "8. Write `/workspace/openspec/changes/%s/design.md` with architecture notes\n", changeName)
	fmt.Fprintf(&sb, "9. Run `cd /workspace && openspec validate %q --json` to verify your spec is valid\n", changeName)
	sb.WriteString("   - Fix any validation errors and re-validate\n")
	fmt.Fprintf(&sb, "10. Run `cd /workspace && openspec status --change %q --json` to confirm all artifacts are complete\n\n", changeName)

	sb.WriteString("### Rules\n")
	sb.WriteString("- Only create spec artifacts, do NOT implement code\n")
	sb.WriteString("- Keep tasks proportional — a simple task should have 3-8 tasks\n")
	sb.WriteString("- Stop after all artifacts pass validation\n")

	return sb.String()
}

func writeVerificationResult(repoPath, changeName string, result VerificationResult) {
	candidates := []string{
		filepath.Join("/workspace", "openspec", "changes", changeName),
		filepath.Join(repoPath, "openspec", "changes", changeName),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			if writeErr := os.WriteFile(filepath.Join(dir, "verification-result.json"), data, 0o644); writeErr != nil {
				slog.Warn("failed to write verification result", "dir", dir, "err", writeErr)
			}
			return
		}
	}

	fallbackDir := filepath.Join(repoPath, ".aot", "verification")
	if mkdirErr := os.MkdirAll(fallbackDir, 0o755); mkdirErr != nil {
		slog.Warn("failed to create fallback verification dir", "dir", fallbackDir, "err", mkdirErr)
		return
	}
	if writeErr := os.WriteFile(filepath.Join(fallbackDir, changeName+"-result.json"), data, 0o644); writeErr != nil {
		slog.Warn("failed to write fallback verification result", "dir", fallbackDir, "err", writeErr)
	}
}

// execInSidecar runs a bash command via the sidecar's ExecCommand RPC.
// Fix 9: returns stdout AND captures stderr in errors (no more 2>/dev/null).
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
// The inter-poll sleep uses a select on ctx.Done() so that cancellation is
// noticed within the sleep window rather than waiting the full 3 seconds.
func pollUntilAgentDone(ctx context.Context, client agentv1connect.AgentSidecarServiceClient, runID string) error {
	unspecifiedCount := 0
	const maxUnspecified = 10
	const pollInterval = 3 * time.Second

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
			unspecifiedCount = 0
		default:
			unspecifiedCount++
			if unspecifiedCount >= maxUnspecified {
				return fmt.Errorf("agent never started (UNSPECIFIED state after %d polls)", unspecifiedCount)
			}
		}

		// Context-aware sleep: returns immediately on cancellation rather than
		// blocking for the full poll interval after the activity context is done.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// ======================================================================
// Tiered Verification — Manage Agent Review Helpers
// ======================================================================

// ReadImplementAgentLog reads the last N lines of the implement agent's
// JSONL log and extracts a summary of assistant messages and tool calls.
// Used by the manage agent review in VerifyRun (Tier 2).
func ReadImplementAgentLog(ctx context.Context, sc agentv1connect.AgentSidecarServiceClient, agentRunName, workDir string) string {
	out, err := execInSidecar(ctx, sc, agentRunName, workDir,
		"tail -200 /workspace/.aot/logs/agent.jsonl 2>/dev/null || echo ''")
	if err != nil || strings.TrimSpace(out) == "" {
		return "(implement agent log not available)"
	}

	var summary strings.Builder
	lineCount := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if json.Unmarshal([]byte(line), &evt) != nil {
			continue
		}
		evtType, _ := evt["type"].(string)
		switch evtType {
		case "message_end":
			if msg, ok := evt["message"].(map[string]interface{}); ok {
				if role, _ := msg["role"].(string); role == "assistant" {
					if contents, ok := msg["content"].([]interface{}); ok {
						for _, c := range contents {
							if cm, ok := c.(map[string]interface{}); ok {
								if text, ok := cm["text"].(string); ok && text != "" {
									if len(text) > 500 {
										text = text[:500] + "..."
									}
									summary.WriteString("[assistant] " + text + "\n")
									lineCount++
								}
							}
						}
					}
				}
			}
		case "tool_execution_end":
			toolName, _ := evt["toolName"].(string)
			if toolName != "" {
				summary.WriteString("[tool:" + toolName + "] completed\n")
				lineCount++
			}
		}
		if lineCount >= 50 {
			summary.WriteString("... (truncated)\n")
			break
		}
	}
	if summary.Len() == 0 {
		return "(no implement agent activity found)"
	}
	return summary.String()
}

// buildManageReviewPrompt assembles the manage agent's review prompt from
// the git diff, spec scenarios, implement agent output, and prior feedback.
func buildManageReviewPrompt(changeName, gitDiff, specContent, implementLog, previousFeedback string) string {
	var prompt strings.Builder

	_, _ = fmt.Fprintf(&prompt, `You are the manage agent reviewing the implementation of OpenSpec change '%s'.

Your job is to evaluate whether the implement agent correctly completed the work. You are a senior engineer reviewing a junior engineer's PR. Be thorough but fair.

`, changeName)

	prompt.WriteString("## Git Diff\n\n```\n")
	if len(gitDiff) > 8000 {
		half := 4000
		prompt.WriteString(gitDiff[:half])
		prompt.WriteString("\n\n... (diff truncated) ...\n\n")
		prompt.WriteString(gitDiff[len(gitDiff)-half:])
	} else {
		prompt.WriteString(gitDiff)
	}
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("## Spec Requirements\n\n")
	prompt.WriteString(specContent)
	prompt.WriteString("\n\n")

	prompt.WriteString("## Implement Agent Activity\n\n")
	prompt.WriteString(implementLog)
	prompt.WriteString("\n\n")

	if previousFeedback != "" {
		prompt.WriteString("## Previous Review Feedback (from last attempt)\n\n")
		prompt.WriteString(previousFeedback)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString(`## Instructions

1. Read each spec requirement and its WHEN/THEN scenarios.
2. Check the git diff to verify each scenario is satisfied.
3. If unclear, use the read tool to examine the modified files.
4. If the implement agent left questions in its output, either answer them (if you can) or escalate to the human via ask_user.
5. Output your verdict as JSON:

{"pass": true/false, "feedback": "detailed review comments", "criteria": [{"scenario": "...", "pass": true/false, "explanation": "..."}]}
`)

	return prompt.String()
}
