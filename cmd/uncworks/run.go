// run.go — uncworks run: submit a new agent run non-interactively.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// multiFlag collects repeated flag values (e.g. --tag foo --tag bar).
type multiFlag []string

func (f *multiFlag) String() string { return strings.Join(*f, ",") }
func (f *multiFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	repo := fs.String("repo", "", "Git repository URL (auto-detected from git remote if omitted)")
	branch := fs.String("branch", "", "Branch to check out (auto-detected from current branch if omitted; fallback: main)")
	name := fs.String("name", "", "Display name for the run (auto-generated from prompt if omitted)")
	prompt := fs.String("prompt", "", "Agent prompt describing the task (required; use '-' to read from stdin)")
	promptFile := fs.String("prompt-file", "", "Read the agent prompt from a file")
	project := fs.String("project", "", "Project name this run belongs to")
	feature := fs.String("feature", "", "Feature/unit-of-work this run contributes to")
	modelTier := fs.String("model-tier", "", "LLM model tier (e.g. deepseek-v3.2, default-cloud, premium)")
	autoPush := fs.Bool("auto-push", false, "Push changes to a feature branch after the run succeeds")
	autoPR := fs.Bool("auto-pr", false, "Create a GitHub PR after the run succeeds (implies --auto-push)")
	wait := fs.Bool("wait", false, "Wait for the run to complete; exit 0 on success, 1 on failure")
	follow := fs.Bool("follow", false, "Stream logs after submitting the run (takes precedence over --wait)")
	timeout := fs.Duration("timeout", 0, "Timeout for --wait mode (e.g. 30m, 1h); 0 means no timeout")
	server := fs.String("server", "", "gRPC server address (overrides config)")
	var tags multiFlag
	fs.Var(&tags, "tag", "Freeform tag for filtering (repeatable, e.g. --tag ci --tag infra)")
	parentRunID := fs.String("parent-run-id", "", "Parent run ID to link this run as a child")
	var envFlags multiFlag
	fs.Var(&envFlags, "env", "Environment variable for the agent pod (repeatable, KEY=VALUE)")
	outputID := fs.Bool("output-id", false, "Print only the run ID (for scripting)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks run --repo <url> --prompt <text> [flags]

Submit a new agent run and print the run ID.

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	// Allow reading prompt from a file.
	if *promptFile != "" {
		raw, err := os.ReadFile(*promptFile)
		if err != nil {
			return fmt.Errorf("reading prompt file %q: %w", *promptFile, err)
		}
		*prompt = strings.TrimRight(string(raw), "\n")
	}

	// Allow reading prompt from stdin when --prompt is "-".
	if *prompt == "-" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading prompt from stdin: %w", err)
		}
		*prompt = strings.TrimRight(string(raw), "\n")
	}

	// Auto-detect repo from git origin if not specified.
	if *repo == "" {
		if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
			*repo = strings.TrimSpace(string(out))
		}
	}

	// Auto-detect current branch if not specified.
	if *branch == "" {
		if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
			b := strings.TrimSpace(string(out))
			if b != "" && b != "HEAD" {
				*branch = b
			}
		}
		if *branch == "" {
			*branch = "main"
		}
	}

	if *repo == "" || *prompt == "" {
		fs.Usage()
		return fmt.Errorf("--repo and --prompt are required")
	}

	envVars := map[string]string{}
	for _, kv := range envFlags {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--env %q: must be KEY=VALUE", kv)
		}
		envVars[parts[0]] = parts[1]
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	spec := &apiv1.AgentRunSpec{
		Backend: apiv1.Backend_BACKEND_POD,
		Repos: []*apiv1.Repository{
			{Url: *repo, Branch: *branch},
		},
		Prompt:      *prompt,
		DisplayName: *name,
		Project:     *project,
		Feature:     *feature,
		ModelTier:   *modelTier,
		AutoPush:    *autoPush || *autoPR,
		AutoPr:      *autoPR,
		Tags:        []string(tags),
		ParentRunId: *parentRunID,
		EnvVars:     envVars,
	}

	req := connect.NewRequest(&apiv1.CreateAgentRunRequest{Spec: spec})
	resp, err := client.CreateAgentRun(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	run := resp.Msg.GetAgentRun()
	if run == nil {
		return fmt.Errorf("server returned empty run")
	}

	if *outputID {
		fmt.Println(run.GetId())
	} else {
		fmt.Printf("Run created: %s\n", run.GetId())
		fmt.Printf("  repo:   %s\n", *repo)
		fmt.Printf("  branch: %s\n", *branch)
		if *project != "" {
			fmt.Printf("  project: %s\n", *project)
		}
	}

	if *follow {
		return runRunsTail([]string{run.GetId(), "--server=" + *server})
	}

	if !*wait {
		if !*outputID {
			fmt.Printf("Follow progress: uncworks runs tail %s\n", run.GetId())
		}
		return nil
	}

	waitCtx := context.Background()
	var waitCancel context.CancelFunc
	if *timeout > 0 {
		waitCtx, waitCancel = context.WithTimeout(waitCtx, *timeout)
		defer waitCancel()
	}

	if !*outputID {
		fmt.Printf("Waiting for run %s\n", run.GetId())
	}

	stream, err := client.WatchAgentRun(waitCtx, connect.NewRequest(&apiv1.WatchAgentRunRequest{Id: run.GetId()}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	startTime := time.Now()
	for stream.Receive() {
		ev := stream.Msg()
		elapsed := int(time.Since(startTime).Seconds())
		switch ev.GetType() {
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
			if !*outputID {
				fmt.Printf("  [%ds] phase: %s\n", elapsed, ev.GetPayload())
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_WAITING_FOR_INPUT:
			if !*outputID {
				fmt.Printf("  [%ds] waiting for input: %s\n", elapsed, ev.GetPayload())
				fmt.Printf("  Use 'uncworks input %s <text>' to respond.\n", run.GetId())
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
			// Stream will close; fall through to terminal check below.
		}
	}
	if err := stream.Err(); err != nil {
		if waitCtx.Err() != nil {
			return fmt.Errorf("timed out after %s waiting for run %s", *timeout, run.GetId())
		}
		return fmt.Errorf("stream error: %s", humanizeErr(err))
	}

	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: run.GetId()}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	phase := getResp.Msg.GetStatus().GetPhase()
	msg := getResp.Msg.GetStatus().GetMessage()

	switch phase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		if !*outputID {
			fmt.Printf("run %s done\n", run.GetId())
			if url := getResp.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
		}
		return nil
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		if msg != "" {
			return fmt.Errorf("run %s failed: %s", run.GetId(), msg)
		}
		return fmt.Errorf("run %s failed", run.GetId())
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		return fmt.Errorf("run %s was cancelled", run.GetId())
	default:
		return fmt.Errorf("run %s ended in unexpected phase: %s", run.GetId(), phaseLabel(phase))
	}
}
