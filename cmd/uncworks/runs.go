// runs.go — uncworks runs: list, get, stream logs, and archive agent runs.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

const runsUsage = `Usage: uncworks runs <subcommand> [flags]

Subcommands:
  list              List recent agent runs
  get <id>          Show full detail for a run
  logs <id>         Stream log output until the run completes
  archive <id>      Mark a run as archived
  unarchive <id>    Remove the archived flag from a run
`

func runRuns(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, runsUsage)
		os.Exit(2)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runRunsList(rest)
	case "get":
		return runRunsGet(rest)
	case "logs":
		return runRunsLogs(rest)
	case "archive":
		return runRunsArchive(rest, true)
	case "unarchive":
		return runRunsArchive(rest, false)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, runsUsage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", sub, runsUsage)
		os.Exit(2)
	}
	return nil
}

// ── list ──────────────────────────────────────────────────────────────────────

func runRunsList(args []string) error {
	fs := flag.NewFlagSet("runs list", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	limit := fs.Int("limit", 20, "Maximum number of runs to return")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs list [flags]\n\nList recent agent runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&apiv1.ListAgentRunsRequest{
		Limit:         int32(*limit),
		ProjectFilter: *project,
		FeatureFilter: *feature,
	})
	resp, err := client.ListAgentRuns(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	runs := resp.Msg.GetAgentRuns()
	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROJECT\tPHASE\tMODEL\tSTARTED")
	for _, r := range runs {
		project := r.GetSpec().GetProject()
		if project == "" {
			project = "-"
		}
		phase := phaseLabel(r.GetStatus().GetPhase())
		model := r.GetSpec().GetModelTier()
		if model == "" {
			model = "default"
		}
		started := "-"
		if r.GetStatus().GetStartedAt() != nil {
			started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
		} else if r.GetCreatedAt() != nil {
			started = r.GetCreatedAt().AsTime().Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.GetId(), project, phase, model, started)
	}
	w.Flush()
	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRunsGet(args []string) error {
	fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs get <id> [flags]\n\nShow full detail for an agent run.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}
	id := fs.Arg(0)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id})
	resp, err := client.GetAgentRun(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	r := resp.Msg
	fmt.Printf("ID:       %s\n", r.GetId())
	fmt.Printf("Name:     %s\n", r.GetName())
	fmt.Printf("Phase:    %s\n", phaseLabel(r.GetStatus().GetPhase()))
	if r.GetStatus().GetMessage() != "" {
		fmt.Printf("Message:  %s\n", r.GetStatus().GetMessage())
	}
	if r.GetSpec().GetProject() != "" {
		fmt.Printf("Project:  %s\n", r.GetSpec().GetProject())
	}
	if r.GetSpec().GetFeature() != "" {
		fmt.Printf("Feature:  %s\n", r.GetSpec().GetFeature())
	}
	if r.GetSpec().GetPrompt() != "" {
		fmt.Printf("Prompt:   %s\n", r.GetSpec().GetPrompt())
	}
	for _, repo := range r.GetSpec().GetRepos() {
		fmt.Printf("Repo:     %s @ %s\n", repo.GetUrl(), repo.GetBranch())
	}
	if r.GetSpec().GetModelTier() != "" {
		fmt.Printf("Model:    %s\n", r.GetSpec().GetModelTier())
	}
	if r.GetStatus().GetStartedAt() != nil {
		fmt.Printf("Started:  %s\n", r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339))
	}
	if r.GetStatus().GetCompletedAt() != nil {
		fmt.Printf("Completed:%s\n", r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339))
		if r.GetStatus().GetStartedAt() != nil {
			dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime()).Round(time.Second)
			fmt.Printf("Duration: %s\n", dur)
		}
	}
	if r.GetStatus().GetPodName() != "" {
		fmt.Printf("Pod:      %s\n", r.GetStatus().GetPodName())
	}
	if r.GetStatus().GetStage() != "" {
		fmt.Printf("Stage:    %s\n", r.GetStatus().GetStage())
	}
	if r.GetStatus().GetPrUrl() != "" {
		fmt.Printf("PR:       %s\n", r.GetStatus().GetPrUrl())
	}
	if len(r.GetChildren()) > 0 {
		fmt.Printf("Children: %v\n", r.GetChildren())
	}
	return nil
}

// ── logs ──────────────────────────────────────────────────────────────────────

func runRunsLogs(args []string) error {
	fs := flag.NewFlagSet("runs logs", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs logs <id> [flags]\n\nStream log output until the run completes.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}
	id := fs.Arg(0)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	// Check if the run is pending before starting the stream.
	getReq := connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id})
	getResp, err := client.GetAgentRun(context.Background(), getReq)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	if getResp.Msg.GetStatus().GetPhase() == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		fmt.Println("waiting for run to start...")
	}

	req := connect.NewRequest(&apiv1.WatchAgentRunRequest{Id: id})
	stream, err := client.WatchAgentRun(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	var finalPhase apiv1.AgentRunPhase
	for stream.Receive() {
		ev := stream.Msg()
		switch ev.GetType() {
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
			if ev.GetPayload() != "" {
				fmt.Print(ev.GetPayload())
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
			fmt.Printf("[phase: %s]\n", ev.GetPayload())
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_WAITING_FOR_INPUT:
			fmt.Printf("[waiting for input: %s]\n", ev.GetPayload())
			fmt.Println("Use 'uncworks input <id> <text>' to respond.")
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
			fmt.Printf("[completed: %s]\n", ev.GetPayload())
		}
	}
	if err := stream.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("stream error: %s", humanizeErr(err))
	}

	// Resolve final phase.
	getResp2, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err == nil {
		finalPhase = getResp2.Msg.GetStatus().GetPhase()
	}

	switch finalPhase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		fmt.Fprintln(os.Stderr, "Run failed.")
		os.Exit(1)
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		fmt.Fprintln(os.Stderr, "Run cancelled.")
		os.Exit(1)
	}
	return nil
}

// ── archive / unarchive ───────────────────────────────────────────────────────

func runRunsArchive(args []string, archived bool) error {
	verb := "archive"
	if !archived {
		verb = "unarchive"
	}
	fs := flag.NewFlagSet("runs "+verb, flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: uncworks runs %s <id> [flags]\n\nFlags:\n", verb)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}
	id := fs.Arg(0)

	body, _ := json.Marshal(map[string]bool{"archived": archived})
	url := serverBaseURL(*server) + "/api/v1/runs/" + id + "/archive"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}

	if archived {
		fmt.Printf("Run %s archived\n", id)
	} else {
		fmt.Printf("Run %s unarchived\n", id)
	}
	return nil
}
