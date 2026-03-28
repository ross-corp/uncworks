// run.go — uncworks run: submit a new agent run non-interactively.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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
		Prompt:  *prompt,
		Project: *project,
		Feature: *feature,
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
	return nil
}
