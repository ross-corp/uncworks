// runs.go — uncworks runs: list, get, stream logs, and archive agent runs.
package main

import (
	"bytes"
	"context"
	"encoding/csv"
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
  archive-failed    Bulk archive all FAILED runs
  prune             Bulk archive all terminal runs older than a given age
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
  count             Print a count of runs (by phase or total)
  export            Export runs as CSV
  diff <id>         Show git commands to inspect a run's diff
  inspect <id>      Diagnostic view: details, graph, and log tail
  wait <id>         Block until a run reaches a terminal phase (exit 1 on failure)
  retry-failed      Bulk retry all FAILED runs matching filters
  summary           Show a dashboard summary of recent run activity
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
	case "archive-failed":
		return runRunsArchiveFailed(rest)
	case "prune":
		return runRunsPrune(rest)
	case "cancel", "kill":
		return runCancel(rest)
	case "stats":
		return runRunsStats(rest)
	case "open":
		return runRunsOpen(rest)
	case "retry", "rerun", "copy":
		return runRunsRetry(rest)
	case "cancel-all", "kill-all":
		return runRunsCancelAll(rest)
	case "graph":
		return runRunsGraph(rest)
	case "latest":
		return runRunsLatest(rest)
	case "count":
		return runRunsCount(rest)
	case "export":
		return runRunsExport(rest)
	case "diff":
		return runRunsDiff(rest)
	case "inspect":
		return runRunsInspect(rest)
	case "wait":
		return runRunsWait(rest)
	case "retry-failed":
		return runRunsRetryFailed(rest)
	case "summary":
		return runRunsSummary(rest)
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
	tag := fs.String("tag", "", "Filter by tag")
	titleContains := fs.String("title-contains", "", "Filter runs by display name substring")
	active := fs.Bool("active", false, "Show only active runs (RUNNING + PENDING + WAITING)")
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

	listArgs := []string{"--limit", fmt.Sprintf("%d", *limit), "--relative"}
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
	if *tag != "" {
		listArgs = append(listArgs, "--tag="+*tag)
	}
	if *titleContains != "" {
		listArgs = append(listArgs, "--title-contains="+*titleContains)
	}
	if *active {
		listArgs = append(listArgs, "--active")
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
	titleContains := fs.String("title-contains", "", "Filter runs by display name substring (case-insensitive)")
	verbose := fs.Bool("verbose", false, "Show extra columns (repo, project)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	relative := fs.Bool("relative", false, "Show relative timestamps (e.g. '5m ago') instead of ISO")
	sortBy := fs.String("sort", "", "Sort by field: started, phase (default: server order / most-recent-first)")
	idsOnly := fs.Bool("ids-only", false, "Print only run IDs (one per line, for scripting)")
	recent := fs.Bool("recent", false, "Shorthand for --since 24h")
	runningOnly := fs.Bool("running", false, "Shorthand for --phase RUNNING")
	failedOnly := fs.Bool("failed", false, "Shorthand for --phase FAILED")
	pendingOnly := fs.Bool("pending", false, "Shorthand for --phase PENDING")
	waitingOnly := fs.Bool("waiting", false, "Shorthand for --phase WAITING")
	activeOnly := fs.Bool("active", false, "Show only active runs (RUNNING + PENDING + WAITING)")
	doneOnly := fs.Bool("done", false, "Shorthand for --phase DONE (successful runs)")
	cancelledOnly := fs.Bool("cancelled", false, "Shorthand for --phase CANCELLED")
	noHeader := fs.Bool("no-header", false, "Omit the column header row (useful for scripting)")
	titleWidth := fs.Int("title-width", 32, "Max characters to show in the title column (min: 10)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs list [flags]\n\nList recent agent runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if *recent && *since == "" {
		*since = "24h"
	}
	phaseShorthands := 0
	for _, b := range []*bool{runningOnly, failedOnly, pendingOnly, waitingOnly, doneOnly, cancelledOnly} {
		if *b {
			phaseShorthands++
		}
	}
	if phaseShorthands > 1 {
		return fmt.Errorf("--running, --failed, --pending, --waiting, --done, and --cancelled are mutually exclusive")
	}
	if *runningOnly && *phase == "" {
		*phase = "RUNNING"
	}
	if *failedOnly && *phase == "" {
		*phase = "FAILED"
	}
	if *pendingOnly && *phase == "" {
		*phase = "PENDING"
	}
	if *waitingOnly && *phase == "" {
		*phase = "WAITING"
	}
	if *doneOnly && *phase == "" {
		*phase = "DONE"
	}
	if *cancelledOnly && *phase == "" {
		*phase = "CANCELLED"
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
		if (!*all && !*activeOnly) || nextCursor == "" {
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
	if *titleContains != "" {
		needle := strings.ToLower(*titleContains)
		filtered := runs[:0]
		for _, r := range runs {
			title := strings.ToLower(r.GetSpec().GetDisplayName())
			if strings.Contains(title, needle) {
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}
	if *activeOnly {
		filtered := runs[:0]
		for _, r := range runs {
			switch r.GetStatus().GetPhase() {
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}
	if *sortBy != "" {
		switch strings.ToLower(*sortBy) {
		case "started":
			sort.Slice(runs, func(i, j int) bool {
				ti := runs[i].GetStatus().GetStartedAt()
				tj := runs[j].GetStatus().GetStartedAt()
				if ti == nil {
					return false
				}
				if tj == nil {
					return true
				}
				return ti.AsTime().After(tj.AsTime())
			})
		case "phase":
			sort.Slice(runs, func(i, j int) bool {
				return runs[i].GetStatus().GetPhase() < runs[j].GetStatus().GetPhase()
			})
		default:
			return fmt.Errorf("--sort %q: must be started or phase", *sortBy)
		}
	}
	if len(runs) == 0 && !*jsonOut && !*idsOnly {
		fmt.Println("No runs found.")
		return nil
	}

	if *idsOnly {
		for _, r := range runs {
			fmt.Println(r.GetId())
		}
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
	if !*noHeader {
		if *verbose {
			fmt.Fprintln(w, "ID\tTITLE\tPHASE\tDURATION\tMODEL\tSTARTED\tREPO\tPROJECT")
		} else {
			fmt.Fprintln(w, "ID\tTITLE\tPHASE\tDURATION\tMODEL\tSTARTED")
		}
	}
	for _, r := range runs {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if title == "" {
			title = "-"
		}
		maxTitle := *titleWidth
		if maxTitle < 10 {
			maxTitle = 10
		}
		if len(title) > maxTitle {
			title = title[:maxTitle-3] + "..."
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
			t := r.GetStatus().GetStartedAt().AsTime()
			if *relative {
				started = relativeTime(t)
			} else {
				started = t.Format(time.RFC3339)
			}
		} else if r.GetCreatedAt() != nil {
			t := r.GetCreatedAt().AsTime()
			if *relative {
				started = relativeTime(t)
			} else {
				started = t.Format(time.RFC3339)
			}
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

	// Build phase summary for footer.
	phaseSummary := func() string {
		phaseCounts := map[string]int{}
		for _, r := range runs {
			phaseCounts[phaseLabel(r.GetStatus().GetPhase())]++
		}
		var parts []string
		for _, ph := range []string{"RUNNING", "PENDING", "WAITING", "FAILED", "DONE", "CANCELLED"} {
			if n := phaseCounts[ph]; n > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", n, strings.ToLower(ph)))
			}
		}
		if len(parts) == 0 {
			return ""
		}
		return " (" + strings.Join(parts, ", ") + ")"
	}

	if nextCursor != "" && !*all {
		fmt.Printf("next-cursor: %s\n", nextCursor)
		fmt.Printf("Showing %d run(s) — use --all or --limit to see more\n", len(runs))
	} else if *all {
		fmt.Printf("Showing all %d run(s)%s\n", len(runs), phaseSummary())
	} else if len(runs) > 0 {
		isFiltered := !sinceTime.IsZero() || *repoURL != "" || *titleContains != "" ||
			*activeOnly || *runningOnly || *failedOnly || *pendingOnly || *waitingOnly || *doneOnly || *cancelledOnly ||
			*project != "" || *feature != "" || *tag != "" || *phase != ""
		suffix := ""
		if isFiltered {
			suffix = " (filtered)"
		}
		fmt.Printf("Showing %d run(s)%s%s\n", len(runs), suffix, phaseSummary())
	}

	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRunsGet(args []string) error {
	fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	showLog := fs.Bool("log", false, "Print the persisted agent log output")
	showLogs := fs.Bool("logs", false, "Alias for --log")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	short := fs.Bool("short", false, "Print a one-line summary: ID PHASE TITLE")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs get <id> [flags]\n\nShow full detail for an agent run.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *showLogs {
		*showLog = true
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	// Support multiple IDs: print each separated by a blank line.
	if fs.NArg() > 1 {
		var firstErr error
		for i, id := range fs.Args() {
			if i > 0 {
				fmt.Println()
			}
			subArgs := []string{id}
			if *server != "" {
				subArgs = append(subArgs, "--server="+*server)
			}
			if *showLog {
				subArgs = append(subArgs, "--log")
			}
			if *jsonOut {
				subArgs = append(subArgs, "--json")
			}
			if *noColor {
				subArgs = append(subArgs, "--no-color")
			}
			if *short {
				subArgs = append(subArgs, "--short")
			}
			if err := runRunsGet(subArgs); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
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
			ID          string            `json:"id"`
			Title       string            `json:"title,omitempty"`
			Phase       string            `json:"phase"`
			Message     string            `json:"message,omitempty"`
			Project     string            `json:"project,omitempty"`
			Feature     string            `json:"feature,omitempty"`
			Prompt      string            `json:"prompt,omitempty"`
			Repo        string            `json:"repo,omitempty"`
			Branch      string            `json:"branch,omitempty"`
			Model       string            `json:"model,omitempty"`
			Tags        []string          `json:"tags,omitempty"`
			EnvVars     map[string]string `json:"env_vars,omitempty"`
			ParentRunID string            `json:"parent_run_id,omitempty"`
			Children    []string          `json:"children,omitempty"`
			Started     string            `json:"started,omitempty"`
			Completed   string            `json:"completed,omitempty"`
			Duration    string            `json:"duration,omitempty"`
			PrURL       string            `json:"pr_url,omitempty"`
		}
		out := runGetJSON{
			ID:       r.GetId(),
			Title:    r.GetSpec().GetDisplayName(),
			Phase:    phaseLabel(r.GetStatus().GetPhase()),
			Message:  r.GetStatus().GetMessage(),
			Project:  r.GetSpec().GetProject(),
			Feature:  r.GetSpec().GetFeature(),
			Prompt:   r.GetSpec().GetPrompt(),
			Model:    r.GetSpec().GetModelTier(),
			Tags:     r.GetSpec().GetTags(),
			EnvVars:  r.GetSpec().GetEnvVars(),
			Children: r.GetChildren(),
			PrURL:    r.GetStatus().GetPrUrl(),
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

	if *short {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		fmt.Printf("%s  %s  %s\n", r.GetId(), phaseLabel(r.GetStatus().GetPhase()), title)
		return nil
	}
	useColorGet := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	fmt.Printf("ID:       %s\n", r.GetId())
	if dn := r.GetSpec().GetDisplayName(); dn != "" {
		fmt.Printf("Title:    %s\n", dn)
	}
	pl := phaseLabel(r.GetStatus().GetPhase())
	if useColorGet {
		switch r.GetStatus().GetPhase() {
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING:
			pl = "\033[32m" + pl + "\033[0m"
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING:
			pl = "\033[33m" + pl + "\033[0m"
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
			pl = "\033[36m" + pl + "\033[0m"
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
			pl = "\033[31m" + pl + "\033[0m"
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
			pl = "\033[90m" + pl + "\033[0m"
		case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
			pl = "\033[35m" + pl + "\033[0m"
		}
	}
	fmt.Printf("Phase:    %s\n", pl)
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
	if envVars := r.GetSpec().GetEnvVars(); len(envVars) > 0 {
		keys := make([]string, 0, len(envVars))
		for k := range envVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("Env:      %s=%s\n", k, envVars[k])
		}
	}
	if r.GetStatus().GetStartedAt() != nil {
		t := r.GetStatus().GetStartedAt().AsTime()
		ago := time.Since(t).Round(time.Second)
		fmt.Printf("Started:  %s (%s ago)\n", t.Format(time.RFC3339), ago)
	}
	if r.GetStatus().GetCompletedAt() != nil {
		t := r.GetStatus().GetCompletedAt().AsTime()
		fmt.Printf("Completed: %s\n", t.Format(time.RFC3339))
		if r.GetStatus().GetStartedAt() != nil {
			dur := t.Sub(r.GetStatus().GetStartedAt().AsTime())
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
	if children := r.GetChildren(); len(children) > 0 {
		fmt.Printf("Children: %s\n", strings.Join(children, ", "))
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
	fmt.Printf("Run:      %s\n", r.GetId())
	fmt.Printf("Phase:    %s\n", phaseLabel(r.GetStatus().GetPhase()))
	if msg := r.GetStatus().GetMessage(); msg != "" {
		fmt.Printf("Message:  %s\n", msg)
	}
	if r.GetStatus().GetStartedAt() != nil && r.GetStatus().GetCompletedAt() != nil {
		dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime()).Round(time.Second)
		fmt.Printf("Duration: %s\n", dur)
	}
	if tier := r.GetSpec().GetModelTier(); tier != "" {
		fmt.Printf("Model:    %s\n", tier)
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
	lines := fs.Int("lines", 0, "Show only the last N lines of output (0 = all)")
	timestamps := fs.Bool("timestamps", false, "Prefix each line with a timestamp (--no-follow only)")
	grep := fs.String("grep", "", "Only show lines matching this substring (case-insensitive; works in both streaming and --no-follow mode)")
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
		allLines := strings.Split(logOutput, "\n")
		if *grep != "" {
			needle := strings.ToLower(*grep)
			filtered := allLines[:0]
			for _, l := range allLines {
				if strings.Contains(strings.ToLower(l), needle) {
					filtered = append(filtered, l)
				}
			}
			allLines = filtered
		}
		if *lines > 0 && len(allLines) > *lines {
			allLines = allLines[len(allLines)-*lines:]
		}
		if *timestamps {
			now := time.Now().Format("15:04:05")
			for _, line := range allLines {
				if line != "" {
					fmt.Printf("[%s] %s\n", now, line)
				}
			}
		} else {
			fmt.Print(strings.Join(allLines, "\n"))
		}
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

	grepNeedle := ""
	if *grep != "" {
		grepNeedle = strings.ToLower(*grep)
	}

	var finalPhase apiv1.AgentRunPhase
	for stream.Receive() {
		ev := stream.Msg()
		switch ev.GetType() {
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
			payload := ev.GetPayload()
			if payload != "" {
				if grepNeedle != "" {
					// Filter line by line.
					for _, line := range strings.Split(payload, "\n") {
						if strings.Contains(strings.ToLower(line), grepNeedle) {
							fmt.Println(line)
						}
					}
				} else {
					fmt.Print(payload)
				}
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
		fmt.Fprintf(fs.Output(), "Usage: uncworks runs %s <id> [<id> ...] [flags]\n\nFlags:\n", verb)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	body, _ := json.Marshal(map[string]bool{"archived": archived})
	baseURL := serverBaseURL(*server)
	var errs []string
	for _, id := range fs.Args() {
		url := baseURL + "/api/v1/runs/" + id + "/archive"
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: build request: %v", id, err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: request failed: %v", id, err))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			_ = resp.Body.Close()
			errs = append(errs, fmt.Sprintf("%s: server returned %d: %s", id, resp.StatusCode, string(b)))
			continue
		}
		_ = resp.Body.Close()
		if archived {
			fmt.Printf("Run %s archived\n", id)
		} else {
			fmt.Printf("Run %s unarchived\n", id)
		}
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("%d operation(s) failed", len(errs))
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

func runRunsArchiveFailed(args []string) error {
	fs := flag.NewFlagSet("runs archive-failed", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	dryRun := fs.Bool("dry-run", false, "Print what would be archived without doing it")
	minAge := fs.Duration("min-age", 0, "Only archive runs completed longer ago than this (e.g. 24h)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs archive-failed [flags]\n\nBulk archive all FAILED runs.\n\nFlags:")
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

	var failedRuns []string
	var cursor string
	for {
		req := connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			PhaseFilter:   apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED,
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
			failedRuns = append(failedRuns, r.GetId())
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if len(failedRuns) == 0 {
		fmt.Println("No FAILED runs to archive.")
		return nil
	}

	if *dryRun {
		fmt.Printf("Would archive %d run(s):\n", len(failedRuns))
		for _, id := range failedRuns {
			fmt.Printf("  %s\n", id)
		}
		return nil
	}

	archived := 0
	archiveBody, _ := json.Marshal(map[string]bool{"archived": true})
	for _, id := range failedRuns {
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
	fmt.Printf("Archived %d/%d run(s).\n", archived, len(failedRuns))
	return nil
}

func runRunsPrune(args []string) error {
	fs := flag.NewFlagSet("runs prune", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	olderThan := fs.Duration("older-than", 7*24*time.Hour, "Archive runs completed longer ago than this (e.g. 24h, 7d)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	dryRun := fs.Bool("dry-run", false, "Print what would be archived without doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	failedOnly := fs.Bool("failed", false, "Only prune FAILED runs (exclude DONE and CANCELLED)")
	doneOnly := fs.Bool("done", false, "Only prune DONE/SUCCEEDED runs (exclude FAILED and CANCELLED)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs prune [flags]\n\nBulk archive all terminal (DONE, FAILED, CANCELLED) runs older than the given age.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *failedOnly && *doneOnly {
		return fmt.Errorf("--failed and --done are mutually exclusive")
	}

	threshold := time.Now().Add(-*olderThan)
	client, err := newClient(*server)
	if err != nil {
		return err
	}

	terminalPhases := []apiv1.AgentRunPhase{
		apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
		apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED,
		apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED,
	}
	if *failedOnly {
		terminalPhases = []apiv1.AgentRunPhase{apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED}
	} else if *doneOnly {
		terminalPhases = []apiv1.AgentRunPhase{apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED}
	}

	var toArchive []string
	for _, phase := range terminalPhases {
		cursor := ""
		for {
			req := connect.NewRequest(&apiv1.ListAgentRunsRequest{
				Limit:         100,
				PhaseFilter:   phase,
				ProjectFilter: *project,
				FeatureFilter: *feature,
				Cursor:        cursor,
			})
			resp, err := client.ListAgentRuns(context.Background(), req)
			if err != nil {
				return fmt.Errorf("%s", humanizeErr(err))
			}
			for _, r := range resp.Msg.GetAgentRuns() {
				completedAt := r.GetStatus().GetCompletedAt()
				if completedAt != nil && completedAt.AsTime().Before(threshold) {
					toArchive = append(toArchive, r.GetId())
				}
			}
			cursor = resp.Msg.GetNextCursor()
			if cursor == "" {
				break
			}
		}
	}

	if len(toArchive) == 0 {
		fmt.Printf("No terminal runs older than %s to archive.\n", *olderThan)
		return nil
	}

	if *dryRun {
		fmt.Printf("Would archive %d run(s) older than %s:\n", len(toArchive), *olderThan)
		for _, id := range toArchive {
			fmt.Printf("  %s\n", id)
		}
		return nil
	}

	if !*yes {
		fmt.Printf("Archive %d terminal run(s) older than %s? [y/N] ", len(toArchive), *olderThan)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	archived := 0
	archiveBody, _ := json.Marshal(map[string]bool{"archived": true})
	for _, id := range toArchive {
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
	fmt.Printf("Archived %d/%d run(s).\n", archived, len(toArchive))
	return nil
}

// ── stats ─────────────────────────────────────────────────────────────────────

func runRunsStats(args []string) error {
	fs := flag.NewFlagSet("runs stats", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	format := fs.String("format", "table", "Output format (table|json)")
	limit := fs.Int("limit", 0, "Count only the N most recent runs (0 = all)")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	reasonLen := fs.Int("reason-length", 120, "Max length of failure reason messages (0 = unlimited)")
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
	failureReasons := map[string]int{}
	for {
		pageSize := int32(100)
		if *limit > 0 && *limit-total < 100 {
			pageSize = int32(*limit - total)
		}
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         pageSize,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
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
			if label == "FAILED" {
				msg := r.GetStatus().GetMessage()
				if msg == "" {
					msg = "(no message)"
				}
				failureReasons[msg]++
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
		type reasonJSON struct {
			Reason string `json:"reason"`
			Count  int    `json:"count"`
		}
		var topReasons []reasonJSON
		if len(failureReasons) > 0 {
			type rc struct {
				reason string
				count  int
			}
			var rcs []rc
			for r, c := range failureReasons {
				rcs = append(rcs, rc{r, c})
			}
			sort.Slice(rcs, func(i, j int) bool {
				if rcs[i].count != rcs[j].count {
					return rcs[i].count > rcs[j].count
				}
				return rcs[i].reason < rcs[j].reason
			})
			if len(rcs) > 5 {
				rcs = rcs[:5]
			}
			for _, r := range rcs {
				reason := r.reason
				if *reasonLen > 0 && len(reason) > *reasonLen {
					reason = reason[:*reasonLen] + "..."
				}
				topReasons = append(topReasons, reasonJSON{Reason: reason, Count: r.count})
			}
		}
		out := map[string]interface{}{
			"total":               total,
			"phases":              counts,
			"success_rate":        successRate,
			"top_failure_reasons": topReasons,
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
	maxCount := 0
	for _, phase := range order {
		if counts[phase] > maxCount {
			maxCount = counts[phase]
		}
	}
	var statsBuf bytes.Buffer
	w := tabwriter.NewWriter(&statsBuf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PHASE\tCOUNT\tPCT\tBAR")
	for _, phase := range order {
		pct := 0.0
		if total > 0 {
			pct = float64(counts[phase]) / float64(total) * 100
		}
		barLen := 0
		if maxCount > 0 {
			barLen = int(float64(counts[phase]) / float64(maxCount) * 20)
		}
		bar := strings.Repeat("█", barLen)
		fmt.Fprintf(w, "%s\t%d\t%.1f%%\t%s\n", phase, counts[phase], pct, bar)
	}
	_ = w.Flush()
	statsOutput := statsBuf.String()
	if term.IsTerminal(int(os.Stdout.Fd())) {
		statsOutput = strings.NewReplacer(
			"RUNNING  ", "\033[32mRUNNING\033[0m  ",
			"PENDING  ", "\033[33mPENDING\033[0m  ",
			"WAITING  ", "\033[36mWAITING\033[0m  ",
			"FAILED   ", "\033[31mFAILED\033[0m   ",
			"DONE     ", "\033[90mDONE\033[0m     ",
			"CANCELLED", "\033[35mCANCELLED\033[0m",
		).Replace(statsOutput)
	}
	fmt.Print(statsOutput)

	if done+failed > 0 {
		rate := float64(done) / float64(done+failed) * 100
		fmt.Printf("\nSuccess rate: %.1f%% (%d/%d completed runs)\n", rate, done, done+failed)
	}
	if medianDuration >= 0 {
		fmt.Printf("Median duration: %s\n", medianDuration.Round(time.Second))
	} else if done > 0 {
		fmt.Printf("Median duration: —\n")
	}
	if len(failureReasons) > 0 {
		type reasonCount struct {
			reason string
			count  int
		}
		var reasons []reasonCount
		for r, c := range failureReasons {
			reasons = append(reasons, reasonCount{r, c})
		}
		sort.Slice(reasons, func(i, j int) bool {
			if reasons[i].count != reasons[j].count {
				return reasons[i].count > reasons[j].count
			}
			return reasons[i].reason < reasons[j].reason
		})
		if len(reasons) > 5 {
			reasons = reasons[:5]
		}
		fmt.Printf("\nTop failure reasons:\n")
		for i, rc := range reasons {
			reason := rc.reason
			if *reasonLen > 0 && len(reason) > *reasonLen {
				reason = reason[:*reasonLen] + "..."
			}
			fmt.Printf("  %d. %s (%d run", i+1, reason, rc.count)
			if rc.count != 1 {
				fmt.Print("s")
			}
			fmt.Println(")")
		}
	}
	return nil
}

// ── open ────────────────────────────────────────────────────────────────────────

func runRunsOpen(args []string) error {
	fs := flag.NewFlagSet("runs open", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	printURL := fs.Bool("print-url", false, "Print the PR URL instead of opening the browser")
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

	if *printURL {
		fmt.Println(prURL)
		return nil
	}

	fmt.Printf("Opening PR: %s\n", prURL)
	if err := openBrowser(prURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// ── inspect ──────────────────────────────────────────────────────────────────

func runRunsInspect(args []string) error {
	fs := flag.NewFlagSet("runs inspect", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	logLines := fs.Int("log-lines", 20, "Number of log tail lines to show (0 = all)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs inspect <id> [flags]\n\nDiagnostic view for a run: full details, graph, and log tail.\n\nFlags:")
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

	resp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	r := resp.Msg
	phase := r.GetStatus().GetPhase()

	// Print full detail.
	getArgs := []string{id}
	if *server != "" {
		getArgs = append(getArgs, "--server="+*server)
	}
	_ = runRunsGet(getArgs)

	// Show execution graph if there are children or for running runs.
	children := r.GetChildren()
	if len(children) > 0 || phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING {
		fmt.Println("\n─── execution graph ────────────────────────────────────────────────────────")
		graphResp, graphErr := client.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{Id: id}))
		if graphErr == nil {
			printGraph(id, graphResp.Msg, term.IsTerminal(int(os.Stdout.Fd())))
		}
	}

	// Show log tail for non-pending runs.
	logOutput := r.GetStatus().GetLogOutput()
	if logOutput != "" {
		fmt.Println("\n─── log tail ───────────────────────────────────────────────────────────────")
		lines := strings.Split(logOutput, "\n")
		if *logLines > 0 && len(lines) > *logLines {
			lines = lines[len(lines)-*logLines:]
			fmt.Printf("(showing last %d lines)\n", *logLines)
		}
		fmt.Println(strings.Join(lines, "\n"))
	} else if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED {
		fmt.Println("\n─── no stored log output ───────────────────────────────────────────────────")
		fmt.Println("Run 'uncworks runs logs " + id + " --no-follow' for stored output.")
	}

	return nil
}

// ── diff ──────────────────────────────────────────────────────────────────────

func runRunsDiff(args []string) error {
	fs := flag.NewFlagSet("runs diff", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	stat := fs.Bool("stat", false, "Show git diff --stat instead of full diff")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs diff <id> [flags]\n\nShow the git commands to inspect the diff for a completed run.\n\nFlags:")
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

	repos := r.GetSpec().GetRepos()
	if len(repos) == 0 {
		return fmt.Errorf("run %s has no repository configured", id)
	}
	repoURL := repos[0].GetUrl()
	baseBranch := repos[0].GetBranch()
	if baseBranch == "" {
		baseBranch = "main"
	}

	prURL := r.GetStatus().GetPrUrl()
	if prURL == "" && !r.GetSpec().GetAutoPush() {
		return fmt.Errorf("run %s has no PR and --auto-push was not set", id)
	}

	agentBranch := fmt.Sprintf("agent/%s", id)
	if prURL != "" {
		// GitHub PR URL: extract branch from link (not always available directly)
		// Show instructions using the PR URL
		fmt.Printf("Run:       %s\n", id)
		fmt.Printf("Repo:      %s\n", repoURL)
		fmt.Printf("Base:      %s\n", baseBranch)
		fmt.Printf("PR:        %s\n", prURL)
		fmt.Println()
		diffFlag := ""
		if *stat {
			diffFlag = " --stat"
		}
		fmt.Printf("To view the diff:\n")
		fmt.Printf("  git fetch origin %s\n", agentBranch)
		fmt.Printf("  git diff%s origin/%s...origin/%s\n", diffFlag, baseBranch, agentBranch)
	} else {
		fmt.Printf("Run:       %s\n", id)
		fmt.Printf("Repo:      %s\n", repoURL)
		fmt.Printf("Branch:    %s\n", agentBranch)
		fmt.Println()
		diffFlag := ""
		if *stat {
			diffFlag = " --stat"
		}
		fmt.Printf("To view the diff:\n")
		fmt.Printf("  git fetch origin %s\n", agentBranch)
		fmt.Printf("  git diff%s origin/%s...origin/%s\n", diffFlag, baseBranch, agentBranch)
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
	autoPush := fs.Bool("auto-push", false, "Push changes to a feature branch after the run succeeds")
	autoPR := fs.Bool("auto-pr", false, "Create a GitHub PR after the run succeeds (implies --auto-push)")
	outputID := fs.Bool("output-id", false, "Print only the new run ID (for scripting)")
	wait := fs.Bool("wait", false, "Wait for the retried run to complete; exit 0 on success, 1 on failure")
	follow := fs.Bool("follow", false, "Stream logs after submitting (takes precedence over --wait)")
	var envFlags multiFlag
	fs.Var(&envFlags, "env", "Override environment variables (repeatable, KEY=VALUE); replaces all env vars if any are provided")
	var tagFlags multiFlag
	fs.Var(&tagFlags, "tag", "Override tags (repeatable); replaces all tags if any are provided")
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
		Backend:     spec.Backend,
		Repos:       spec.Repos,
		Prompt:      spec.Prompt,
		Project:     spec.Project,
		Feature:     spec.Feature,
		ModelTier:   spec.ModelTier,
		Tags:        spec.Tags,
		AutoPush:    spec.AutoPush,
		AutoPr:      spec.AutoPr,
		ParentRunId: spec.GetParentRunId(),
		EnvVars:     spec.GetEnvVars(),
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
	if *autoPush || *autoPR {
		newSpec.AutoPush = *autoPush || *autoPR
		newSpec.AutoPr = *autoPR
	}
	if len(tagFlags) > 0 {
		newSpec.Tags = []string(tagFlags)
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
		if !*follow && !*wait {
			fmt.Printf("Follow progress: uncworks runs tail %s\n", newRun.GetId())
		}
	}
	if *follow {
		return runRunsTail([]string{newRun.GetId(), "--server=" + *server})
	}
	if *wait {
		return runRunsTail([]string{newRun.GetId(), "--server=" + *server})
	}
	return nil
}

// ── retry-failed ─────────────────────────────────────────────────────────────

func runRunsRetryFailed(args []string) error {
	fs := flag.NewFlagSet("runs retry-failed", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Only retry runs in this project")
	feature := fs.String("feature", "", "Only retry runs for this feature")
	tag := fs.String("tag", "", "Only retry runs with this tag")
	since := fs.String("since", "", "Only retry runs created within this window (e.g. 1h, 24h, 7d)")
	limit := fs.Int("limit", 0, "Retry at most N runs (0 = no limit)")
	dryRun := fs.Bool("dry-run", false, "Print what would be retried without actually doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	modelTier := fs.String("model-tier", "", "Override model tier for all retried runs")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs retry-failed [flags]\n\nBulk retry all FAILED runs matching the given filters.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
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

	// Fetch FAILED runs.
	var failedRuns []*apiv1.AgentRun
	cursor := ""
	for {
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			PhaseFilter:   apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}
		resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					continue
				}
			}
			failedRuns = append(failedRuns, r)
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if *limit > 0 && len(failedRuns) > *limit {
		failedRuns = failedRuns[:*limit]
	}

	if len(failedRuns) == 0 {
		fmt.Println("No failed runs found matching the given filters.")
		return nil
	}

	fmt.Printf("Found %d failed run(s) to retry:\n", len(failedRuns))
	for _, r := range failedRuns {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Printf("  %s  %s\n", r.GetId(), title)
	}

	if *dryRun {
		fmt.Printf("\nDry run: no runs created.\n")
		return nil
	}

	if !*yes {
		fmt.Printf("\nRetry %d run(s)? [y/N] ", len(failedRuns))
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println()
	for _, r := range failedRuns {
		spec := r.GetSpec()
		newSpec := &apiv1.AgentRunSpec{
			Backend:     spec.Backend,
			Repos:       spec.Repos,
			Prompt:      spec.Prompt,
			Project:     spec.Project,
			Feature:     spec.Feature,
			ModelTier:   spec.ModelTier,
			Tags:        spec.Tags,
			AutoPush:    spec.AutoPush,
			AutoPr:      spec.AutoPr,
			ParentRunId: spec.GetParentRunId(),
			EnvVars:     spec.GetEnvVars(),
		}
		if *modelTier != "" {
			newSpec.ModelTier = *modelTier
		}
		createResp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{Spec: newSpec}))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s → error: %s\n", r.GetId(), humanizeErr(err))
			continue
		}
		fmt.Printf("  %s → %s\n", r.GetId(), createResp.Msg.GetAgentRun().GetId())
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
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	n := fs.Int("n", 1, "Number of latest runs to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	showLog := fs.Bool("log", false, "Also print the stored agent log output")
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

	if *n < 1 {
		return fmt.Errorf("--n must be >= 1")
	}
	listReq := &apiv1.ListAgentRunsRequest{
		Limit:         int32(*n),
		ProjectFilter: *project,
		FeatureFilter: *feature,
		TagFilter:     *tag,
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

	for i, r := range runs {
		if i > 0 {
			fmt.Println()
		}
		getArgs := []string{r.GetId()}
		if *server != "" {
			getArgs = append(getArgs, "--server="+*server)
		}
		if *jsonOut {
			getArgs = append(getArgs, "--json")
		}
		if *showLog {
			getArgs = append(getArgs, "--log")
		}
		if err := runRunsGet(getArgs); err != nil {
			return err
		}
	}
	return nil
}

// ── export ────────────────────────────────────────────────────────────────────

func runRunsExport(args []string) error {
	fs := flag.NewFlagSet("runs export", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED)")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	outFile := fs.String("out", "", "Write output to file instead of stdout")
	format := fs.String("format", "csv", "Output format: csv, tsv, or json")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs export [flags]\n\nExport runs as CSV (stdout by default).\n\nFlags:")
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
		Limit:         100,
		ProjectFilter: *project,
		FeatureFilter: *feature,
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
		case "WAITING":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
		case "CANCELLED":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		default:
			return fmt.Errorf("invalid phase %q", *phase)
		}
	}

	var allRuns []*apiv1.AgentRun
	cursor := ""
	for {
		listReq.Cursor = cursor
		resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
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
			allRuns = append(allRuns, r)
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	var out *os.File
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			return fmt.Errorf("create %s: %w", *outFile, err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	if *format == "json" {
		type exportJSON struct {
			ID         string   `json:"id"`
			Title      string   `json:"title,omitempty"`
			Phase      string   `json:"phase"`
			Project    string   `json:"project,omitempty"`
			Feature    string   `json:"feature,omitempty"`
			Model      string   `json:"model,omitempty"`
			Started    string   `json:"started,omitempty"`
			Completed  string   `json:"completed,omitempty"`
			DurationS  float64  `json:"duration_s,omitempty"`
			PrURL      string   `json:"pr_url,omitempty"`
			Tags       []string `json:"tags,omitempty"`
		}
		var rows []exportJSON
		for _, r := range allRuns {
			row := exportJSON{
				ID:      r.GetId(),
				Title:   r.GetSpec().GetDisplayName(),
				Phase:   phaseLabel(r.GetStatus().GetPhase()),
				Project: r.GetSpec().GetProject(),
				Feature: r.GetSpec().GetFeature(),
				Model:   r.GetSpec().GetModelTier(),
				PrURL:   r.GetStatus().GetPrUrl(),
				Tags:    r.GetSpec().GetTags(),
			}
			if r.GetStatus().GetStartedAt() != nil {
				row.Started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
			}
			if r.GetStatus().GetCompletedAt() != nil {
				row.Completed = r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339)
				if r.GetStatus().GetStartedAt() != nil {
					row.DurationS = r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime()).Seconds()
				}
			}
			rows = append(rows, row)
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rows); err != nil {
			return fmt.Errorf("json encode: %w", err)
		}
		if *outFile != "" {
			fmt.Fprintf(os.Stderr, "Exported %d run(s) to %s\n", len(allRuns), *outFile)
		}
		return nil
	}

	w := csv.NewWriter(out)
	if *format == "tsv" {
		w.Comma = '\t'
	}
	_ = w.Write([]string{"id", "title", "phase", "project", "feature", "model", "started", "completed", "duration_s", "pr_url", "tags"})
	for _, r := range allRuns {
		started := ""
		completed := ""
		durationS := ""
		if r.GetStatus().GetStartedAt() != nil {
			started = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
		}
		if r.GetStatus().GetCompletedAt() != nil {
			completed = r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339)
			if r.GetStatus().GetStartedAt() != nil {
				dur := r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime())
				durationS = fmt.Sprintf("%.0f", dur.Seconds())
			}
		}
		_ = w.Write([]string{
			r.GetId(),
			r.GetSpec().GetDisplayName(),
			phaseLabel(r.GetStatus().GetPhase()),
			r.GetSpec().GetProject(),
			r.GetSpec().GetFeature(),
			r.GetSpec().GetModelTier(),
			started,
			completed,
			durationS,
			r.GetStatus().GetPrUrl(),
			strings.Join(r.GetSpec().GetTags(), ";"),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("csv write: %w", err)
	}
	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "Exported %d run(s) to %s\n", len(allRuns), *outFile)
	}
	return nil
}

// ── count ─────────────────────────────────────────────────────────────────────

func runRunsCount(args []string) error {
	fs := flag.NewFlagSet("runs count", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	phaseFilter := fs.String("phase", "", "Count only runs in this phase (RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	byPhase := fs.Bool("by-phase", false, "Show count breakdown by phase instead of total")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs count [flags]\n\nPrint the number of runs matching the given filters.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	var sinceTime time.Time
	if *since != "" {
		d, err := parseSinceDuration(*since)
		if err != nil {
			return fmt.Errorf("--since %q: %w", *since, err)
		}
		sinceTime = time.Now().Add(-d)
	}

	var wantPhase apiv1.AgentRunPhase
	phaseFiltered := false
	if *phaseFilter != "" {
		phaseFiltered = true
		switch strings.ToUpper(*phaseFilter) {
		case "RUNNING":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
		case "DONE":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
		case "FAILED":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
		case "PENDING":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
		case "WAITING":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
		case "CANCELLED":
			wantPhase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		default:
			return fmt.Errorf("invalid phase %q: must be RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED", *phaseFilter)
		}
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	count := 0
	phaseCounts := map[string]int{}
	cursor := ""
	for {
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}
		if phaseFiltered {
			listReq.PhaseFilter = wantPhase
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
			count++
			if *byPhase {
				phaseCounts[phaseLabel(r.GetStatus().GetPhase())]++
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if *byPhase {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PHASE\tCOUNT")
		for _, ph := range []string{"RUNNING", "PENDING", "WAITING", "DONE", "FAILED", "CANCELLED", "UNKNOWN"} {
			if n, ok := phaseCounts[ph]; ok {
				fmt.Fprintf(w, "%s\t%d\n", ph, n)
			}
		}
		w.Flush()
		fmt.Printf("Total: %d\n", count)
	} else {
		fmt.Println(count)
	}
	return nil
}

// ── summary ───────────────────────────────────────────────────────────────────

func runRunsSummary(args []string) error {
	fs := flag.NewFlagSet("runs summary", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	since := fs.String("since", "24h", "Time window for summary (e.g. 1h, 24h, 7d)")
	project := fs.String("project", "", "Filter by project name")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs summary [flags]\n\nShow a dashboard summary of recent run activity.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	d, err := parseSinceDuration(*since)
	if err != nil {
		return fmt.Errorf("--since %q: %w", *since, err)
	}
	sinceTime := time.Now().Add(-d)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	phaseCounts := map[string]int{}
	var activeRuns []*apiv1.AgentRun
	var latestRun *apiv1.AgentRun
	total := 0
	cursor := ""
	for {
		listReq := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			Cursor:        cursor,
		}
		resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil || !ts.AsTime().After(sinceTime) {
				continue
			}
			total++
			label := phaseLabel(r.GetStatus().GetPhase())
			phaseCounts[label]++
			switch r.GetStatus().GetPhase() {
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
				activeRuns = append(activeRuns, r)
			}
			if latestRun == nil {
				latestRun = r
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	useColor := term.IsTerminal(int(os.Stdout.Fd()))
	colorPhase := func(label string) string {
		if !useColor {
			return label
		}
		switch label {
		case "RUNNING":
			return "\033[32m" + label + "\033[0m"
		case "PENDING":
			return "\033[33m" + label + "\033[0m"
		case "WAITING":
			return "\033[36m" + label + "\033[0m"
		case "FAILED":
			return "\033[31m" + label + "\033[0m"
		case "DONE":
			return "\033[90m" + label + "\033[0m"
		case "CANCELLED":
			return "\033[35m" + label + "\033[0m"
		}
		return label
	}

	fmt.Printf("Runs in the last %s", *since)
	if *project != "" {
		fmt.Printf(" (project: %s)", *project)
	}
	fmt.Printf(":\n\n")

	if total == 0 {
		fmt.Println("  No runs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, ph := range []string{"RUNNING", "PENDING", "WAITING", "DONE", "FAILED", "CANCELLED"} {
		if n := phaseCounts[ph]; n > 0 {
			pct := n * 100 / total
			bar := strings.Repeat("█", pct/5)
			fmt.Fprintf(w, "  %s\t%d\t(%d%%)\t%s\n", colorPhase(ph), n, pct, bar)
		}
	}
	fmt.Fprintf(w, "  ─────────────\t\t\t\n")
	fmt.Fprintf(w, "  TOTAL\t%d\t\t\n", total)
	w.Flush()

	if len(activeRuns) > 0 {
		fmt.Printf("\nActive runs (%d):\n", len(activeRuns))
		for _, r := range activeRuns {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			age := ""
			if ts := r.GetCreatedAt(); ts != nil {
				age = "  " + relativeTime(ts.AsTime())
			}
			fmt.Printf("  %s  %-40s  %s%s\n", r.GetId(), title, colorPhase(phaseLabel(r.GetStatus().GetPhase())), age)
		}
	}

	if latestRun != nil {
		title := latestRun.GetSpec().GetDisplayName()
		if title == "" {
			title = latestRun.GetSpec().GetProject()
		}
		age := ""
		if ts := latestRun.GetCreatedAt(); ts != nil {
			age = " (" + relativeTime(ts.AsTime()) + ")"
		}
		fmt.Printf("\nMost recent: %s  %s%s  %s\n",
			latestRun.GetId(), title, age,
			colorPhase(phaseLabel(latestRun.GetStatus().GetPhase())))
	}

	return nil
}

// parseSinceDuration parses a human duration like "1h", "24h", "7d".
// Standard time.ParseDuration handles h/m/s; "d" is handled manually.
// ── wait ──────────────────────────────────────────────────────────────────────

func runRunsWait(args []string) error {
	fs := flag.NewFlagSet("runs wait", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	timeout := fs.Duration("timeout", 0, "Max time to wait (e.g. 10m, 1h); 0 = no limit")
	quiet := fs.Bool("quiet", false, "Suppress all output; use exit code only")
	log := fs.Bool("log", false, "Stream log lines while waiting (like logs --follow)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs wait <id> [flags]\n\nBlock until the run reaches a terminal phase.\nExits 0 on success, 1 on failure or cancellation.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
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

	ctx := context.Background()
	var cancel context.CancelFunc
	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req := connect.NewRequest(&apiv1.WatchAgentRunRequest{Id: id})
	stream, err := client.WatchAgentRun(ctx, req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	var finalPayload string
	var finalType apiv1.AgentRunEventType
	for stream.Receive() {
		ev := stream.Msg()
		switch ev.GetType() {
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
			if *log && ev.GetPayload() != "" {
				fmt.Print(ev.GetPayload())
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
			if !*quiet {
				fmt.Printf("[%s] phase: %s\n", id, ev.GetPayload())
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_WAITING_FOR_INPUT:
			if !*quiet {
				fmt.Printf("[%s] waiting for input: %s\n", id, ev.GetPayload())
				fmt.Printf("Use 'uncworks input %s <text>' to respond.\n", id)
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
			finalPayload = ev.GetPayload()
			finalType = ev.GetType()
		}
	}
	if err := stream.Err(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("timed out after %s waiting for run %s", *timeout, id)
		}
		return fmt.Errorf("stream error: %s", humanizeErr(err))
	}
	_ = finalType

	// Get final status to determine exit code.
	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}
	phase := getResp.Msg.GetStatus().GetPhase()
	msg := getResp.Msg.GetStatus().GetMessage()
	if finalPayload == "" {
		finalPayload = msg
	}

	switch phase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		if !*quiet {
			fmt.Printf("[%s] done\n", id)
			if url := getResp.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
		}
		return nil
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		if finalPayload != "" {
			return fmt.Errorf("run %s failed: %s", id, finalPayload)
		}
		return fmt.Errorf("run %s failed", id)
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		return fmt.Errorf("run %s was cancelled", id)
	default:
		return fmt.Errorf("run %s ended in unexpected phase: %s", id, phaseLabel(phase))
	}
}

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
