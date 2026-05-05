// run.go — uncworks run: submit a new agent run non-interactively.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	repo := fs.String("repo", "", "Git repository URL (required)")
	branch := fs.String("branch", "main", "Branch to check out")
	prompt := fs.String("prompt", "", "Agent prompt describing the task (required)")
	project := fs.String("project", "", "Project name this run belongs to")
	feature := fs.String("feature", "", "Feature/unit-of-work this run contributes to")
	modelTier := fs.String("model-tier", "", "LLM model tier (e.g. deepseek-v3.2, default-cloud, premium)")
	autoPush := fs.Bool("auto-push", false, "Push changes to a feature branch after the run succeeds")
	autoPR := fs.Bool("auto-pr", false, "Create a GitHub PR after the run succeeds (implies --auto-push)")
	wait := fs.Bool("wait", false, "Wait for the run to complete; exit 0 on success, 1 on failure")
	server := fs.String("server", "", "gRPC server address (overrides config)")
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

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	spec := &apiv1.AgentRunSpec{
		Backend: apiv1.Backend_BACKEND_POD,
		Repos: []*apiv1.Repository{
			{Url: *repo, Branch: *branch},
		},
		Prompt:    *prompt,
		Project:   *project,
		Feature:   *feature,
		ModelTier: *modelTier,
		AutoPush:  *autoPush || *autoPR,
		AutoPr:    *autoPR,
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

	fmt.Printf("Run created: %s\n", run.GetId())
	fmt.Printf("Follow progress: uncworks runs logs %s\n", run.GetId())

	if !*wait {
		return nil
	}

	fmt.Printf("Waiting for run %s to complete...\n", run.GetId())
	startTime := time.Now()
	var lastPhase apiv1.AgentRunPhase
	var lastStage, lastMsg string
	for {
		time.Sleep(10 * time.Second)
		getReq := connect.NewRequest(&apiv1.GetAgentRunRequest{Id: run.GetId()})
		getResp, err := client.GetAgentRun(context.Background(), getReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: poll error: %s\n", humanizeErr(err))
			continue
		}
		phase := getResp.Msg.GetStatus().GetPhase()
		msg := getResp.Msg.GetStatus().GetMessage()
		stage := getResp.Msg.GetStatus().GetStage()
		elapsed := int(time.Since(startTime).Seconds())
		
		// Only print if phase, stage, or message changed
		if phase != lastPhase || stage != lastStage || msg != lastMsg {
			lastPhase = phase
			lastStage = stage
			lastMsg = msg
			if stage != "" {
				fmt.Printf("  [%s | %ds | stage:%s] %s\n", phaseLabel(phase), elapsed, stage, msg)
			} else {
				fmt.Printf("  [%s | %ds] %s\n", phaseLabel(phase), elapsed, msg)
			}
		} else {
			// Show a progress indicator when status hasn't changed
			fmt.Print(".")
			os.Stdout.Sync()
		}
		switch phase {
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
			if url := getResp.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
			return nil
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED, apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
			return fmt.Errorf("run %s ended with phase: %s", run.GetId(), phase)
		}
	}
}
