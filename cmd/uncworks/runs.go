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
	"sort"
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
  tail <id>         Stream logs and show summary when run completes
  watch             Auto-refresh the runs list (like watch kubectl get pods)
  archive <id>      Mark a run as archived
  unarchive <id>    Remove the archived flag from a run
  archive-done      Bulk archive all SUCCEEDED runs
  cancel <id>       Request cancellation of a running agent
  kill <id>         Alias for cancel
  stats             Show aggregate counts of runs by phase
  open <id>         Open the PR URL for a completed run in browser
  retry <id>        Create a new run with the same spec as an existing run
  rerun <id>        Alias for retry
  copy <id>         Alias for retry
  cancel-all        Cancel all active (non-terminal) runs
  graph <id>        Show the run graph (parent/child relationships)
  latest            Show the most recent run in detail
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
		return runRunsWatch(rest)
	case "archive":
		return runRunsArchive(rest, true)
	case "unarchive":
		return runRunsArchive(rest, false)
	case "archive-done":
		return runRunsArchiveDone(rest)
	case "cancel", "kill":
		return runCancel(rest)
	case "stats":
		return runRunsStats(rest)
	case "open":
		return runRunsOpen(rest)
	case "retry", "rerun", "copy":
		return runRunsRetry(rest)
	case "cancel-all":
		return runRunsCancelAll(rest)
	case "graph":
		return runRunsGraph(rest)
	case "latest":
		return runRunsLatest(rest)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, runsUsage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", sub, runsUsage)
		os.Exit(2)
	}
	return nil
}

// ── watch ─────────────────────────────────────────────────────────────────────

