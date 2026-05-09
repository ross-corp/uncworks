// run.go — uncworks run: submit a new agent run non-interactively.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	repo := fs.String("repo", "", "Git repository URL (required)")
	branch := fs.String("branch", "main", "Branch to check out")
	name := fs.String("name", "", "Display name for the run (auto-generated from prompt if omitted)")
	prompt := fs.String("prompt", "", "Agent prompt describing the task (required)")
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

	startTime := time.Now()
	waitCtx := context.Background()
	var waitCancel context.CancelFunc
	if *timeout > 0 {
		waitCtx, waitCancel = context.WithTimeout(waitCtx, *timeout)
		defer waitCancel()
	}
	var lastPhase apiv1.AgentRunPhase
	var lastStage, lastMsg string
	var needNewline bool
	var dotCount int

	if !*outputID {
		fmt.Printf("Waiting for run %s ", run.GetId())
	}

	for {
		select {
		case <-waitCtx.Done():
			if !*outputID && dotCount > 0 {
				fmt.Println()
			}
			return fmt.Errorf("timed out after %s waiting for run %s", *timeout, run.GetId())
		case <-time.After(10 * time.Second):
		}
		getReq := connect.NewRequest(&apiv1.GetAgentRunRequest{Id: run.GetId()})
		getResp, err := client.GetAgentRun(waitCtx, getReq)
		if err != nil {
			if !*outputID && needNewline {
				fmt.Println()
				needNewline = false
			}
			fmt.Fprintf(os.Stderr, "warn: poll error: %s\n", humanizeErr(err))
			continue
		}
		phase := getResp.Msg.GetStatus().GetPhase()
		msg := getResp.Msg.GetStatus().GetMessage()
		stage := getResp.Msg.GetStatus().GetStage()
		elapsed := int(time.Since(startTime).Seconds())

		if !*outputID {
			fmt.Print(".")
			dotCount++
			needNewline = true
		}

		if phase != lastPhase || stage != lastStage || msg != lastMsg {
			lastPhase = phase
			lastStage = stage
			lastMsg = msg
			if !*outputID && needNewline {
				fmt.Println()
				needNewline = false
			}
			if stage != "" {
				fmt.Printf("  [%s | %ds | stage:%s] %s\n", phaseLabel(phase), elapsed, stage, msg)
			} else {
				fmt.Printf("  [%s | %ds] %s\n", phaseLabel(phase), elapsed, msg)
			}
		}
		switch phase {
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
			if !*outputID && needNewline {
				fmt.Println()
			}
			if url := getResp.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
			return nil
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
			if !*outputID && needNewline {
				fmt.Println()
			}
			if msg != "" {
				return fmt.Errorf("run %s failed: %s", run.GetId(), msg)
			}
			return fmt.Errorf("run %s failed", run.GetId())
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
			if !*outputID && needNewline {
				fmt.Println()
			}
			return fmt.Errorf("run %s was cancelled", run.GetId())
		}
	}
}
