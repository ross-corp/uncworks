package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// ======================================================================
// PlanRun — Fix 1 (real validation), Fix 2 (openspec init), Fix 3 (partial)
// ======================================================================

func (a *Activities) PlanRun(ctx context.Context, input PlanRunInput) (PlanRunOutput, error) {
	activity.RecordHeartbeat(ctx, "starting plan stage")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// Step 1: Determine workspaces
	// repoDir = where the code lives (resolved by sidecar to /workspace/<repo>)
	// specDir = where OpenSpec artifacts live (/workspace — NOT inside the repo)
	workDir := input.RepoPath
	specDir := "/workspace"
	log.Printf("[PlanRun %s] repoDir=%s specDir=%s", input.AgentRunName, workDir, specDir)

	// Step 2: OpenSpec init in workspace root (not inside repo)
	activity.RecordHeartbeat(ctx, "initializing openspec in workspace")
	initOut, initErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		"test -f openspec/config.yaml || openspec init --tools pi --force")
	if initErr != nil {
		log.Printf("[PlanRun %s] openspec init warning (non-fatal): %v stdout=%q", input.AgentRunName, initErr, initOut)
	} else {
		log.Printf("[PlanRun %s] openspec init OK: %s", input.AgentRunName, truncate(initOut, 200))
	}

	// Step 3: Scaffold the change BEFORE starting the agent (idempotent — skip if already exists)
	activity.RecordHeartbeat(ctx, "scaffolding openspec change")
	checkCmd := fmt.Sprintf("test -d openspec/changes/%s && echo exists || echo missing", input.AgentRunName)
	checkOut, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, checkCmd)
	if strings.TrimSpace(checkOut) == "exists" {
		log.Printf("[PlanRun %s] change directory already exists, skipping scaffold", input.AgentRunName)
	} else {
		newChangeCmd := fmt.Sprintf("openspec new change %q", input.AgentRunName)
		newOut, newErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, newChangeCmd)
		if newErr != nil {
			return PlanRunOutput{}, fmt.Errorf("scaffold openspec change: %w (output: %s)", newErr, newOut)
		}
		log.Printf("[PlanRun %s] scaffolded change: %s", input.AgentRunName, truncate(newOut, 200))
	}

	// Step 4: Verify the change was created via status
	activity.RecordHeartbeat(ctx, "verifying scaffolded change")
	statusCmd := fmt.Sprintf("openspec status --change %q --json", input.AgentRunName)
	scaffoldStatusOut, scaffoldStatusErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir, statusCmd)
	if scaffoldStatusErr != nil {
		return PlanRunOutput{}, fmt.Errorf("verify scaffolded change: %w (output: %s)", scaffoldStatusErr, scaffoldStatusOut)
	}
	log.Printf("[PlanRun %s] status response: %s", input.AgentRunName, truncate(scaffoldStatusOut, 300))

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
			log.Printf("[PlanRun %s] parse proposal instructions warning: %v", input.AgentRunName, parseErr)
		}
	} else {
		log.Printf("[PlanRun %s] openspec instructions proposal warning: %v", input.AgentRunName, err)
	}

	specsInstrOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec instructions specs --change %q --json", input.AgentRunName))
	if err == nil {
		if t, parseErr := parseOpenSpecInstructionsResponse(specsInstrOut); parseErr == nil {
			specsTemplate = t
		} else {
			log.Printf("[PlanRun %s] parse specs instructions warning: %v", input.AgentRunName, parseErr)
		}
	} else {
		log.Printf("[PlanRun %s] openspec instructions specs warning: %v", input.AgentRunName, err)
	}

	tasksInstrOut, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec instructions tasks --change %q --json", input.AgentRunName))
	if err == nil {
		if t, parseErr := parseOpenSpecInstructionsResponse(tasksInstrOut); parseErr == nil {
			tasksTemplate = t
		} else {
			log.Printf("[PlanRun %s] parse tasks instructions warning: %v", input.AgentRunName, parseErr)
		}
	} else {
		log.Printf("[PlanRun %s] openspec instructions tasks warning: %v", input.AgentRunName, err)
	}

	// Step 5: Build structured agent prompt with exact paths and templates
	prompt := buildPlanAgentPrompt(input.Prompt, input.SpecContent, input.AgentRunName,
		scaffoldStatus, proposalTemplate, specsTemplate, tasksTemplate)

	envVars := map[string]string{}
	if input.Model != "" {
		envVars["PI_MODEL"] = input.Model
	}

	log.Printf("[PlanRun %s] starting plan agent in workDir=%s", input.AgentRunName, workDir)
	_, err = sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: input.AgentRunName,
		Prompt:     prompt,
		RepoPath:   workDir,
		Stage:      "plan",
		EnvVars:    envVars,
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
		fmt.Sprintf("openspec validate \"%s\" --json", input.AgentRunName))
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
		fmt.Sprintf("openspec status --change \"%s\" --json", input.AgentRunName))
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

