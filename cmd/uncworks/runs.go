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
	"strings"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

const runsUsage = `Usage: uncworks runs <subcommand> [flags]

Subcommands:
  list              List recent agent runs
  get <id>          Show full detail for a run
  describe <id>     Show full detail including persisted log output
  logs <id>         Stream log output until the run completes
  watch <id>        Alias for logs
  tail <id>         Stream logs and show summary when run completes
  watch <id>        Alias for 'logs' (stream live output)
  archive <id>      Mark a run as archived
  unarchive <id>    Remove the archived flag from a run
  cancel <id>       Request cancellation of a running agent
  stats             Show aggregate counts of runs by phase
  open <id>         Open the PR URL for a completed run in browser
  retry <id>        Create a new run with the same spec as an existing run
  rerun <id>        Alias for retry
  cancel-all        Cancel all active (non-terminal) runs
  graph <id>        Show the run graph (parent/child relationships)
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
	case "describe":
		return runRunsDescribe(rest)
	case "logs":
		return runRunsLogs(rest)
	case "tail":
		return runRunsTail(rest)
	case "watch":
		return runRunsLogs(rest)
	case "archive":
		return runRunsArchive(rest, true)
	case "unarchive":
		return runRunsArchive(rest, false)
	case "cancel":
		return runCancel(rest)
	case "stats":
		return runRunsStats(rest)
	case "open":
		return runRunsOpen(rest)
	case "retry", "rerun":
		return runRunsRetry(rest)
	case "cancel-all":
		return runRunsCancelAll(rest)
	case "graph":
		return runRunsGraph(rest)
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
	includeArchived := fs.Bool("include-archived", false, "Include archived runs")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED)")
	tag := fs.String("tag", "", "Filter by tag")
	parentRunID := fs.String("parent-run-id", "", "Filter by parent run ID")
	cursor := fs.String("cursor", "", "Pagination cursor from previous response")
	jsonOut := fs.Bool("json", false, "Output as JSON")
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

	listReq := &apiv1.ListAgentRunsRequest{
		Limit:         int32(*limit),
		ProjectFilter: *project,
		FeatureFilter: *feature,
	}
	
	if *phase != "" {
		var phaseEnum apiv1.AgentRunPhase
		phaseUpper := strings.ToUpper(*phase)
		switch phaseUpper {
		case "RUNNING":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
		case "DONE":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
		case "FAILED":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
		case "PENDING":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
		case "WAITING":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
		case "CANCELLED":
			phaseEnum = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		default:
			return fmt.Errorf("invalid phase value %q, must be one of: RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED", *phase)
		}
		listReq.PhaseFilter = phaseEnum
	}
	
	if *tag != "" {
		listReq.TagFilter = *tag
	}
	
	if *parentRunID != "" {
		listReq.ParentRunId = *parentRunID
	}
	
	if *cursor != "" {
		listReq.Cursor = *cursor
	}

	req := connect.NewRequest(listReq)
	if *includeArchived {
		req.Header().Set("X-Include-Archived", "true")
	}
	resp, err := client.ListAgentRuns(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	runs := resp.Msg.GetAgentRuns()
	if len(runs) == 0 && !*jsonOut {
		fmt.Println("No runs found.")
		return nil
	}

	if *jsonOut {
		type runJSON struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Phase    string `json:"phase"`
			Duration string `json:"duration"`
			Model    string `json:"model"`
			Started  string `json:"started"`
		}
		out := make([]runJSON, 0, len(runs))
		for _, r := range runs {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			model := r.GetSpec().GetModelTier()
			if model == "" {
				model = "default"
			}
			started := ""
			if r.GetStatus().GetStartedAt() != nil {
				started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
			} else if r.GetCreatedAt() != nil {
				started = r.GetCreatedAt().AsTime().Format(time.RFC3339)
			}
			out = append(out, runJSON{
				ID:       r.GetId(),
				Title:    title,
				Phase:    phaseLabel(r.GetStatus().GetPhase()),
				Duration: runDuration(r),
				Model:    model,
				Started:  started,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tPHASE\tDURATION\tMODEL\tSTARTED")
	for _, r := range runs {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if title == "" {
			title = "-"
		}
		if len(title) > 32 {
			title = title[:29] + "..."
		}
		phase := phaseLabel(r.GetStatus().GetPhase())
		model := r.GetSpec().GetModelTier()
		if model == "" {
			model = "default"
		}
		if len(model) > 15 {
			model = model[:12] + "..."
		}
		started := "-"
		if r.GetStatus().GetStartedAt() != nil {
			started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
		} else if r.GetCreatedAt() != nil {
			started = r.GetCreatedAt().AsTime().Format(time.RFC3339)
		}
		duration := runDuration(r)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.GetId(), title, phase, duration, model, started)
	}
	w.Flush()

	if resp.Msg.GetNextCursor() != "" {
		fmt.Printf("next-cursor: %s\n", resp.Msg.GetNextCursor())
	}

	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRunsGet(args []string) error {
	fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	showLog := fs.Bool("log", false, "Print the persisted agent log output")
	jsonOut := fs.Bool("json", false, "Output as JSON")
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

	if *jsonOut {
		type runGetJSON struct {
			ID        string   `json:"id"`
			Title     string   `json:"title,omitempty"`
			Phase     string   `json:"phase"`
			Message   string   `json:"message,omitempty"`
			Project   string   `json:"project,omitempty"`
			Feature   string   `json:"feature,omitempty"`
			Prompt    string   `json:"prompt,omitempty"`
			Repo      string   `json:"repo,omitempty"`
			Model     string   `json:"model,omitempty"`
			Tags      []string `json:"tags,omitempty"`
			Started   string   `json:"started,omitempty"`
			Completed string   `json:"completed,omitempty"`
			Duration  string   `json:"duration,omitempty"`
			PrURL     string   `json:"pr_url,omitempty"`
		}
		out := runGetJSON{
			ID:      r.GetId(),
			Title:   r.GetSpec().GetDisplayName(),
			Phase:   phaseLabel(r.GetStatus().GetPhase()),
			Message: r.GetStatus().GetMessage(),
			Project: r.GetSpec().GetProject(),
			Feature: r.GetSpec().GetFeature(),
			Prompt:  r.GetSpec().GetPrompt(),
			Model:   r.GetSpec().GetModelTier(),
			Tags:    r.GetSpec().GetTags(),
			PrURL:   r.GetStatus().GetPrUrl(),
		}
		if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
			out.Repo = repos[0].GetUrl() + " @ " + repos[0].GetBranch()
		}
		if r.GetStatus().GetStartedAt() != nil {
			out.Started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
		}
		if r.GetStatus().GetCompletedAt() != nil {
			out.Completed = r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339)
			if r.GetStatus().GetStartedAt() != nil {
				dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime())
				out.Duration = formatDuration(dur)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("ID:       %s\n", r.GetId())
	if dn := r.GetSpec().GetDisplayName(); dn != "" {
		fmt.Printf("Title:    %s\n", dn)
	}
	fmt.Printf("Phase:    %s\n", phaseLabel(r.GetStatus().GetPhase()))
	if r.GetStatus().GetMessage() != "" {
		msg := r.GetStatus().GetMessage()
		if r.GetStatus().GetPhase() == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED {
			if term.IsTerminal(int(os.Stdout.Fd())) {
				fmt.Printf("Message:  \033[1;31m%s\033[0m\n", msg)
			} else {
				fmt.Printf("ERROR:    %s\n", msg)
			}
		} else {
			fmt.Printf("Message:  %s\n", msg)
		}
	}
	if r.GetSpec().GetProject() != "" {
		fmt.Printf("Project:  %s\n", r.GetSpec().GetProject())
	}
	if r.GetSpec().GetFeature() != "" {
		fmt.Printf("Feature:  %s\n", r.GetSpec().GetFeature())
	}
	if len(r.GetSpec().GetTags()) > 0 {
		fmt.Printf("Tags:     %s\n", strings.Join(r.GetSpec().GetTags(), ", "))
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
			dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime())
			fmt.Printf("Duration: %s\n", formatDuration(dur))
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
	if *showLog && r.GetStatus().GetLogOutput() != "" {
		fmt.Printf("\n--- agent log ---\n%s\n", r.GetStatus().GetLogOutput())
	}
	return nil
}

func runRunsDescribe(args []string) error {
	return runRunsGet(append(args, "--log"))
}

func runRunsTail(args []string) error {
	fs := flag.NewFlagSet("runs tail", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs tail <id> [flags]\n\nStream logs and show a summary when the run completes.\n\nFlags:")
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

	if err := runRunsLogs([]string{id, "--server=" + *server}); err != nil {
		return err
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}
	resp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	r := resp.Msg
	fmt.Printf("\n─── summary ───────────────────────────────────────────────────────────────\n")
	fmt.Printf("Phase:    %s\n", phaseLabel(r.GetStatus().GetPhase()))
	if r.GetStatus().GetStartedAt() != nil && r.GetStatus().GetCompletedAt() != nil {
		dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime()).Round(time.Second)
		fmt.Printf("Duration: %s\n", dur)
	}
	if r.GetStatus().GetPrUrl() != "" {
		fmt.Printf("PR:       %s\n", r.GetStatus().GetPrUrl())
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
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		fmt.Println("Run succeeded.")
		if getResp2 != nil {
			if url := getResp2.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
		}
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

// ── stats ─────────────────────────────────────────────────────────────────────

func runRunsStats(args []string) error {
	fs := flag.NewFlagSet("runs stats", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs stats [flags]\n\nShow aggregate counts of agent runs by phase.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	counts := map[string]int{
		"PENDING":   0,
		"RUNNING":   0,
		"WAITING":   0,
		"DONE":      0,
		"FAILED":    0,
		"CANCELLED": 0,
	}
	order := []string{"RUNNING", "PENDING", "WAITING", "DONE", "FAILED", "CANCELLED"}

	cursor := ""
	total := 0
	for {
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			Cursor:        cursor,
		}
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			label := phaseLabel(r.GetStatus().GetPhase())
			counts[label]++
			total++
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	fmt.Printf("Total: %d\n\n", total)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PHASE\tCOUNT")
	for _, phase := range order {
		fmt.Fprintf(w, "%s\t%d\n", phase, counts[phase])
	}
	_ = w.Flush()
	return nil
}

// ── open ────────────────────────────────────────────────────────────────────────

func runRunsOpen(args []string) error {
	fs := flag.NewFlagSet("runs open", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs open <id> [flags]\n\nOpen the PR URL for a completed agent run in the default browser.\n\nFlags:")
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
	prURL := r.GetStatus().GetPrUrl()
	if prURL == "" {
		return fmt.Errorf("run %s has no PR URL", id)
	}

	fmt.Printf("Opening PR URL: %s\n", prURL)
	if err := openBrowser(prURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// ── retry ────────────────────────────────────────────────────────────────────

func runRunsRetry(args []string) error {
	fs := flag.NewFlagSet("runs retry", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs retry <id> [flags]\n\nCreate a new run with the same spec as an existing run.\n\nFlags:")
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

	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	orig := getResp.Msg
	spec := orig.GetSpec()
	if spec == nil {
		return fmt.Errorf("run %s has no spec", id)
	}

	newSpec := &apiv1.AgentRunSpec{
		Backend:   spec.Backend,
		Repos:     spec.Repos,
		Prompt:    spec.Prompt,
		Project:   spec.Project,
		Feature:   spec.Feature,
		ModelTier: spec.ModelTier,
		Tags:      spec.Tags,
		AutoPush:  spec.AutoPush,
		AutoPr:    spec.AutoPr,
	}

	createResp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{Spec: newSpec}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	newRun := createResp.Msg.GetAgentRun()
	fmt.Printf("Run created: %s\n", newRun.GetId())
	fmt.Printf("Follow progress: uncworks runs logs %s\n", newRun.GetId())
	return nil
}

// ── cancel-all ───────────────────────────────────────────────────────────────

func runRunsCancelAll(args []string) error {
	fs := flag.NewFlagSet("runs cancel-all", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	dryRun := fs.Bool("dry-run", false, "Print what would be cancelled without actually doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs cancel-all [flags]\n\nCancel all active (non-terminal) runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	// Collect all active runs by paginating through the list
	var activeRuns []string
	var cursor string
	for {
		req := connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:  100,
			Cursor: cursor,
		})
		resp, err := client.ListAgentRuns(context.Background(), req)
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			phase := r.GetStatus().GetPhase()
			if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING ||
				phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING ||
				phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
				activeRuns = append(activeRuns, r.GetId())
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if len(activeRuns) == 0 {
		fmt.Println("No active runs to cancel.")
		return nil
	}

	if *dryRun {
		fmt.Printf("Would cancel %d run(s):\n", len(activeRuns))
		for _, id := range activeRuns {
			fmt.Printf("  %s\n", id)
		}
		return nil
	}

	if !*yes {
		fmt.Printf("Active runs to cancel:\n")
		for _, id := range activeRuns {
			fmt.Printf("  %s\n", id)
		}
		fmt.Printf("Cancel %d run(s)? [y/N]: ", len(activeRuns))
		var answer string
		if _, err := fmt.Scanln(&answer); err != nil {
			answer = ""
		}
		if answer != "y" && answer != "Y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	cancelled := 0
	for _, id := range activeRuns {
		_, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: id}))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  failed to cancel %s: %s\n", id, humanizeErr(err))
		} else {
			fmt.Printf("  cancelled %s\n", id)
			cancelled++
		}
	}
	fmt.Printf("Cancelled %d/%d run(s).\n", cancelled, len(activeRuns))
	return nil
}

// ── graph ─────────────────────────────────────────────────────────────────────

func runRunsGraph(args []string) error {
	fs := flag.NewFlagSet("runs graph", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs graph <id> [flags]\n\nPrint the execution tree for a run.\n\nFlags:")
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

	req := connect.NewRequest(&apiv1.GetRunGraphRequest{Id: id})
	resp, err := client.GetRunGraph(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	printGraph(id, resp.Msg)
	return nil
}