func runRunsWatch(args []string) error {
	fs := flag.NewFlagSet("runs watch", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	interval := fs.Int("interval", 5, "Refresh interval in seconds")
	limit := fs.Int("limit", 20, "Max runs to show per refresh")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs watch [flags]\n\nAuto-refresh the runs list every N seconds. Press Ctrl+C to stop.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *interval < 1 {
		return fmt.Errorf("--interval must be >= 1")
	}

	listArgs := []string{"--limit", fmt.Sprintf("%d", *limit)}
	if *server != "" {
		listArgs = append(listArgs, "--server="+*server)
	}
	if *since != "" {
		listArgs = append(listArgs, "--since="+*since)
	}
	if *phase != "" {
		listArgs = append(listArgs, "--phase="+*phase)
	}
	if *project != "" {
		listArgs = append(listArgs, "--project="+*project)
	}
	if *feature != "" {
		listArgs = append(listArgs, "--feature="+*feature)
	}

	for {
		fmt.Print("\033[H\033[2J") // clear screen + move cursor home
		fmt.Printf("uncworks runs watch — every %ds  %s  (Ctrl+C to stop)\n\n",
			*interval, time.Now().Format("15:04:05"))
		_ = runRunsList(listArgs)
		time.Sleep(time.Duration(*interval) * time.Second)
	}
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
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	all := fs.Bool("all", false, "Fetch all pages (overrides --limit)")
	repoURL := fs.String("repo-url", "", "Filter runs by repository URL (substring match)")
	verbose := fs.Bool("verbose", false, "Show extra columns (repo, project)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs list [flags]\n\nList recent agent runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var sinceTime time.Time
	if *since != "" {
		d, err := parseSinceDuration(*since)
		if err != nil {
			return fmt.Errorf("--since %q: %w", *since, err)
		}
		sinceTime = time.Now().Add(-d)
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

	var runs []*apiv1.AgentRun
	var nextCursor string
	fetchCursor := *cursor
	for {
		listReq.Cursor = fetchCursor
		req := connect.NewRequest(listReq)
		if *includeArchived {
			req.Header().Set("X-Include-Archived", "true")
		}
		resp, err := client.ListAgentRuns(context.Background(), req)
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		runs = append(runs, resp.Msg.GetAgentRuns()...)
		nextCursor = resp.Msg.GetNextCursor()
		if !*all || nextCursor == "" {
			break
		}
		fetchCursor = nextCursor
	}

	if !sinceTime.IsZero() {
		filtered := runs[:0]
		for _, r := range runs {
			ts := r.GetCreatedAt()
			if ts != nil && ts.AsTime().After(sinceTime) {
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}
	if *repoURL != "" {
		filtered := runs[:0]
		for _, r := range runs {
			for _, repo := range r.GetSpec().GetRepos() {
				if strings.Contains(repo.GetUrl(), *repoURL) {
					filtered = append(filtered, r)
					break
				}
			}
		}
		runs = filtered
	}
	if len(runs) == 0 && !*jsonOut {
		fmt.Println("No runs found.")
		return nil
	}

	if *jsonOut {
		type runJSON struct {
			ID       string   `json:"id"`
			Title    string   `json:"title"`
			Phase    string   `json:"phase"`
			Duration string   `json:"duration"`
			Model    string   `json:"model"`
			Started  string   `json:"started"`
			Project  string   `json:"project,omitempty"`
			Feature  string   `json:"feature,omitempty"`
			Tags     []string `json:"tags,omitempty"`
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
				Project:  r.GetSpec().GetProject(),
				Feature:  r.GetSpec().GetFeature(),
				Tags:     r.GetSpec().GetTags(),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	var listBuf bytes.Buffer
	w := tabwriter.NewWriter(&listBuf, 0, 0, 2, ' ', 0)
	if *verbose {
		fmt.Fprintln(w, "ID\tTITLE\tPHASE\tDURATION\tMODEL\tSTARTED\tREPO\tPROJECT")
	} else {
		fmt.Fprintln(w, "ID\tTITLE\tPHASE\tDURATION\tMODEL\tSTARTED")
	}
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
		if *verbose {
			repo := "—"
			if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
				repo = repos[0].GetUrl()
				if len(repo) > 40 {
					repo = repo[:37] + "..."
				}
			}
			project := r.GetSpec().GetProject()
			if project == "" {
				project = "—"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.GetId(), title, phase, duration, model, started, repo, project)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.GetId(), title, phase, duration, model, started)
		}
	}
	w.Flush()

	output := listBuf.String()
	if useColor {
		output = strings.NewReplacer(
			"RUNNING  ", "\033[32mRUNNING\033[0m  ",
			"RUNNING\t", "\033[32mRUNNING\033[0m\t",
			"PENDING  ", "\033[33mPENDING\033[0m  ",
			"PENDING\t", "\033[33mPENDING\033[0m\t",
			"WAITING  ", "\033[36mWAITING\033[0m  ",
			"WAITING\t", "\033[36mWAITING\033[0m\t",
			"FAILED   ", "\033[31mFAILED\033[0m   ",
			"FAILED\t", "\033[31mFAILED\033[0m\t",
			"DONE     ", "\033[90mDONE\033[0m     ",
			"DONE\t", "\033[90mDONE\033[0m\t",
			"CANCELLED", "\033[35mCANCELLED\033[0m",
		).Replace(output)
	}
	fmt.Print(output)

	if nextCursor != "" && !*all {
		fmt.Printf("next-cursor: %s\n", nextCursor)
	}

	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRunsGet(args []string) error {
	fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	showLog := fs.Bool("log", false, "Print the persisted agent log output")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
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
			ID          string   `json:"id"`
			Title       string   `json:"title,omitempty"`
			Phase       string   `json:"phase"`
			Message     string   `json:"message,omitempty"`
			Project     string   `json:"project,omitempty"`
			Feature     string   `json:"feature,omitempty"`
			Prompt      string   `json:"prompt,omitempty"`
			Repo        string   `json:"repo,omitempty"`
			Branch      string   `json:"branch,omitempty"`
			Model       string   `json:"model,omitempty"`
			Tags        []string `json:"tags,omitempty"`
			ParentRunID string   `json:"parent_run_id,omitempty"`
			Started     string   `json:"started,omitempty"`
			Completed   string   `json:"completed,omitempty"`
			Duration    string   `json:"duration,omitempty"`
			PrURL       string   `json:"pr_url,omitempty"`
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
			out.Repo = repos[0].GetUrl()
			out.Branch = repos[0].GetBranch()
		}
		out.ParentRunID = r.GetSpec().GetParentRunId()
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
			if !*noColor && term.IsTerminal(int(os.Stdout.Fd())) {
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
		fmt.Printf("Completed: %s\n", r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339))
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
	if r.GetSpec().GetParentRunId() != "" {
		fmt.Printf("Parent:   %s\n", r.GetSpec().GetParentRunId())
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
	noFollow := fs.Bool("no-follow", false, "Print stored log output only (don't stream live)")
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

	if *noFollow {
		logOutput := getResp.Msg.GetStatus().GetLogOutput()
		if logOutput == "" {
			fmt.Fprintf(os.Stderr, "No stored log output for run %s\n", id)
			return nil
		}
		fmt.Print(logOutput)
		return nil
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

func runRunsArchiveDone(args []string) error {
	fs := flag.NewFlagSet("runs archive-done", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	dryRun := fs.Bool("dry-run", false, "Print what would be archived without doing it")
	minAge := fs.Duration("min-age", 0, "Only archive runs completed longer ago than this (e.g. 24h, 7d)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs archive-done [flags]\n\nBulk archive all SUCCEEDED runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var minAgeThreshold time.Time
	if *minAge > 0 {
		minAgeThreshold = time.Now().Add(-*minAge)
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	var doneRuns []string
	var cursor string
	for {
		req := connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			PhaseFilter:   apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			Cursor:        cursor,
		})
		resp, err := client.ListAgentRuns(context.Background(), req)
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			if !minAgeThreshold.IsZero() {
				completedAt := r.GetStatus().GetCompletedAt()
				if completedAt == nil || !completedAt.AsTime().Before(minAgeThreshold) {
					continue
				}
			}
			doneRuns = append(doneRuns, r.GetId())
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if len(doneRuns) == 0 {
		fmt.Println("No SUCCEEDED runs to archive.")
		return nil
	}

	if *dryRun {
		fmt.Printf("Would archive %d run(s):\n", len(doneRuns))
		for _, id := range doneRuns {
			fmt.Printf("  %s\n", id)
		}
		return nil
	}

	archived := 0
	archiveBody, _ := json.Marshal(map[string]bool{"archived": true})
	for _, id := range doneRuns {
		url := serverBaseURL(*server) + "/api/v1/runs/" + id + "/archive"
		archReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(archiveBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  failed to build request for %s: %v\n", id, err)
			continue
		}
		archReq.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(archReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  failed to archive %s: %v\n", id, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "  failed to archive %s: status %d\n", id, resp.StatusCode)
			continue
		}
		archived++
	}
	fmt.Printf("Archived %d/%d run(s).\n", archived, len(doneRuns))
	return nil
}

// ── stats ─────────────────────────────────────────────────────────────────────

func runRunsStats(args []string) error {
	fs := flag.NewFlagSet("runs stats", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	format := fs.String("format", "table", "Output format (table|json)")
	limit := fs.Int("limit", 0, "Count only the N most recent runs (0 = all)")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs stats [flags]\n\nShow aggregate counts of agent runs by phase.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *format != "table" && *format != "json" {
		return fmt.Errorf("invalid format %q: must be table or json", *format)
	}

	var sinceTime time.Time
	if *since != "" {
		d, err := parseSinceDuration(*since)
		if err != nil {
			return fmt.Errorf("--since %q: %w", *since, err)
		}
		sinceTime = time.Now().Add(-d)
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
	var doneDurations []time.Duration
	for {
		pageSize := int32(100)
		if *limit > 0 && *limit-total < 100 {
			pageSize = int32(*limit - total)
		}
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         pageSize,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			Cursor:        cursor,
		}
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetStatus().GetStartedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					continue
				}
			}
			label := phaseLabel(r.GetStatus().GetPhase())
			counts[label]++
			total++
			if label == "DONE" {
				sa := r.GetStatus().GetStartedAt()
				ca := r.GetStatus().GetCompletedAt()
				if sa != nil && ca != nil {
					doneDurations = append(doneDurations, ca.AsTime().Sub(sa.AsTime()))
				}
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || (*limit > 0 && total >= *limit) {
			break
		}
	}

	medianDuration := func() time.Duration {
		if len(doneDurations) == 0 {
			return -1
		}
		sorted := make([]time.Duration, len(doneDurations))
		copy(sorted, doneDurations)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		mid := len(sorted) / 2
		if len(sorted)%2 == 0 {
			return (sorted[mid-1] + sorted[mid]) / 2
		}
		return sorted[mid]
	}()

	done := counts["DONE"]
	failed := counts["FAILED"]

	if *format == "json" {
		successRate := 0.0
		if done+failed > 0 {
			successRate = float64(done) / float64(done+failed) * 100
		}
		out := map[string]interface{}{
			"total":        total,
			"phases":       counts,
			"success_rate": successRate,
		}
		if medianDuration >= 0 {
			out["median_duration_seconds"] = medianDuration.Seconds()
		}
		if *since != "" {
			out["window"] = *since
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	header := "Stats (all time)"
	if *since != "" {
		header = fmt.Sprintf("Stats (last %s)", *since)
	}
	fmt.Printf("%s — Total: %d\n\n", header, total)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PHASE\tCOUNT\tPCT")
	for _, phase := range order {
		pct := 0.0
		if total > 0 {
			pct = float64(counts[phase]) / float64(total) * 100
		}
		fmt.Fprintf(w, "%s\t%d\t%.1f%%\n", phase, counts[phase], pct)
	}
	_ = w.Flush()

	if done+failed > 0 {
		rate := float64(done) / float64(done+failed) * 100
		fmt.Printf("\nSuccess rate: %.1f%% (%d/%d completed runs)\n", rate, done, done+failed)
	}
	if medianDuration >= 0 {
		fmt.Printf("Median duration: %s\n", medianDuration.Round(time.Second))
	} else if done > 0 {
		fmt.Printf("Median duration: —\n")
	}
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
		// No PR URL — show branch info if available
		if repos := r.GetSpec().GetRepos(); len(repos) > 0 && r.GetSpec().GetAutoPush() {
			fmt.Printf("Run %s: no PR created — branch was pushed from %s\n", id, repos[0].GetUrl())
		} else {
			return fmt.Errorf("run %s has no PR — was --auto-pr used?", id)
		}
		return nil
	}

	fmt.Printf("Opening PR: %s\n", prURL)
	if err := openBrowser(prURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// ── retry ────────────────────────────────────────────────────────────────────

func runRunsRetry(args []string) error {
	fs := flag.NewFlagSet("runs retry", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	prompt := fs.String("prompt", "", "Override the agent prompt")
	branch := fs.String("branch", "", "Override the branch")
	modelTier := fs.String("model-tier", "", "Override the model tier")
	name := fs.String("name", "", "Override the display name")
	outputID := fs.Bool("output-id", false, "Print only the new run ID (for scripting)")
	wait := fs.Bool("wait", false, "Wait for the retried run to complete; exit 0 on success, 1 on failure")
	var envFlags multiFlag
	fs.Var(&envFlags, "env", "Override environment variables (repeatable, KEY=VALUE); replaces all env vars if any are provided")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs retry <id> [flags]\n\nCreate a new run with the same spec as an existing run. Use flags to override specific fields.\n\nFlags:")
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

	if *prompt != "" {
		newSpec.Prompt = *prompt
	}
	if *branch != "" && len(newSpec.Repos) > 0 {
		newSpec.Repos[0].Branch = *branch
	}
	if *modelTier != "" {
		newSpec.ModelTier = *modelTier
	}
	if *name != "" {
		newSpec.DisplayName = *name
	}
	if len(envFlags) > 0 {
		envVars := map[string]string{}
		for _, kv := range envFlags {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("--env %q: must be KEY=VALUE", kv)
			}
			envVars[parts[0]] = parts[1]
		}
		newSpec.EnvVars = envVars
	}

	createResp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{Spec: newSpec}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	newRun := createResp.Msg.GetAgentRun()
	if *outputID {
		fmt.Println(newRun.GetId())
	} else {
		fmt.Printf("Run created: %s\n", newRun.GetId())
		if !*wait {
			fmt.Printf("Follow progress: uncworks runs tail %s\n", newRun.GetId())
		}
	}
	if *wait {
		return runRunsTail([]string{newRun.GetId(), "--server=" + *server})
	}
	return nil
}

// ── cancel-all ───────────────────────────────────────────────────────────────

func runRunsCancelAll(args []string) error {
	fs := flag.NewFlagSet("runs cancel-all", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	dryRun := fs.Bool("dry-run", false, "Print what would be cancelled without actually doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	limit := fs.Int("limit", 0, "Cancel at most N runs (0 = no limit)")
	since := fs.String("since", "", "Only cancel runs created within this window (e.g. 1h, 24h, 7d)")
	phaseFilter := fs.String("phase", "", "Only cancel runs in this phase (RUNNING, PENDING, WAITING)")
	project := fs.String("project", "", "Only cancel runs in this project")
	feature := fs.String("feature", "", "Only cancel runs for this feature")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs cancel-all [flags]\n\nCancel all active (non-terminal) runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var sinceTime time.Time
	if *since != "" {
		d, err := parseSinceDuration(*since)
		if err != nil {
			return fmt.Errorf("--since %q: %w", *since, err)
		}
		sinceTime = time.Now().Add(-d)
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
			Limit:         100,
			Cursor:        cursor,
			ProjectFilter: *project,
			FeatureFilter: *feature,
		})
		resp, err := client.ListAgentRuns(context.Background(), req)
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			phase := r.GetStatus().GetPhase()
			isActive := phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING ||
				phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING ||
				phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
			if !isActive {
				continue
			}
			if *phaseFilter != "" {
				var wantPhase apiv1.AgentRunPhase
				switch strings.ToUpper(*phaseFilter) {
				case "RUNNING":
					wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
				case "PENDING":
					wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
				case "WAITING":
					wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
				}
				if phase != wantPhase {
					continue
				}
			}
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					continue
				}
			}
			activeRuns = append(activeRuns, r.GetId())
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if *limit > 0 && len(activeRuns) > *limit {
		activeRuns = activeRuns[:*limit]
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
	// Delegate to the top-level graph command which has full --json support
	return runGraph(args)
}

func runRunsLatest(args []string) error {
	fs := flag.NewFlagSet("runs latest", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, etc.)")
	project := fs.String("project", "", "Filter by project name")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs latest [flags]\n\nShow the most recent agent run in detail.\n\nFlags:")
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
		Limit:         1,
		ProjectFilter: *project,
	}
	if *phase != "" {
		switch strings.ToUpper(*phase) {
		case "RUNNING":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
		case "DONE":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
		case "FAILED":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
		case "PENDING":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
		case "CANCELLED":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	runs := resp.Msg.GetAgentRuns()
	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	getArgs := []string{runs[0].GetId()}
	if *server != "" {
		getArgs = append(getArgs, "--server="+*server)
	}
	if *jsonOut {
		getArgs = append(getArgs, "--json")
	}
	return runRunsGet(getArgs)
}

// parseSinceDuration parses a human duration like "1h", "24h", "7d".
// Standard time.ParseDuration handles h/m/s; "d" is handled manually.
func parseSinceDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(n, "%d", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid duration %q: days must be a positive integer", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