// ======================================================================
// VerifyRun — Fix 4-9
// ======================================================================

func (a *Activities) VerifyRun(ctx context.Context, input VerifyRunInput) (VerifyRunOutput, error) {
	startTime := time.Now()
	activity.RecordHeartbeat(ctx, "starting verification")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sidecarClient := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// workDir = repo dir (resolved by sidecar), specDir = /workspace for openspec
	workDir := input.RepoPath
	specDir := "/workspace"
	log.Printf("[VerifyRun %s] repoDir=%s specDir=%s", input.AgentRunName, workDir, specDir)

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
		fmt.Sprintf("openspec validate \"%s\" --json", input.ChangeName))
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

	// ── Gate 4: LLM judge ──
	activity.RecordHeartbeat(ctx, "running LLM evaluation")

	gitDiff, _ := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir,
		"git diff HEAD~1 --stat 2>/dev/null || echo 'no git diff available'")

	_, err = sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		AgentRunId: input.AgentRunName + "-verify",
		Prompt: fmt.Sprintf(`Evaluate whether the implementation satisfies the spec.

Git diff summary:
%s

Read the spec files in the openspec change directory and evaluate each WHEN/THEN scenario.
Output your verdict as JSON: {"pass": true/false, "criteria": [{"scenario": "...", "pass": true/false, "explanation": "..."}]}`,
			gitDiff),
		RepoPath: workDir,
		Stage:    "verify",
	}))
	if err != nil {
		log.Printf("LLM judge failed to start: %v", err)
	} else {
		if pollErr := pollUntilAgentDone(ctx, sidecarClient, input.AgentRunName+"-verify"); pollErr != nil {
			log.Printf("LLM judge failed: %v", pollErr)
		} else {
			activity.RecordHeartbeat(ctx, "parsing LLM verdict")
			verdictJSON, err := execInSidecar(ctx, sidecarClient, input.AgentRunName, workDir,
				"cat .aot/logs/agent.jsonl 2>/dev/null | tail -50")
			if err == nil {
				verdict := parseLLMVerdict(verdictJSON)
				if verdict != nil {
					result.LLMVerdict = verdict
					if !verdict.Pass {
						var failedCriteria []string
						for _, c := range verdict.Criteria {
							if !c.Pass {
								failedCriteria = append(failedCriteria, fmt.Sprintf("%s: %s", c.Scenario, c.Explanation))
							}
						}
						result.Pass = false
						result.FailureReport = fmt.Sprintf("LLM judge failed: %s", strings.Join(failedCriteria, "; "))
						return VerifyRunOutput{Result: result}, nil
					}
				}
			}
		}
	}

	// ── Gate 5: Archive ──
	activity.RecordHeartbeat(ctx, "archiving change")

	archiveOut, archiveErr := execInSidecar(ctx, sidecarClient, input.AgentRunName, specDir,
		fmt.Sprintf("openspec archive \"%s\" --yes", input.ChangeName))
	if archiveErr != nil {
		// Fix 8: report archive errors (was swallowed with || true)
		// Archive failure is informational, not a gate blocker
		log.Printf("openspec archive warning: %v (output: %s)", archiveErr, archiveOut)
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

// ======================================================================
// detectTestCommands — Fix 6: actually extract commands (was stub)
// ======================================================================

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

// ======================================================================
// File existence checks — uses spec parsing
// ======================================================================

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
			_ = os.WriteFile(filepath.Join(dir, "verification-result.json"), data, 0o644)
			return
		}
	}

	fallbackDir := filepath.Join(repoPath, ".aot", "verification")
	_ = os.MkdirAll(fallbackDir, 0o755)
	_ = os.WriteFile(filepath.Join(fallbackDir, changeName+"-result.json"), data, 0o644)
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
func pollUntilAgentDone(ctx context.Context, client agentv1connect.AgentSidecarServiceClient, runID string) error {
	unspecifiedCount := 0
	const maxUnspecified = 10

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

		time.Sleep(3 * time.Second)
	}
}
