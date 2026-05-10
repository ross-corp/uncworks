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
	"os/exec"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

const runsUsage = `Usage: uncworks runs <subcommand> [flags]

Subcommands:
  list              List recent agent runs (--json, --active, --phase, --project, --since, --show-tags, --show-diff, --model)
  get <id>          Show full detail for a run (--json, --short, --wait, --poll N, --prompt-only, multiple IDs)
  show <id>         Alias for get
  describe <id>     Show full detail including persisted log output
  logs <id>         Stream log output until the run completes (--no-follow, --grep, --lines, --last)
  tail <id>         Stream logs and show summary when run completes (--last)
  watch             Auto-refresh the runs list (--interval, --active, --phase, --project)
  wait <id>         Block until a run reaches a terminal phase (--last, --timeout, --log, --on-success)
  cost              Show cost and diff summary across runs (--since, --project, --model)
  velocity          Show runs-per-hour chart for the past 24h (--buckets, --project)
  percentiles       Show p50/p95/p99 duration for DONE runs (--since, --project)
  anomalies         Show DONE runs with unusually long duration (above 2x p75) (--since, --project)
  stats             Show aggregate counts of runs by phase (--format, --since, --by-project, --model)
  count             Print a count of runs (--by-phase, --by-feature, --by-tag, --project, --since, --model)
  score             Show success rate across 1h/24h/7d/30d windows (--project, --feature, --json)
  rate              Alias for score
  tally             Show daily run counts for the past N days (--days, --project, --include-archived, --json)
  summary           Show a dashboard summary of recent run activity
  latest            Show the most recent N runs (--n, --phase, --project, --tag, --ids-only)
  graph <id>        Show the run graph (parent/child relationships) (--watch for live refresh)
  inspect <id>      Diagnostic view: details, graph, and log tail (--last, --log-lines)
  diff <id>         Fetch and show git diff for a run's branch; auto-executes on TTY (--last, --stat, --print-cmd)
  commits <id>      Show git log (commits made by the agent) for a run's branch; auto-executes on TTY (--last, --oneline)
  log <id>          Alias for commits
  verify <id>       Show the verification result for a spec-driven run (--json, --last)
  compare <a> <b>   Side-by-side comparison of two runs (--json)
  open <id>         Open the PR URL for a completed run in browser (--last, --print-url)
  open-pr <id>      Alias for open
  ui <id>           Open a run in the UNCWORKS web dashboard (--last, --print-url)
  env <id>          Show env vars for a run (--export for shell export statements)
  retry <id>        Re-run with same spec; override with --prompt, --branch, --model, --append-prompt, --diff (--last)
  rerun <id>        Alias for retry
  copy <id>         Alias for retry
  retry-last        Retry the most recent run (alias for retry --last)
  tail-last         Tail the most recent run (alias for tail --last)
  retry-failed      Bulk retry all FAILED runs matching filters (--project, --since, --dry-run, --list)
  cancel <id>       Request cancellation of a running agent (multiple IDs supported)
  kill <id>         Alias for cancel
  cancel-all        Cancel all active runs (--project, --tag, --title-contains)
  archive <id>      Mark a run as archived
  unarchive <id>    Remove the archived flag from a run
  archive-done      Bulk archive all SUCCEEDED runs
  archive-failed    Bulk archive all FAILED runs
  prune             Bulk archive terminal runs older than a given age (--older-than, --failed, --done)
  export            Export runs as CSV, JSON, TSV, or markdown (--format, --since, --project, --out)
  multi-tail        Tail logs from multiple runs simultaneously (--active for auto-discover)
  top               Live view of active runs sorted by elapsed time (--feature, --title-contains)
  batch <file>      Submit multiple runs from a JSON file (--dry-run, --wait, --output-ids)
  histogram         Show a bar chart of run activity over a time window (--since, --buckets, --sparkline)
  group             Show runs organized into groups (--by project|feature|tag|model, --count-only, --json)
  search <term>     Search runs by prompt text, title, or project (--phase, --since, --project)
  timeline          Show a chronological view of completed runs (--since, --project)
  slow              Show slowest completed runs sorted by duration (--limit, --since, --project)
  alias             Show all available subcommand aliases
  prompt <id>       Print only the prompt of a run (alias for get --field prompt)
  id <id>           Print only the ID of a run (alias for get --field id)
  phase <id>        Print only the phase of a run (alias for get --field phase)
  status <id>       Alias for phase
  message <id>      Print only the status message (alias for get --field message)
  model <id>        Print only the model tier (alias for get --field model)
  branch <id>       Print only the branch (alias for get --field branch)
  pr-url <id>       Print only the PR URL (alias for get --field pr-url)
  duration <id>     Print only the run duration (alias for get --field duration)
  age <id>          Print only the run age (alias for get --field age)
  tags <id>         Print only the tags (alias for get --field tags)
  children <id>     List child runs of a parent run ID
  notify <id>       Wait for a run and send macOS notification when done (alias for wait --notify)

Shorthand subcommands:
  today             Runs from the last 24h (alias for list --since 24h --all)
  week              Runs from the last 7d (alias for list --since 7d --all)
  recent            Alias for today
  failed            Show FAILED runs (alias for list --failed)
  done              Show DONE runs (alias for list --done)
  succeeded         Alias for done
  pending           Show PENDING runs (alias for list --pending)
  running           Show RUNNING runs (alias for list --running)
  waiting           Show WAITING runs (alias for list --waiting)
  cancelled         Show CANCELLED runs (alias for list --cancelled)
  active            Show all active runs — RUNNING, PENDING, WAITING (alias for list --active --all)
  last-failed       Show the most recent FAILED run
  zero-commits      Show succeeded runs that made no code changes (alias for list --zero-commits --done --all)
  committed         Show runs that committed code changes (alias for list --has-diff --all)
  approvals         Show runs waiting for approval (alias for list --phase WAITING --approval-mode hitl --all)
  queue             Show all pending runs (alias for list --pending --all)
  by-project        Group by project (alias for group --by project)
  by-feature        Group by feature (alias for group --by feature)
  by-model          Group by model tier (alias for group --by model)
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
	case "get", "show":
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
	case "cost":
		return runRunsCost(rest)
	case "velocity":
		return runRunsVelocity(rest)
	case "percentiles", "pct":
		return runRunsPercentiles(rest)
	case "anomalies":
		return runRunsAnomalies(rest)
	case "stats":
		return runRunsStats(rest)
	case "open", "open-pr", "pr":
		return runRunsOpen(rest)
	case "retry", "rerun", "copy", "duplicate":
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
	case "verify":
		return runRunsVerify(rest)
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
	case "multi-tail", "multi-logs":
		return runRunsMultiTail(rest)
	case "top":
		return runRunsTop(rest)
	case "batch":
		return runRunsBatch(rest)
	case "histogram":
		return runRunsHistogram(rest)
	case "group":
		return runRunsGroup(rest)
	case "ui":
		return runRunsUI(rest)
	case "search":
		return runRunsSearch(rest)
	case "timeline":
		return runRunsTimeline(rest)
	case "compare":
		return runRunsCompare(rest)
	case "alias", "aliases":
		return runRunsAlias(rest)
	case "today":
		return runRunsList(append([]string{"--since", "24h", "--all"}, rest...))
	case "week":
		return runRunsList(append([]string{"--since", "7d", "--all"}, rest...))
	case "failed":
		return runRunsList(append([]string{"--failed"}, rest...))
	case "done", "succeeded":
		return runRunsList(append([]string{"--done"}, rest...))
	case "pending":
		return runRunsList(append([]string{"--pending"}, rest...))
	case "running":
		return runRunsList(append([]string{"--running"}, rest...))
	case "waiting":
		return runRunsList(append([]string{"--waiting"}, rest...))
	case "cancelled":
		return runRunsList(append([]string{"--cancelled"}, rest...))
	case "recent":
		return runRunsList(append([]string{"--since", "24h", "--all"}, rest...))
	case "by-project":
		return runRunsGroup(append([]string{"--by", "project"}, rest...))
	case "by-feature":
		return runRunsGroup(append([]string{"--by", "feature"}, rest...))
	case "by-model":
		return runRunsGroup(append([]string{"--by", "model"}, rest...))
	case "env":
		return runRunsEnv(rest)
	case "slow":
		return runRunsSlow(rest)
	case "score", "rate":
		return runRunsScore(rest)
	case "tally":
		return runRunsTally(rest)
	case "queue":
		return runRunsList(append([]string{"--pending", "--all"}, rest...))
	case "retry-last":
		return runRunsRetry(append([]string{"--last"}, rest...))
	case "tail-last":
		return runRunsTail(append([]string{"--last"}, rest...))
	case "notify":
		return runRunsWait(append([]string{"--notify"}, rest...))
	case "children":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "usage: uncworks runs children <parent-run-id>")
			return fmt.Errorf("parent run ID required")
		}
		return runRunsList(append([]string{"--parent-run-id", rest[0], "--all"}, rest[1:]...))
	case "commits", "log":
		return runRunsCommits(rest)
	case "question":
		return runRunsGetField(rest, "question")
	case "prompt":
		return runRunsGetField(rest, "prompt")
	case "id":
		return runRunsGetField(rest, "id")
	case "phase", "status":
		return runRunsGetField(rest, "phase")
	case "message", "msg":
		return runRunsGetField(rest, "message")
	case "pr-url", "pr-link":
		return runRunsGetField(rest, "pr-url")
	case "branch":
		return runRunsGetField(rest, "branch")
	case "model":
		return runRunsGetField(rest, "model")
	case "duration":
		return runRunsGetField(rest, "duration")
	case "age":
		return runRunsGetField(rest, "age")
	case "tags":
		return runRunsGetField(rest, "tags")
	case "active":
		return runRunsList(append([]string{"--active", "--all"}, rest...))
	case "last-failed":
		return runRunsList(append([]string{"--failed", "--limit=1"}, rest...))
	case "zero-commits":
		return runRunsList(append([]string{"--zero-commits", "--done", "--all"}, rest...))
	case "committed":
		return runRunsList(append([]string{"--has-diff", "--all"}, rest...))
	case "approvals":
		return runRunsList(append([]string{"--phase", "WAITING", "--approval-mode", "hitl", "--all"}, rest...))
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, runsUsage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", sub, runsUsage)
		os.Exit(2)
	}
	return nil
}

// runRunsGetField is a tiny helper for single-field shortcuts like "runs prompt" and "runs id".
func runRunsGetField(args []string, field string) error {
	newArgs := append([]string{"--field", field}, args...)
	return runRunsGet(newArgs)
}

// normalizeRunArgs moves run ID arguments (matching ar-xxxxxx) to the end of
// the args slice so that flag parsing works even when the user writes
// 'runs get <id> --flag' instead of 'runs get --flag <id>'.
func normalizeRunArgs(args []string) []string {
	var flags, ids []string
	i := 0
	for i < len(args) {
		arg := args[i]
		if isRunID(arg) {
			ids = append(ids, arg)
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// If the flag doesn't embed its value with '=', the next arg (if
			// it's not a flag and not a run ID) is the flag's value.
			if !strings.Contains(arg, "=") && i+1 < len(args) &&
				!strings.HasPrefix(args[i+1], "-") && !isRunID(args[i+1]) {
				flags = append(flags, args[i+1])
				i += 2
				continue
			}
			i++
			continue
		}
		// Non-flag, non-ID positional arg — leave in place (e.g. subcommand names).
		flags = append(flags, arg)
		i++
	}
	return append(flags, ids...)
}

// isRunID reports whether s looks like an agent run ID (ar- followed by 6 lowercase alphanumeric chars).
func isRunID(s string) bool {
	if len(s) != 9 || s[:3] != "ar-" {
		return false
	}
	for _, c := range s[3:] {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
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
	titleShortW := fs.String("title", "", "Shorthand for --title-contains")
	stage := fs.String("stage", "", "Filter by run stage (e.g. planning, executing, verifying)")
	active := fs.Bool("active", false, "Show only active runs (RUNNING + PENDING + WAITING)")
	sortBy := fs.String("sort", "", "Sort by field: started, phase, elapsed, title, model, project")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs watch [flags]\n\nAuto-refresh the runs list every N seconds. Press Ctrl+C to stop.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *titleShortW != "" && *titleContains == "" {
		*titleContains = *titleShortW
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
	if *stage != "" {
		listArgs = append(listArgs, "--stage="+*stage)
	}
	if *active {
		listArgs = append(listArgs, "--active")
	}
	if *sortBy != "" {
		listArgs = append(listArgs, "--sort="+*sortBy)
	}
	if *noColor {
		listArgs = append(listArgs, "--no-color")
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	for {
		if useColor {
			fmt.Print("\033[H\033[2J")
		}
		var filters []string
		if *active {
			filters = append(filters, "active")
		}
		if *project != "" {
			filters = append(filters, "project:"+*project)
		}
		if *feature != "" {
			filters = append(filters, "feature:"+*feature)
		}
		if *phase != "" {
			filters = append(filters, "phase:"+*phase)
		}
		if *tag != "" {
			filters = append(filters, "tag:"+*tag)
		}
		if *titleContains != "" {
			filters = append(filters, "title:"+*titleContains)
		}
		if *stage != "" {
			filters = append(filters, "stage:"+*stage)
		}
		if *since != "" {
			filters = append(filters, "since:"+*since)
		}
		filterStr := ""
		if len(filters) > 0 {
			filterStr = "  [" + strings.Join(filters, " ") + "]"
		}
		fmt.Printf("uncworks runs watch — every %ds  %s%s  (Ctrl+C to stop)\n\n",
			*interval, time.Now().Format("15:04:05"), filterStr)
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
	stage := fs.String("stage", "", "Filter by run stage (e.g. planning, executing, verifying)")
	cursor := fs.String("cursor", "", "Pagination cursor from previous response")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	all := fs.Bool("all", false, "Fetch all pages (overrides --limit)")
	repoURL := fs.String("repo-url", "", "Filter runs by repository URL (substring match)")
	titleContains := fs.String("title-contains", "", "Filter runs by display name substring (case-insensitive)")
	verbose := fs.Bool("verbose", false, "Show extra columns (repo, project)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	relative := fs.Bool("relative", false, "Show relative timestamps (e.g. '5m ago') instead of ISO")
	sortBy := fs.String("sort", "", "Sort by field: started, phase, elapsed, title, model, project (default: server order / most-recent-first)")
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
	showTags := fs.Bool("show-tags", false, "Add a tags column to the output")
	showPR := fs.Bool("show-pr", false, "Add a PR URL column to the output")
	showFeature := fs.Bool("show-feature", false, "Add a feature column to the output")
	showMessage := fs.Bool("show-message", false, "Add a STATUS MESSAGE column (truncated to 60 chars)")
	showDiff := fs.Bool("show-diff", false, "Add a DIFF column showing +additions/-deletions line counts")
	showCost := fs.Bool("show-cost", false, "Add a COST column showing estimated run cost")
	noModel := fs.Bool("no-model", false, "Hide the MODEL column")
	titleShort := fs.String("title", "", "Shorthand for --title-contains")
	countOnly := fs.Bool("count", false, "Print only the total count of matching runs")
	modelFilter := fs.String("model", "", "Filter by model tier substring (case-insensitive, e.g. deepseek, claude)")
	zeroCommits := fs.Bool("zero-commits", false, "Filter for succeeded runs that made no code changes")
	hasDiff := fs.Bool("has-diff", false, "Filter for runs that committed code changes (totalAdditions > 0 or totalDeletions > 0)")
	approvalModeFilter := fs.String("approval-mode", "", "Filter by approval mode (hitl, llm-judge, hybrid, or none for runs without approval)")
	showApproval := fs.Bool("show-approval", false, "Add an APPROVAL column to the output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs list [flags]\n\nList recent agent runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if *titleShort != "" && *titleContains == "" {
		*titleContains = *titleShort
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

	if *stage != "" {
		listReq.StageFilter = strings.ToLower(*stage)
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
		page := resp.Msg.GetAgentRuns()
		runs = append(runs, page...)
		nextCursor = resp.Msg.GetNextCursor()
		// When using --since without --include-archived, auto-paginate until we
		// pass the time window (archived runs are returned after non-archived in
		// a separate segment, so the early-exit heuristic doesn't work reliably
		// when --include-archived is set — in that case we must fetch all pages).
		passedSince := false
		if !sinceTime.IsZero() && !*includeArchived && len(page) > 0 {
			last := page[len(page)-1].GetCreatedAt()
			if last != nil && !last.AsTime().After(sinceTime) {
				passedSince = true
			}
		}
		if (!*all && !*activeOnly && (sinceTime.IsZero() || passedSince)) || nextCursor == "" {
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
	if *modelFilter != "" {
		needle := strings.ToLower(*modelFilter)
		filtered := runs[:0]
		for _, r := range runs {
			if strings.Contains(strings.ToLower(r.GetSpec().GetModelTier()), needle) {
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
	if *zeroCommits {
		filtered := runs[:0]
		for _, r := range runs {
			if r.GetStatus().GetPhase() == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED &&
				strings.Contains(r.GetStatus().GetMessage(), "no changes committed") {
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}
	if *hasDiff {
		filtered := runs[:0]
		for _, r := range runs {
			if r.GetStatus().GetTotalAdditions() > 0 || r.GetStatus().GetTotalDeletions() > 0 {
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}
	if *approvalModeFilter != "" {
		needle := strings.ToLower(*approvalModeFilter)
		filtered := runs[:0]
		for _, r := range runs {
			mode := strings.ToLower(r.GetSpec().GetApprovalMode())
			if needle == "none" {
				if mode == "" {
					filtered = append(filtered, r)
				}
			} else if mode == needle {
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
		case "elapsed", "duration":
			// Oldest started = longest elapsed time first.
			sort.Slice(runs, func(i, j int) bool {
				ti := runs[i].GetStatus().GetStartedAt()
				tj := runs[j].GetStatus().GetStartedAt()
				if ti == nil {
					return false
				}
				if tj == nil {
					return true
				}
				return ti.AsTime().Before(tj.AsTime())
			})
		case "title", "name":
			sort.Slice(runs, func(i, j int) bool {
				ti := runs[i].GetSpec().GetDisplayName()
				tj := runs[j].GetSpec().GetDisplayName()
				return strings.ToLower(ti) < strings.ToLower(tj)
			})
		case "model":
			sort.Slice(runs, func(i, j int) bool {
				return strings.ToLower(runs[i].GetSpec().GetModelTier()) < strings.ToLower(runs[j].GetSpec().GetModelTier())
			})
		case "project":
			sort.Slice(runs, func(i, j int) bool {
				return strings.ToLower(runs[i].GetSpec().GetProject()) < strings.ToLower(runs[j].GetSpec().GetProject())
			})
		default:
			return fmt.Errorf("--sort %q: must be started, phase, elapsed, title, model, or project", *sortBy)
		}
	}
	if len(runs) == 0 && !*jsonOut && !*idsOnly {
		fmt.Println("No runs found.")
		return nil
	}

	if *countOnly {
		fmt.Println(len(runs))
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
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Phase       string   `json:"phase"`
			Duration    string   `json:"duration"`
			Model       string   `json:"model"`
			CreatedAt   string   `json:"created_at,omitempty"`
			StartedAt   string   `json:"started_at,omitempty"`
			CompletedAt string   `json:"completed_at,omitempty"`
			Age         string   `json:"age,omitempty"`
			Project     string   `json:"project,omitempty"`
			Feature     string   `json:"feature,omitempty"`
			Tags        []string `json:"tags,omitempty"`
			ParentRunID string   `json:"parent_run_id,omitempty"`
			Repo        string   `json:"repo,omitempty"`
			Branch      string   `json:"branch,omitempty"`
			PRUrl       string   `json:"pr_url,omitempty"`
			Message     string   `json:"message,omitempty"`
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
			createdAt := ""
			if r.GetCreatedAt() != nil {
				createdAt = r.GetCreatedAt().AsTime().Format(time.RFC3339)
			}
			startedAt := ""
			if r.GetStatus().GetStartedAt() != nil {
				startedAt = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
			}
			completedAt := ""
			if r.GetStatus().GetCompletedAt() != nil {
				completedAt = r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339)
			}
			age := ""
			if r.GetCreatedAt() != nil {
				age = relativeTime(r.GetCreatedAt().AsTime())
			}
			repo := ""
			branch := ""
			if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
				repo = repos[0].GetUrl()
				branch = repos[0].GetBranch()
			}
			out = append(out, runJSON{
				ID:          r.GetId(),
				Title:       title,
				Phase:       phaseLabel(r.GetStatus().GetPhase()),
				Duration:    runDuration(r),
				Model:       model,
				CreatedAt:   createdAt,
				StartedAt:   startedAt,
				CompletedAt: completedAt,
				Age:         age,
				Project:     r.GetSpec().GetProject(),
				Feature:     r.GetSpec().GetFeature(),
				Tags:        r.GetSpec().GetTags(),
				ParentRunID: r.GetSpec().GetParentRunId(),
				Repo:        repo,
				Branch:      branch,
				PRUrl:       r.GetStatus().GetPrUrl(),
				Message:     r.GetStatus().GetMessage(),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	useRelative := *relative || term.IsTerminal(int(os.Stdout.Fd()))
	var listBuf bytes.Buffer
	w := tabwriter.NewWriter(&listBuf, 0, 0, 2, ' ', 0)
	if !*noHeader {
		hdr := "ID\tTITLE\tPHASE\tDURATION"
		if !*noModel {
			hdr += "\tMODEL"
		}
		hdr += "\tSTARTED"
		if *verbose {
			hdr += "\tREPO\tPROJECT\tSTAGE"
		}
		if *showFeature {
			hdr += "\tFEATURE"
		}
		if *showTags {
			hdr += "\tTAGS"
		}
		if *showPR {
			hdr += "\tPR"
		}
		if *showMessage {
			hdr += "\tMESSAGE"
		}
		if *showDiff {
			hdr += "\tDIFF"
		}
		if *showCost {
			hdr += "\tCOST"
		}
		if *showApproval {
			hdr += "\tAPPROVAL"
		}
		fmt.Fprintln(w, hdr)
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
			if useRelative {
				started = relativeTime(t)
			} else {
				started = t.Format(time.RFC3339)
			}
		} else if r.GetCreatedAt() != nil {
			t := r.GetCreatedAt().AsTime()
			if useRelative {
				started = relativeTime(t)
			} else {
				started = t.Format(time.RFC3339)
			}
		}
		duration := runDuration(r)
		tags := strings.Join(r.GetSpec().GetTags(), ",")
		prURL := r.GetStatus().GetPrUrl()

		row := r.GetId() + "\t" + title + "\t" + phase + "\t" + duration
		if !*noModel {
			row += "\t" + model
		}
		row += "\t" + started
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
			stageCol := r.GetStatus().GetStage()
			if stageCol == "" {
				stageCol = "—"
			}
			row += "\t" + repo + "\t" + project + "\t" + stageCol
		}
		if *showFeature {
			feat := r.GetSpec().GetFeature()
			if feat == "" {
				feat = "—"
			}
			row += "\t" + feat
		}
		if *showTags {
			row += "\t" + tags
		}
		if *showPR {
			if prURL == "" {
				prURL = "—"
			}
			row += "\t" + prURL
		}
		if *showMessage {
			msg := r.GetStatus().GetMessage()
			if msg == "" {
				msg = "—"
			} else if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			row += "\t" + msg
		}
		if *showDiff {
			add := r.GetStatus().GetTotalAdditions()
			del := r.GetStatus().GetTotalDeletions()
			if add > 0 || del > 0 {
				row += fmt.Sprintf("\t+%d/-%d", add, del)
			} else {
				row += "\t—"
			}
		}
		if *showCost {
			if cost := r.GetStatus().GetTotalCost(); cost != "" {
				row += "\t" + cost
			} else {
				row += "\t—"
			}
		}
		if *showApproval {
			approval := r.GetSpec().GetApprovalMode()
			if approval == "" {
				approval = "—"
			}
			row += "\t" + approval
		}
		fmt.Fprintln(w, row)
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

	// Aggregate diff stats for footer
	var totalAdd, totalDel int32
	for _, r := range runs {
		totalAdd += r.GetStatus().GetTotalAdditions()
		totalDel += r.GetStatus().GetTotalDeletions()
	}
	diffFooter := ""
	if totalAdd > 0 || totalDel > 0 {
		diffFooter = fmt.Sprintf("  ·  +%d -%d lines", totalAdd, totalDel)
	}

	if nextCursor != "" && !*all {
		fmt.Fprintf(os.Stderr, "next-cursor: %s\n", nextCursor)
		fmt.Printf("Showing %d run(s)%s%s — use --all or --limit to see more\n", len(runs), phaseSummary(), diffFooter)
	} else if *all {
		fmt.Printf("Showing all %d run(s)%s%s\n", len(runs), phaseSummary(), diffFooter)
	} else if len(runs) > 0 {
		isFiltered := !sinceTime.IsZero() || *repoURL != "" || *titleContains != "" ||
			*activeOnly || *runningOnly || *failedOnly || *pendingOnly || *waitingOnly || *doneOnly || *cancelledOnly ||
			*project != "" || *feature != "" || *tag != "" || *phase != ""
		suffix := ""
		if isFiltered {
			suffix = " (filtered)"
		}
		fmt.Printf("Showing %d run(s)%s%s%s\n", len(runs), suffix, phaseSummary(), diffFooter)
	}

	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRunsGet(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	showLog := fs.Bool("log", false, "Print the persisted agent log output")
	showLogs := fs.Bool("logs", false, "Alias for --log")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	short := fs.Bool("short", false, "Print a one-line summary: ID PHASE TITLE")
	waitFlag := fs.Bool("wait", false, "If the run is active, wait until it reaches a terminal phase then show details")
	poll := fs.Int("poll", 0, "Auto-refresh every N seconds until the run reaches a terminal phase (0 = disabled)")
	promptOnly := fs.Bool("prompt-only", false, "Print only the agent prompt text (useful for piping or editing)")
	field := fs.String("field", "", "Print only this field's value: id, phase, title, model, project, feature, branch, repo, pr-url, pod, duration, prompt, message, stage, age, created-at, created-at-iso, started-at, started-at-iso, completed-at, completed-at-iso, parent, tags")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
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
	if *lastRun && fs.NArg() == 0 {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		r0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(r0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		latestID := r0.Msg.GetAgentRuns()[0].GetId()
		var filtered []string
		for _, a := range args {
			if a != "--last" && a != "-last" {
				filtered = append(filtered, a)
			}
		}
		filtered = append(filtered, latestID) // ID must come after flags
		return runRunsGet(filtered)
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

	// --poll mode: refresh every N seconds until terminal.
	if *poll > 0 {
		useColorPoll := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
		for {
			if useColorPoll {
				fmt.Print("\033[2J\033[H")
			}
			subArgs := []string{id}
			if *server != "" {
				subArgs = append(subArgs, "--server="+*server)
			}
			if *showLog {
				subArgs = append(subArgs, "--log")
			}
			if *noColor {
				subArgs = append(subArgs, "--no-color")
			}
			if *short {
				subArgs = append(subArgs, "--short")
			}
			// Use a fresh flag set for the sub-call (no --poll to avoid recursion).
			_ = runRunsGet(subArgs)

			// Check phase to see if we should stop.
			client2, err2 := newClient(*server)
			if err2 == nil {
				resp2, err2 := client2.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
				if err2 == nil {
					ph := resp2.Msg.GetStatus().GetPhase()
					if ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
						ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED ||
						ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
						return nil
					}
				}
			}
			fmt.Printf("\n(refreshing every %ds — Ctrl+C to stop)\n", *poll)
			time.Sleep(time.Duration(*poll) * time.Second)
		}
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id})
	resp, err := client.GetAgentRun(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	// If --wait and run is active, wait for it to complete.
	if *waitFlag {
		phase := resp.Msg.GetStatus().GetPhase()
		isActive := phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
		if isActive {
			waitArgs := []string{id}
			if *server != "" {
				waitArgs = append(waitArgs, "--server="+*server)
			}
			// Ignore the error from wait (non-zero exit if failed); re-fetch for display.
			_ = runRunsWait(append(waitArgs, "--quiet"))
			resp2, err2 := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
			if err2 != nil {
				return fmt.Errorf("%s", humanizeErr(err2))
			}
			resp = resp2
		}
	}

	r := resp.Msg

	if *promptOnly {
		fmt.Println(r.GetSpec().GetPrompt())
		return nil
	}

	if *field != "" {
		var val string
		repo, branch := "", ""
		if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
			repo = repos[0].GetUrl()
			branch = repos[0].GetBranch()
		}
		switch strings.ToLower(*field) {
		case "id":
			val = r.GetId()
		case "phase":
			val = phaseLabel(r.GetStatus().GetPhase())
		case "title", "name":
			val = r.GetSpec().GetDisplayName()
		case "model", "model-tier":
			val = r.GetSpec().GetModelTier()
		case "project":
			val = r.GetSpec().GetProject()
		case "feature":
			val = r.GetSpec().GetFeature()
		case "branch":
			val = branch
		case "repo":
			val = repo
		case "pr-url", "pr":
			val = r.GetStatus().GetPrUrl()
		case "pod":
			val = r.GetStatus().GetPodName()
		case "prompt":
			val = r.GetSpec().GetPrompt()
		case "duration":
			val = runDuration(r)
		case "message", "status-message":
			val = r.GetStatus().GetMessage()
		case "stage":
			val = r.GetStatus().GetStage()
		case "age", "created-at":
			if ts := r.GetCreatedAt(); ts != nil {
				val = relativeTime(ts.AsTime())
			}
		case "created-at-iso":
			if ts := r.GetCreatedAt(); ts != nil {
				val = ts.AsTime().Format(time.RFC3339)
			}
		case "completed-at":
			if ts := r.GetStatus().GetCompletedAt(); ts != nil {
				val = relativeTime(ts.AsTime())
			}
		case "tags":
			val = strings.Join(r.GetSpec().GetTags(), ",")
		case "parent", "parent-run-id", "parent-id":
			val = r.GetSpec().GetParentRunId()
		case "started-at":
			if ts := r.GetStatus().GetStartedAt(); ts != nil {
				val = relativeTime(ts.AsTime())
			}
		case "started-at-iso":
			if ts := r.GetStatus().GetStartedAt(); ts != nil {
				val = ts.AsTime().Format(time.RFC3339)
			}
		case "completed-at-iso":
			if ts := r.GetStatus().GetCompletedAt(); ts != nil {
				val = ts.AsTime().Format(time.RFC3339)
			}
		case "question", "pending-question":
			if r.GetStatus().GetPhase() == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
				val = r.GetStatus().GetMessage()
			}
		default:
			return fmt.Errorf("unknown field %q: must be id, phase, title, model, project, feature, branch, repo, pr-url, pod, duration, prompt, message, stage, age, created-at, created-at-iso, started-at, started-at-iso, completed-at, completed-at-iso, parent, tags, or question", *field)
		}
		fmt.Println(val)
		return nil
	}

	if *jsonOut {
		type runGetJSON struct {
			ID                 string            `json:"id"`
			Title              string            `json:"title,omitempty"`
			Phase              string            `json:"phase"`
			Stage              string            `json:"stage,omitempty"`
			Message            string            `json:"message,omitempty"`
			VerificationResult string            `json:"verification_result,omitempty"`
			Project            string            `json:"project,omitempty"`
			Feature            string            `json:"feature,omitempty"`
			Prompt             string            `json:"prompt,omitempty"`
			Repo               string            `json:"repo,omitempty"`
			Branch             string            `json:"branch,omitempty"`
			Model              string            `json:"model,omitempty"`
			Tags               []string          `json:"tags,omitempty"`
			EnvVars            map[string]string `json:"env_vars,omitempty"`
			ParentRunID        string            `json:"parent_run_id,omitempty"`
			Children           []string          `json:"children,omitempty"`
			CreatedAt          string            `json:"created_at,omitempty"`
			Started            string            `json:"started,omitempty"`
			Completed          string            `json:"completed,omitempty"`
			Duration           string            `json:"duration,omitempty"`
			PrURL              string            `json:"pr_url,omitempty"`
			RetryCount         int32             `json:"retry_count,omitempty"`
			PodName            string            `json:"pod_name,omitempty"`
			TraceID            string            `json:"trace_id,omitempty"`
		}
		out := runGetJSON{
			ID:                 r.GetId(),
			Title:              r.GetSpec().GetDisplayName(),
			Phase:              phaseLabel(r.GetStatus().GetPhase()),
			Stage:              r.GetStatus().GetStage(),
			Message:            r.GetStatus().GetMessage(),
			VerificationResult: r.GetStatus().GetVerificationResult(),
			Project:            r.GetSpec().GetProject(),
			Feature:            r.GetSpec().GetFeature(),
			Prompt:             r.GetSpec().GetPrompt(),
			Model:              r.GetSpec().GetModelTier(),
			Tags:               r.GetSpec().GetTags(),
			EnvVars:            r.GetSpec().GetEnvVars(),
			Children:           r.GetChildren(),
			PrURL:              r.GetStatus().GetPrUrl(),
			RetryCount:         r.GetStatus().GetRetryCount(),
			PodName:            r.GetStatus().GetPodName(),
			TraceID:            r.GetStatus().GetTraceId(),
		}
		if r.GetCreatedAt() != nil {
			out.CreatedAt = r.GetCreatedAt().AsTime().Format(time.RFC3339)
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
	if r.GetStatus().GetRetryCount() > 0 {
		fmt.Printf("Retries:  %d\n", r.GetStatus().GetRetryCount())
	}
	if r.GetStatus().GetVerificationResult() != "" {
		fmt.Printf("Verify:   %s\n", r.GetStatus().GetVerificationResult())
	}
	if r.GetStatus().GetPrUrl() != "" {
		fmt.Printf("PR:       %s\n", r.GetStatus().GetPrUrl())
	}
	if cost := r.GetStatus().GetTotalCost(); cost != "" {
		add := r.GetStatus().GetTotalAdditions()
		del := r.GetStatus().GetTotalDeletions()
		if add > 0 || del > 0 {
			fmt.Printf("Cost:     %s  (+%d -%d lines)\n", cost, add, del)
		} else {
			fmt.Printf("Cost:     %s\n", cost)
		}
	} else if add, del := r.GetStatus().GetTotalAdditions(), r.GetStatus().GetTotalDeletions(); add > 0 || del > 0 {
		fmt.Printf("Diff:     +%d -%d lines\n", add, del)
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
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs tail", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs tail <id> [flags]\n\nStream logs and show a summary when the run completes.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		c, err := newClient(*server)
		if err != nil {
			return err
		}
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		if len(resp.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = resp.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

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
	if cost := r.GetStatus().GetTotalCost(); cost != "" {
		add := r.GetStatus().GetTotalAdditions()
		del := r.GetStatus().GetTotalDeletions()
		if add > 0 || del > 0 {
			fmt.Printf("Cost:     %s  (+%d -%d lines)\n", cost, add, del)
		} else {
			fmt.Printf("Cost:     %s\n", cost)
		}
	} else if add, del := r.GetStatus().GetTotalAdditions(), r.GetStatus().GetTotalDeletions(); add > 0 || del > 0 {
		fmt.Printf("Diff:     +%d -%d lines\n", add, del)
	}
	return nil
}

// ── logs ──────────────────────────────────────────────────────────────────────

func runRunsLogs(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs logs", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	noFollow := fs.Bool("no-follow", false, "Print stored log output only (don't stream live)")
	lines := fs.Int("lines", 0, "Show only the last N lines of output (0 = all)")
	head := fs.Int("head", 0, "Show only the first N lines of output (0 = all; --no-follow only)")
	timestamps := fs.Bool("timestamps", false, "Prefix each line with a timestamp")
	grep := fs.String("grep", "", "Only show lines matching this substring (case-insensitive; works in both streaming and --no-follow mode)")
	save := fs.String("save", "", "Save log output to a file (in addition to stdout)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs logs <id> [flags]\n\nStream log output until the run completes.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		c2, err2 := newClient(*server)
		if err2 != nil {
			return err2
		}
		resp2, err2 := c2.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err2 != nil {
			return fmt.Errorf("%s", humanizeErr(err2))
		}
		if runs := resp2.Msg.GetAgentRuns(); len(runs) > 0 {
			id = runs[0].GetId()
			fmt.Printf("Latest run: %s\n", id)
		} else {
			return fmt.Errorf("no runs found")
		}
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

	// --save only works in --no-follow mode.
	if *save != "" && !*noFollow {
		fmt.Fprintf(os.Stderr, "note: --save only works with --no-follow; output will not be saved in streaming mode\n")
	}

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

	// Automatically use stored logs for terminal runs (no need to stream).
	terminalPhase := func() bool {
		ph := getResp.Msg.GetStatus().GetPhase()
		return ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
			ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED ||
			ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	}
	if !*noFollow && terminalPhase() {
		*noFollow = true
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
		if *head > 0 && len(allLines) > *head {
			allLines = allLines[:*head]
		} else if *lines > 0 && len(allLines) > *lines {
			allLines = allLines[len(allLines)-*lines:]
		}
		var outputLines []string
		if *timestamps {
			ts := time.Now().Format("15:04:05")
			for _, line := range allLines {
				if line != "" {
					outputLines = append(outputLines, fmt.Sprintf("[%s] %s", ts, line))
				}
			}
		} else {
			outputLines = allLines
		}
		output := strings.Join(outputLines, "\n")
		fmt.Print(output)
		if len(output) > 0 && !strings.HasSuffix(output, "\n") {
			fmt.Println()
		}
		if *save != "" {
			if saveErr := os.WriteFile(*save, []byte(output+"\n"), 0o644); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save log to %q: %v\n", *save, saveErr)
			} else {
				fmt.Fprintf(os.Stderr, "log saved to %s\n", *save)
			}
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
		tsPrefix := ""
		if *timestamps && ev.GetTimestamp() != nil {
			tsPrefix = "[" + ev.GetTimestamp().AsTime().Local().Format("15:04:05") + "] "
		}
		switch ev.GetType() {
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
			payload := ev.GetPayload()
			if payload != "" {
				if grepNeedle != "" {
					for _, line := range strings.Split(payload, "\n") {
						if strings.Contains(strings.ToLower(line), grepNeedle) {
							fmt.Printf("%s%s\n", tsPrefix, line)
						}
					}
				} else if tsPrefix != "" {
					for _, line := range strings.Split(strings.TrimRight(payload, "\n"), "\n") {
						fmt.Printf("%s%s\n", tsPrefix, line)
					}
				} else {
					fmt.Print(payload)
				}
			}
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
			fmt.Printf("%s[phase: %s]\n", tsPrefix, ev.GetPayload())
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_WAITING_FOR_INPUT:
			fmt.Printf("%s[waiting for input: %s]\n", tsPrefix, ev.GetPayload())
			fmt.Println("Use 'uncworks input <id> <text>' to respond.")
		case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
			fmt.Printf("%s[completed: %s]\n", tsPrefix, ev.GetPayload())
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
	args = normalizeRunArgs(args)
	verb := "archive"
	if !archived {
		verb = "unarchive"
	}
	fs := flag.NewFlagSet("runs "+verb, flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: uncworks runs %s <id> [<id> ...] [flags]\n\nFlags:\n", verb)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	ids := fs.Args()
	if *lastRun {
		c, err := newClient(*server)
		if err != nil {
			return err
		}
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		if len(resp.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		ids = []string{resp.Msg.GetAgentRuns()[0].GetId()}
	} else if len(ids) == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	body, _ := json.Marshal(map[string]bool{"archived": archived})
	baseURL := serverBaseURL(*server)
	var errs []string
	for _, id := range ids {
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
	tag := fs.String("tag", "", "Only prune runs with this tag")
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
				TagFilter:     *tag,
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
	byProject := fs.Bool("by-project", false, "Show run count breakdown by project")
	byModel := fs.Bool("by-model", false, "Show run count breakdown by model tier")
	byFeatureStat := fs.Bool("by-feature", false, "Show run count breakdown by feature name")
	byTagStat := fs.Bool("by-tag", false, "Show run count breakdown by tag")
	trend := fs.Bool("trend", false, "Compare current --since window to the previous equal window (requires --since)")
	modelFilter := fs.String("model", "", "Filter by model tier substring (case-insensitive)")
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
	projectCounts := map[string]int{}
	modelCounts := map[string]int{}
	featureCounts2 := map[string]int{}
	tagCounts2 := map[string]int{}
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
		modelNeedle := strings.ToLower(*modelFilter)
		passedSinceStats := false
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					passedSinceStats = true
					continue
				}
			}
			if modelNeedle != "" && !strings.Contains(strings.ToLower(r.GetSpec().GetModelTier()), modelNeedle) {
				continue
			}
			label := phaseLabel(r.GetStatus().GetPhase())
			counts[label]++
			total++
			if *byProject {
				proj := r.GetSpec().GetProject()
				if proj == "" {
					proj = "(none)"
				}
				projectCounts[proj]++
			}
			if *byModel {
				model := r.GetSpec().GetModelTier()
				if model == "" {
					model = "default"
				}
				modelCounts[model]++
			}
			if *byFeatureStat {
				feat := r.GetSpec().GetFeature()
				if feat == "" {
					feat = "(none)"
				}
				featureCounts2[feat]++
			}
			if *byTagStat {
				tags := r.GetSpec().GetTags()
				if len(tags) == 0 {
					tagCounts2["(untagged)"]++
				} else {
					for _, t := range tags {
						tagCounts2[t]++
					}
				}
			}
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
		if cursor == "" || (*limit > 0 && total >= *limit) || (!sinceTime.IsZero() && passedSinceStats) {
			break
		}
	}

	sortedDurations := make([]time.Duration, len(doneDurations))
	copy(sortedDurations, doneDurations)
	sort.Slice(sortedDurations, func(i, j int) bool { return sortedDurations[i] < sortedDurations[j] })

	medianDuration := func() time.Duration {
		if len(sortedDurations) == 0 {
			return -1
		}
		mid := len(sortedDurations) / 2
		if len(sortedDurations)%2 == 0 {
			return (sortedDurations[mid-1] + sortedDurations[mid]) / 2
		}
		return sortedDurations[mid]
	}()

	avgDuration := func() time.Duration {
		if len(sortedDurations) == 0 {
			return -1
		}
		var total time.Duration
		for _, d := range sortedDurations {
			total += d
		}
		return total / time.Duration(len(sortedDurations))
	}()

	p90Duration := func() time.Duration {
		if len(sortedDurations) == 0 {
			return -1
		}
		idx := int(float64(len(sortedDurations)) * 0.9)
		if idx >= len(sortedDurations) {
			idx = len(sortedDurations) - 1
		}
		return sortedDurations[idx]
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
			out["avg_duration_seconds"] = avgDuration.Seconds()
			out["p90_duration_seconds"] = p90Duration.Seconds()
		}
		if *since != "" {
			out["window"] = *since
		}
		if *byProject {
			out["by_project"] = projectCounts
		}
		if *byModel {
			out["by_model"] = modelCounts
		}
		if *byFeatureStat {
			out["by_feature"] = featureCounts2
		}
		if *byTagStat {
			out["by_tag"] = tagCounts2
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
		fmt.Printf("Avg duration:    %s\n", avgDuration.Round(time.Second))
		fmt.Printf("Median duration: %s\n", medianDuration.Round(time.Second))
		fmt.Printf("P90 duration:    %s\n", p90Duration.Round(time.Second))
	} else if done > 0 {
		fmt.Printf("Duration: —\n")
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

	printBreakdown := func(label string, m map[string]int) {
		if len(m) == 0 {
			return
		}
		type kv struct {
			key   string
			count int
		}
		var pairs []kv
		for k, v := range m {
			pairs = append(pairs, kv{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].count != pairs[j].count {
				return pairs[i].count > pairs[j].count
			}
			return pairs[i].key < pairs[j].key
		})
		fmt.Printf("\n%s breakdown:\n", label)
		bw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, p := range pairs {
			pct := 0.0
			if total > 0 {
				pct = float64(p.count) / float64(total) * 100
			}
			fmt.Fprintf(bw, "  %s\t%d\t(%.1f%%)\n", p.key, p.count, pct)
		}
		_ = bw.Flush()
	}

	if *byProject {
		printBreakdown("Project", projectCounts)
	}
	if *byModel {
		printBreakdown("Model", modelCounts)
	}
	if *byFeatureStat {
		printBreakdown("Feature", featureCounts2)
	}
	if *byTagStat {
		printBreakdown("Tag", tagCounts2)
	}

	if *trend && *since != "" {
		d, _ := parseSinceDuration(*since)
		prevStart := sinceTime.Add(-d)
		// Fetch the previous period for comparison.
		prevTotal, prevDone, prevFailed := 0, 0, 0
		prevCursor := ""
		prevClient, _ := newClient(*server)
		for {
			lr := &apiv1.ListAgentRunsRequest{
				Limit:         100,
				ProjectFilter: *project,
				FeatureFilter: *feature,
				TagFilter:     *tag,
				Cursor:        prevCursor,
			}
			prevResp, prevErr := prevClient.ListAgentRuns(context.Background(), connect.NewRequest(lr))
			if prevErr != nil {
				break
			}
			for _, r := range prevResp.Msg.GetAgentRuns() {
				ts := r.GetCreatedAt()
				if ts == nil {
					continue
				}
				t := ts.AsTime()
				if !t.After(prevStart) || !t.Before(sinceTime) {
					continue
				}
				prevTotal++
				ph := phaseLabel(r.GetStatus().GetPhase())
				if ph == "DONE" {
					prevDone++
				} else if ph == "FAILED" {
					prevFailed++
				}
			}
			prevCursor = prevResp.Msg.GetNextCursor()
			if prevCursor == "" {
				break
			}
		}
		trendArrow := func(cur, prev int) string {
			if cur > prev {
				return "↑"
			} else if cur < prev {
				return "↓"
			}
			return "→"
		}
		fmt.Printf("\nTrend vs previous %s:\n", *since)
		fmt.Printf("  Total:   %d %s %d\n", total, trendArrow(total, prevTotal), prevTotal)
		fmt.Printf("  Done:    %d %s %d\n", done, trendArrow(done, prevDone), prevDone)
		fmt.Printf("  Failed:  %d %s %d\n", failed, trendArrow(failed, prevFailed), prevFailed)
		if done+failed > 0 && prevDone+prevFailed > 0 {
			curRate := float64(done) / float64(done+failed) * 100
			prevRate := float64(prevDone) / float64(prevDone+prevFailed) * 100
			fmt.Printf("  Success: %.1f%% %s %.1f%%\n", curRate, trendArrow(int(curRate), int(prevRate)), prevRate)
		}
	}

	return nil
}

// ── open ────────────────────────────────────────────────────────────────────────

func runRunsOpen(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs open", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	printURL := fs.Bool("print-url", false, "Print the PR URL instead of opening the browser")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs open <id> [flags]\n\nOpen the PR URL for a completed agent run in the default browser.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		r0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(r0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = r0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

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

// ── ui ────────────────────────────────────────────────────────────────────────

func runRunsUI(args []string) error {
	fs := flag.NewFlagSet("runs ui", flag.ContinueOnError)
	webURL := fs.String("web-url", "", "Override web dashboard base URL (e.g. http://host:port)")
	printURL := fs.Bool("print-url", false, "Print the URL instead of opening the browser")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs ui <id> [flags]\n\nOpen a run in the UNCWORKS web dashboard.\nRequires web_url in config (run: uncworks config set-web-url <url>).\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	var id string
	if *lastRun {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		r0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(r0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = r0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() == 0 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

	base := *webURL
	if base == "" {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		base = cfg.WebURL
	}
	if base == "" {
		return fmt.Errorf("web_url not configured — run: uncworks config set-web-url <url>")
	}
	base = strings.TrimRight(base, "/")
	url := base + "/run/" + id

	if *printURL {
		fmt.Println(url)
		return nil
	}
	fmt.Printf("Opening: %s\n", url)
	return openBrowser(url)
}

// ── inspect ──────────────────────────────────────────────────────────────────

func runRunsInspect(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs inspect", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	logLines := fs.Int("log-lines", 20, "Number of log tail lines to show (0 = all)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs inspect <id> [flags]\n\nDiagnostic view for a run: full details, graph, and log tail.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		client0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		resp0, err0 := client0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(resp0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = resp0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
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

// ── verify ────────────────────────────────────────────────────────────────────

func runRunsVerify(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	jsonOut := fs.Bool("json", false, "Print raw JSON output")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs verify <id> [flags]\n\nFetch and display the verification result for a spec-driven run.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	var id string
	if *lastRun && fs.NArg() == 0 {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		r0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(r0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = r0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() == 0 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	base := strings.TrimRight(cfg.WebURL, "/")
	if base == "" {
		return fmt.Errorf("web_url not configured — run: uncworks config set-web-url <url>")
	}
	url := base + "/api/v1/runs/" + id + "/verification"

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("fetch verification: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no verification result found for run %s (not a spec-driven run, or not yet completed)", id)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if *jsonOut {
		fmt.Println(string(body))
		return nil
	}

	type AutomatedCheck struct {
		Name    string `json:"name"`
		Pass    bool   `json:"pass"`
		Output  string `json:"output,omitempty"`
		Command string `json:"command,omitempty"`
	}
	type LLMVerdict struct {
		Pass          bool   `json:"pass"`
		Confidence    string `json:"confidence,omitempty"`
		Summary       string `json:"summary,omitempty"`
		FailureReason string `json:"failureReason,omitempty"`
	}
	type VerificationResult struct {
		Pass            bool             `json:"pass"`
		TasksCompleted  int              `json:"tasksCompleted"`
		TasksTotal      int              `json:"tasksTotal"`
		AutomatedChecks []AutomatedCheck `json:"automatedChecks"`
		LLMVerdict      *LLMVerdict      `json:"llmVerdict,omitempty"`
		FailureReport   string           `json:"failureReport,omitempty"`
		ExecutionTimeMs int64            `json:"executionTimeMs"`
	}

	var vr VerificationResult
	if err := json.Unmarshal(body, &vr); err != nil {
		return fmt.Errorf("parsing verification result: %w", err)
	}

	passStr := "PASSED"
	if !vr.Pass {
		passStr = "FAILED"
	}
	fmt.Printf("Verification: %s\n", passStr)
	if vr.TasksTotal > 0 {
		fmt.Printf("Tasks:        %d/%d completed\n", vr.TasksCompleted, vr.TasksTotal)
	}
	if vr.ExecutionTimeMs > 0 {
		fmt.Printf("Duration:     %s\n", (time.Duration(vr.ExecutionTimeMs) * time.Millisecond).Round(time.Second))
	}
	if len(vr.AutomatedChecks) > 0 {
		fmt.Println("\nChecks:")
		for _, c := range vr.AutomatedChecks {
			mark := "✓"
			if !c.Pass {
				mark = "✗"
			}
			out := c.Output
			if len(out) > 120 {
				out = out[:120] + "…"
			}
			if out != "" {
				fmt.Printf("  %s %s: %s\n", mark, c.Name, out)
			} else {
				fmt.Printf("  %s %s\n", mark, c.Name)
			}
		}
	}
	if vr.LLMVerdict != nil {
		fmt.Printf("\nLLM Verdict:  %s", map[bool]string{true: "pass", false: "fail"}[vr.LLMVerdict.Pass])
		if vr.LLMVerdict.Confidence != "" {
			fmt.Printf(" (confidence: %s)", vr.LLMVerdict.Confidence)
		}
		fmt.Println()
		if vr.LLMVerdict.Summary != "" {
			fmt.Printf("Summary:      %s\n", vr.LLMVerdict.Summary)
		}
	}
	if vr.FailureReport != "" {
		fmt.Printf("\nFailure:\n  %s\n", vr.FailureReport)
	}
	return nil
}

// ── diff ──────────────────────────────────────────────────────────────────────

func runRunsDiff(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs diff", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	stat := fs.Bool("stat", false, "Show git diff --stat instead of full diff")
	execFlag := fs.Bool("exec", false, "Run git fetch + diff (default when stdout is a TTY)")
	printCmd := fs.Bool("print-cmd", false, "Print git commands instead of executing them")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs diff <id> [flags]\n\nFetch and show the git diff for a completed run's branch.\nWhen stdout is a TTY, runs git automatically; otherwise prints the git commands.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		client0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		resp0, err0 := client0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(resp0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = resp0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

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

	agentBranch := fmt.Sprintf("aot/%s", id)
	diffArgs := []string{"diff", fmt.Sprintf("origin/%s...origin/%s", baseBranch, agentBranch)}
	if *stat {
		diffArgs = []string{"diff", "--stat", fmt.Sprintf("origin/%s...origin/%s", baseBranch, agentBranch)}
	}

	if prURL != "" {
		fmt.Printf("Run:       %s\n", id)
		fmt.Printf("Repo:      %s\n", repoURL)
		fmt.Printf("Base:      %s\n", baseBranch)
		fmt.Printf("PR:        %s\n", prURL)
		fmt.Println()
	} else {
		fmt.Printf("Run:       %s\n", id)
		fmt.Printf("Repo:      %s\n", repoURL)
		fmt.Printf("Branch:    %s\n", agentBranch)
		fmt.Println()
	}

	shouldExec := *execFlag || (!*printCmd && term.IsTerminal(int(os.Stdout.Fd())))
	if shouldExec {
		fmt.Printf("$ git fetch origin %s\n", agentBranch)
		fetchCmd := exec.Command("git", "fetch", "origin", agentBranch)
		fetchCmd.Stdout = os.Stdout
		fetchCmd.Stderr = os.Stderr
		if err := fetchCmd.Run(); err != nil {
			return fmt.Errorf("git fetch failed: %w", err)
		}
		fmt.Printf("$ git %s\n\n", strings.Join(diffArgs, " "))
		diffCmd := exec.Command("git", diffArgs...)
		diffCmd.Stdout = os.Stdout
		diffCmd.Stderr = os.Stderr
		return diffCmd.Run()
	}

	statFlagStr := ""
	if *stat {
		statFlagStr = " --stat"
	}
	fmt.Printf("To view the diff:\n")
	fmt.Printf("  git fetch origin %s\n", agentBranch)
	fmt.Printf("  git diff%s origin/%s...origin/%s\n", statFlagStr, baseBranch, agentBranch)
	return nil
}

// ── commits ──────────────────────────────────────────────────────────────────

func runRunsCommits(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs commits", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	oneline := fs.Bool("oneline", false, "Show one-line log output (default: full log)")
	execFlag := fs.Bool("exec", false, "Run git fetch + log (default when stdout is a TTY)")
	printCmd := fs.Bool("print-cmd", false, "Print git commands instead of executing them")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs commits <id> [flags]\n\nShow git commits made by a run on its feature branch.\nWhen stdout is a TTY, runs git automatically; otherwise prints the git commands.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var id string
	if *lastRun {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		resp0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(resp0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = resp0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

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
	baseBranch := repos[0].GetBranch()
	if baseBranch == "" {
		baseBranch = "main"
	}
	agentBranch := fmt.Sprintf("aot/%s", id)

	fmt.Printf("Run:   %s\n", id)
	fmt.Printf("Base:  %s\n", baseBranch)
	fmt.Printf("Agent: %s\n\n", agentBranch)

	logArgs := []string{"log", fmt.Sprintf("origin/%s..origin/%s", baseBranch, agentBranch)}
	if *oneline {
		logArgs = append(logArgs, "--oneline")
	}

	shouldExec := *execFlag || (!*printCmd && term.IsTerminal(int(os.Stdout.Fd())))
	if shouldExec {
		fmt.Printf("$ git fetch origin %s\n", agentBranch)
		fetchCmd := exec.Command("git", "fetch", "origin", agentBranch)
		fetchCmd.Stdout = os.Stdout
		fetchCmd.Stderr = os.Stderr
		if err := fetchCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "git fetch failed: %v\n", err)
		}
		fmt.Printf("$ git %s\n\n", strings.Join(logArgs, " "))
		logCmd := exec.Command("git", logArgs...)
		logCmd.Stdout = os.Stdout
		logCmd.Stderr = os.Stderr
		return logCmd.Run()
	}

	fmt.Printf("To view commits:\n")
	fmt.Printf("  git fetch origin %s\n", agentBranch)
	fmt.Printf("  git log origin/%s..origin/%s\n", baseBranch, agentBranch)
	return nil
}

// ── retry ────────────────────────────────────────────────────────────────────

func runRunsRetry(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs retry", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	prompt := fs.String("prompt", "", "Override the agent prompt")
	promptFile := fs.String("prompt-file", "", "Read the override prompt from a file")
	editPrompt := fs.Bool("editor", false, "Open $EDITOR to compose the override prompt interactively")
	appendPrompt := fs.String("append-prompt", "", "Append additional context to the original prompt")
	branch := fs.String("branch", "", "Override the branch")
	modelTier := fs.String("model-tier", "", "Override the model tier")
	modelShort := fs.String("model", "", "Shorthand for --model-tier")
	name := fs.String("name", "", "Override the display name")
	autoPush := fs.Bool("auto-push", false, "Push changes to a feature branch after the run succeeds")
	autoPR := fs.Bool("auto-pr", false, "Create a GitHub PR after the run succeeds (implies --auto-push)")
	outputID := fs.Bool("output-id", false, "Print only the new run ID (for scripting)")
	wait := fs.Bool("wait", false, "Wait for the retried run to complete; exit 0 on success, 1 on failure")
	follow := fs.Bool("follow", false, "Stream logs after submitting (takes precedence over --wait)")
	diffFlag := fs.Bool("diff", false, "Show git diff commands for the original run (before waiting)")
	var envFlags multiFlag
	fs.Var(&envFlags, "env", "Override environment variables (repeatable, KEY=VALUE); replaces all env vars if any are provided")
	var addEnvFlags multiFlag
	fs.Var(&addEnvFlags, "add-env", "Add or override individual environment variables (repeatable, KEY=VALUE); merged with existing env vars")
	var tagFlags multiFlag
	fs.Var(&tagFlags, "tag", "Override tags (repeatable); replaces all tags if any are provided")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs retry <id> [flags]\n\nCreate a new run with the same spec as an existing run. Use flags to override specific fields.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *modelShort != "" && *modelTier == "" {
		*modelTier = *modelShort
	}
	// Load prompt from file if specified.
	if *promptFile != "" {
		raw, err := os.ReadFile(*promptFile)
		if err != nil {
			return fmt.Errorf("reading prompt file %q: %w", *promptFile, err)
		}
		*prompt = strings.TrimRight(string(raw), "\n")
	}
	// Open $EDITOR to compose the prompt interactively.
	if *editPrompt && *prompt == "" {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vi"
		}
		tmpf, tmpErr := os.CreateTemp("", "uncworks-retry-prompt-*.txt")
		if tmpErr != nil {
			return fmt.Errorf("creating temp file for editor: %w", tmpErr)
		}
		tmpPath := tmpf.Name()
		_ = tmpf.Close()
		defer os.Remove(tmpPath)
		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}
		raw, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("reading editor output: %w", err)
		}
		*prompt = strings.TrimSpace(string(raw))
		if *prompt == "" {
			return fmt.Errorf("prompt is empty (editor produced no content)")
		}
	}
	if *lastRun && fs.NArg() == 0 {
		c0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		r0, err0 := c0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(r0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		latestRetryID := r0.Msg.GetAgentRuns()[0].GetId()
		var filtered []string
		for _, a := range args {
			if a != "--last" && a != "-last" {
				filtered = append(filtered, a)
			}
		}
		filtered = append(filtered, latestRetryID)
		return runRunsRetry(filtered)
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	// Multi-ID support: retry each and collect new IDs.
	if fs.NArg() > 1 {
		ids := fs.Args()
		var newIDs []string
		for _, rid := range ids {
			subArgs := []string{rid}
			if *server != "" {
				subArgs = append(subArgs, "--server="+*server)
			}
			if *prompt != "" {
				subArgs = append(subArgs, "--prompt="+*prompt)
			}
			if *appendPrompt != "" {
				subArgs = append(subArgs, "--append-prompt="+*appendPrompt)
			}
			if *modelTier != "" {
				subArgs = append(subArgs, "--model-tier="+*modelTier)
			}
			if *branch != "" {
				subArgs = append(subArgs, "--branch="+*branch)
			}
			if *name != "" {
				subArgs = append(subArgs, "--name="+*name)
			}
			if *autoPush {
				subArgs = append(subArgs, "--auto-push")
			}
			if *autoPR {
				subArgs = append(subArgs, "--auto-pr")
			}
			if *outputID {
				subArgs = append(subArgs, "--output-id")
			}
			if *diffFlag {
				subArgs = append(subArgs, "--diff")
			}
			for _, t := range tagFlags {
				subArgs = append(subArgs, "--tag="+t)
			}
			for _, e := range envFlags {
				subArgs = append(subArgs, "--env="+e)
			}
			for _, e := range addEnvFlags {
				subArgs = append(subArgs, "--add-env="+e)
			}
			if err := runRunsRetry(subArgs); err != nil {
				fmt.Fprintf(os.Stderr, "  failed to retry %s: %v\n", rid, err)
			} else if *outputID {
				newIDs = append(newIDs, rid) // id is printed in subArgs call
			}
		}
		if *wait && len(newIDs) > 0 {
			waitArgs := append(newIDs, "--server="+*server)
			return runRunsWait(waitArgs)
		}
		return nil
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
	if *appendPrompt != "" {
		newSpec.Prompt = strings.TrimRight(newSpec.Prompt, "\n") + "\n\n" + *appendPrompt
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
	if len(addEnvFlags) > 0 {
		if newSpec.EnvVars == nil {
			newSpec.EnvVars = map[string]string{}
		}
		for _, kv := range addEnvFlags {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("--add-env %q: must be KEY=VALUE", kv)
			}
			newSpec.EnvVars[parts[0]] = parts[1]
		}
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

	if *diffFlag && !*outputID {
		repos := orig.GetSpec().GetRepos()
		if len(repos) > 0 && (orig.GetSpec().GetAutoPush() || orig.GetStatus().GetPrUrl() != "") {
			baseBranch := repos[0].GetBranch()
			if baseBranch == "" {
				baseBranch = "main"
			}
			agentBranch := fmt.Sprintf("aot/%s", id)
			fmt.Println()
			fmt.Printf("Original run diff (%s):\n", id)
			if prURL := orig.GetStatus().GetPrUrl(); prURL != "" {
				fmt.Printf("  PR: %s\n", prURL)
			}
			fmt.Printf("  git fetch origin %s\n", agentBranch)
			fmt.Printf("  git diff origin/%s...origin/%s\n", baseBranch, agentBranch)
		}
	}

	if *follow {
		return runRunsTail([]string{newRun.GetId(), "--server=" + *server})
	}
	if *wait {
		return runRunsWait([]string{newRun.GetId(), "--server=" + *server})
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
	listOnly := fs.Bool("list", false, "Print a table of runs that would be retried, then exit (implies --dry-run)")
	dryRun := fs.Bool("dry-run", false, "Print what would be retried without actually doing it")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	verbose := fs.Bool("verbose", false, "Show a prompt preview for each run before confirming")
	modelTier := fs.String("model-tier", "", "Override model tier for all retried runs")
	appendPrompt := fs.String("append-prompt", "", "Append this text to the original prompt of each retried run")
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

	if *listOnly {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tSTARTED\tTITLE\n")
		for _, r := range failedRuns {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			started := ""
			if ts := r.GetCreatedAt(); ts != nil {
				started = relativeTime(ts.AsTime())
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.GetId(), started, title)
		}
		w.Flush()
		fmt.Printf("\n%d run(s) would be retried.\n", len(failedRuns))
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
		age := ""
		if ts := r.GetCreatedAt(); ts != nil {
			age = "  (" + relativeTime(ts.AsTime()) + ")"
		}
		fmt.Printf("  %s  %s%s\n", r.GetId(), title, age)
		if *verbose {
			prompt := r.GetSpec().GetPrompt()
			if len(prompt) > 120 {
				prompt = prompt[:117] + "..."
			}
			fmt.Printf("           prompt: %s\n", strings.ReplaceAll(prompt, "\n", " "))
		}
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
		if *appendPrompt != "" {
			newSpec.Prompt = strings.TrimRight(newSpec.Prompt, "\n") + "\n\n" + *appendPrompt
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
	tag := fs.String("tag", "", "Only cancel runs with this tag")
	titleContains := fs.String("title-contains", "", "Only cancel runs whose title contains this string (case-insensitive)")
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
			TagFilter:     *tag,
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
			if *titleContains != "" {
				needle := strings.ToLower(*titleContains)
				if !strings.Contains(strings.ToLower(r.GetSpec().GetDisplayName()), needle) {
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
	idsOnly := fs.Bool("ids-only", false, "Print only run IDs (one per line, for scripting)")
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

	if *idsOnly {
		for _, r := range runs {
			fmt.Println(r.GetId())
		}
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
	tag := fs.String("tag", "", "Filter by tag")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, PENDING, WAITING, CANCELLED)")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	outFile := fs.String("out", "", "Write output to file instead of stdout")
	format := fs.String("format", "csv", "Output format: csv, tsv, json, or markdown")
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
			ID          string   `json:"id"`
			Title       string   `json:"title,omitempty"`
			Phase       string   `json:"phase"`
			Message     string   `json:"message,omitempty"`
			Project     string   `json:"project,omitempty"`
			Feature     string   `json:"feature,omitempty"`
			Model       string   `json:"model,omitempty"`
			CreatedAt   string   `json:"created_at,omitempty"`
			StartedAt   string   `json:"started_at,omitempty"`
			CompletedAt string   `json:"completed_at,omitempty"`
			DurationS   float64  `json:"duration_s,omitempty"`
			PrURL       string   `json:"pr_url,omitempty"`
			Tags        []string `json:"tags,omitempty"`
			ParentRunID string   `json:"parent_run_id,omitempty"`
			Repo        string   `json:"repo,omitempty"`
			Branch      string   `json:"branch,omitempty"`
		}
		var rows []exportJSON
		for _, r := range allRuns {
			row := exportJSON{
				ID:          r.GetId(),
				Title:       r.GetSpec().GetDisplayName(),
				Phase:       phaseLabel(r.GetStatus().GetPhase()),
				Message:     r.GetStatus().GetMessage(),
				Project:     r.GetSpec().GetProject(),
				Feature:     r.GetSpec().GetFeature(),
				Model:       r.GetSpec().GetModelTier(),
				PrURL:       r.GetStatus().GetPrUrl(),
				Tags:        r.GetSpec().GetTags(),
				ParentRunID: r.GetSpec().GetParentRunId(),
			}
			if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
				row.Repo = repos[0].GetUrl()
				row.Branch = repos[0].GetBranch()
			}
			if r.GetCreatedAt() != nil {
				row.CreatedAt = r.GetCreatedAt().AsTime().Format(time.RFC3339)
			}
			if r.GetStatus().GetStartedAt() != nil {
				row.StartedAt = r.GetStatus().GetStartedAt().AsTime().Format(time.RFC3339)
			}
			if r.GetStatus().GetCompletedAt() != nil {
				row.CompletedAt = r.GetStatus().GetCompletedAt().AsTime().Format(time.RFC3339)
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

	if *format == "markdown" {
		fmt.Fprintln(out, "| ID | Title | Phase | Project | Feature | Model | Duration | Started | PR |")
		fmt.Fprintln(out, "|---|---|---|---|---|---|---|---|---|")
		for _, r := range allRuns {
			title := r.GetSpec().GetDisplayName()
			started := ""
			if ts := r.GetStatus().GetStartedAt(); ts != nil {
				started = ts.AsTime().Format("2006-01-02 15:04")
			}
			dur := runDuration(r)
			prURL := r.GetStatus().GetPrUrl()
			prCell := ""
			if prURL != "" {
				prCell = "[PR](" + prURL + ")"
			}
			fmt.Fprintf(out, "| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				r.GetId(), title, phaseLabel(r.GetStatus().GetPhase()),
				r.GetSpec().GetProject(), r.GetSpec().GetFeature(),
				r.GetSpec().GetModelTier(), dur, started, prCell)
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
	_ = w.Write([]string{"id", "title", "phase", "project", "feature", "model", "repo", "branch", "started", "completed", "duration_s", "pr_url", "tags", "parent_run_id"})
	for _, r := range allRuns {
		started := ""
		completed := ""
		durationS := ""
		repoURL := ""
		branch := ""
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
		if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
			repoURL = repos[0].GetUrl()
			branch = repos[0].GetBranch()
		}
		_ = w.Write([]string{
			r.GetId(),
			r.GetSpec().GetDisplayName(),
			phaseLabel(r.GetStatus().GetPhase()),
			r.GetSpec().GetProject(),
			r.GetSpec().GetFeature(),
			r.GetSpec().GetModelTier(),
			repoURL,
			branch,
			started,
			completed,
			durationS,
			r.GetStatus().GetPrUrl(),
			strings.Join(r.GetSpec().GetTags(), ";"),
			r.GetSpec().GetParentRunId(),
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

// ── histogram ─────────────────────────────────────────────────────────────────

func runRunsHistogram(args []string) error {
	fs := flag.NewFlagSet("runs histogram", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	since := fs.String("since", "24h", "Time window to cover (e.g. 1h, 24h, 7d)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	buckets := fs.Int("buckets", 0, "Number of time buckets (0 = auto: 24 for <=24h windows, 7 for <=7d, else 30)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	jsonOut := fs.Bool("json", false, "Output as JSON array of bucket objects")
	sparkline := fs.Bool("sparkline", false, "Output a compact single-line sparkline using Unicode block chars")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs histogram [flags]\n\nShow a bar chart of run starts bucketed over a time window.\n\nFlags:")
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

	numBuckets := *buckets
	if numBuckets <= 0 {
		switch {
		case d <= 24*time.Hour:
			numBuckets = 24
		case d <= 7*24*time.Hour:
			numBuckets = 7
		default:
			numBuckets = 30
		}
	}
	bucketDur := d / time.Duration(numBuckets)

	counts := make([]int, numBuckets)
	phaseBuckets := map[string][]int{
		"DONE":    make([]int, numBuckets),
		"FAILED":  make([]int, numBuckets),
		"RUNNING": make([]int, numBuckets),
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	cursor := ""
	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}))
		if err != nil {
			// On connection error mid-pagination, render with data collected so far.
			break
		}
		done := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetStatus().GetStartedAt()
			if ts == nil {
				ts = r.GetCreatedAt()
			}
			if ts == nil {
				continue
			}
			t := ts.AsTime()
			if t.Before(sinceTime) {
				done = true
				break
			}
			offset := time.Since(sinceTime) - time.Since(t)
			idx := int(offset / bucketDur)
			if idx < 0 {
				idx = 0
			}
			if idx >= numBuckets {
				idx = numBuckets - 1
			}
			counts[idx]++
			label := phaseLabel(r.GetStatus().GetPhase())
			if pb, ok := phaseBuckets[label]; ok {
				pb[idx]++
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || done {
			break
		}
	}

	maxCount := 0
	for _, n := range counts {
		if n > maxCount {
			maxCount = n
		}
	}

	if *jsonOut {
		type bucketJSON struct {
			Start   string `json:"start"`
			End     string `json:"end"`
			Count   int    `json:"count"`
			Done    int    `json:"done"`
			Failed  int    `json:"failed"`
			Running int    `json:"running"`
		}
		var out []bucketJSON
		for i := 0; i < numBuckets; i++ {
			start := sinceTime.Add(time.Duration(i) * bucketDur)
			end := start.Add(bucketDur)
			out = append(out, bucketJSON{
				Start:   start.UTC().Format(time.RFC3339),
				End:     end.UTC().Format(time.RFC3339),
				Count:   counts[i],
				Done:    phaseBuckets["DONE"][i],
				Failed:  phaseBuckets["FAILED"][i],
				Running: phaseBuckets["RUNNING"][i],
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))

	if *sparkline {
		const blocks = "▁▂▃▄▅▆▇█"
		runeBlocks := []rune(blocks)
		nBlocks := len(runeBlocks)
		var sb strings.Builder
		for _, n := range counts {
			if n == 0 {
				sb.WriteRune(' ')
			} else if maxCount == 0 {
				sb.WriteRune(runeBlocks[0])
			} else {
				idx := int(float64(n)/float64(maxCount)*float64(nBlocks-1) + 0.5)
				if idx >= nBlocks {
					idx = nBlocks - 1
				}
				sb.WriteRune(runeBlocks[idx])
			}
		}
		total := 0
		for _, n := range counts {
			total += n
		}
		from := sinceTime.Format("01/02 15:04")
		to := time.Now().Format("01/02 15:04")
		fmt.Printf("%s  (%s to %s, %d runs)\n", sb.String(), from, to, total)
		return nil
	}

	const barWidth = 30
	fmt.Printf("Run activity — last %s  (bucket size: %s)\n\n", *since, bucketDur.Round(time.Minute))

	for i := 0; i < numBuckets; i++ {
		bucketStart := sinceTime.Add(time.Duration(i) * bucketDur)
		label := bucketStart.Format("01/02 15:04")
		if bucketDur >= 24*time.Hour {
			label = bucketStart.Format("01/02")
		} else if bucketDur >= time.Hour {
			label = bucketStart.Format("15:04")
		}

		n := counts[i]
		barLen := 0
		if maxCount > 0 {
			barLen = int(float64(n) / float64(maxCount) * barWidth)
		}
		bar := strings.Repeat("█", barLen)

		doneN := phaseBuckets["DONE"][i]
		failN := phaseBuckets["FAILED"][i]

		if useColor && n > 0 {
			color := "\033[32m" // green for mostly done
			if failN > doneN {
				color = "\033[31m" // red if more failures
			} else if failN > 0 {
				color = "\033[33m" // yellow if mixed
			}
			bar = color + bar + "\033[0m"
		}

		fmt.Printf("  %s │%-*s %d", label, barWidth, bar, n)
		if n > 0 {
			extras := []string{}
			if doneN > 0 {
				extras = append(extras, fmt.Sprintf("%d✓", doneN))
			}
			if failN > 0 {
				extras = append(extras, fmt.Sprintf("%d✗", failN))
			}
			if len(extras) > 0 {
				fmt.Printf(" (%s)", strings.Join(extras, " "))
			}
		}
		fmt.Println()
	}

	total := 0
	for _, n := range counts {
		total += n
	}
	fmt.Printf("\nTotal: %d runs\n", total)
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
	byFeature := fs.Bool("by-feature", false, "Show count breakdown by feature name")
	byTag := fs.Bool("by-tag", false, "Show count breakdown by tag (runs with multiple tags are counted per tag)")
	byProject := fs.Bool("by-project", false, "Show count breakdown by project name")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	modelFilter := fs.String("model", "", "Filter by model tier substring (case-insensitive)")
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
	featureCounts := map[string]int{}
	tagCounts := map[string]int{}
	projectCounts := map[string]int{}
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
			break
		}
		countModelNeedle := strings.ToLower(*modelFilter)
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetStatus().GetStartedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					continue
				}
			}
			if countModelNeedle != "" && !strings.Contains(strings.ToLower(r.GetSpec().GetModelTier()), countModelNeedle) {
				continue
			}
			count++
			if *byPhase {
				phaseCounts[phaseLabel(r.GetStatus().GetPhase())]++
			}
			if *byFeature {
				feat := r.GetSpec().GetFeature()
				if feat == "" {
					feat = "(none)"
				}
				featureCounts[feat]++
			}
			if *byTag {
				tags := r.GetSpec().GetTags()
				if len(tags) == 0 {
					tagCounts["(untagged)"]++
				} else {
					for _, t := range tags {
						tagCounts[t]++
					}
				}
			}
			if *byProject {
				proj := r.GetSpec().GetProject()
				if proj == "" {
					proj = "(none)"
				}
				projectCounts[proj]++
			}
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if *jsonOut {
		out := map[string]interface{}{"count": count}
		if *byPhase {
			out["by_phase"] = phaseCounts
		}
		if *byFeature {
			out["by_feature"] = featureCounts
		}
		if *byTag {
			out["by_tag"] = tagCounts
		}
		if *byProject {
			out["by_project"] = projectCounts
		}
		if *byPhase || *byFeature || *byTag || *byProject {
			out["total"] = count
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
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
	} else if *byFeature {
		type pair struct {
			k string
			v int
		}
		var pairs []pair
		for k, v := range featureCounts {
			pairs = append(pairs, pair{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].v != pairs[j].v {
				return pairs[i].v > pairs[j].v
			}
			return pairs[i].k < pairs[j].k
		})
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "FEATURE\tCOUNT")
		for _, p := range pairs {
			fmt.Fprintf(w, "%s\t%d\n", p.k, p.v)
		}
		w.Flush()
		fmt.Printf("Total: %d\n", count)
	} else if *byTag {
		type pair struct {
			k string
			v int
		}
		var pairs []pair
		for k, v := range tagCounts {
			pairs = append(pairs, pair{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].v != pairs[j].v {
				return pairs[i].v > pairs[j].v
			}
			return pairs[i].k < pairs[j].k
		})
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TAG\tCOUNT")
		for _, p := range pairs {
			fmt.Fprintf(w, "%s\t%d\n", p.k, p.v)
		}
		w.Flush()
		fmt.Printf("Total: %d\n", count)
	} else if *byProject {
		type pair struct {
			k string
			v int
		}
		var pairs []pair
		for k, v := range projectCounts {
			pairs = append(pairs, pair{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].v != pairs[j].v {
				return pairs[i].v > pairs[j].v
			}
			return pairs[i].k < pairs[j].k
		})
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROJECT\tCOUNT")
		for _, p := range pairs {
			fmt.Fprintf(w, "%s\t%d\n", p.k, p.v)
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
	watch := fs.Bool("watch", false, "Auto-refresh the summary every --interval seconds (Ctrl+C to stop)")
	interval := fs.Int("interval", 10, "Refresh interval in seconds for --watch mode")
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

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	useColorSum := term.IsTerminal(int(os.Stdout.Fd()))

	doSummary := func() error {
		sinceTime := time.Now().Add(-d)

	phaseCounts := map[string]int{}
	projectCounts := map[string]int{}
	var activeRuns []*apiv1.AgentRun
	var recentCompleted []*apiv1.AgentRun
	var recentFailed []*apiv1.AgentRun
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
			break
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil || !ts.AsTime().After(sinceTime) {
				continue
			}
			total++
			label := phaseLabel(r.GetStatus().GetPhase())
			phaseCounts[label]++
			if proj := r.GetSpec().GetProject(); proj != "" {
				projectCounts[proj]++
			}
			switch r.GetStatus().GetPhase() {
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
				activeRuns = append(activeRuns, r)
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
				if len(recentFailed) < 5 {
					recentFailed = append(recentFailed, r)
				}
				if len(recentCompleted) < 5 {
					recentCompleted = append(recentCompleted, r)
				}
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
				apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
				if len(recentCompleted) < 5 {
					recentCompleted = append(recentCompleted, r)
				}
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

	colorPhase := func(label string) string {
		if !useColorSum {
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

	if len(projectCounts) > 0 && *project == "" {
		type projEntry struct {
			name  string
			count int
		}
		var projs []projEntry
		for k, v := range projectCounts {
			projs = append(projs, projEntry{k, v})
		}
		sort.Slice(projs, func(i, j int) bool {
			if projs[i].count != projs[j].count {
				return projs[i].count > projs[j].count
			}
			return projs[i].name < projs[j].name
		})
		maxProj := 5
		if len(projs) < maxProj {
			maxProj = len(projs)
		}
		fmt.Printf("\nTop projects:\n")
		wProj := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, p := range projs[:maxProj] {
			pct := p.count * 100 / total
			fmt.Fprintf(wProj, "  %s\t%d\t(%d%%)\n", p.name, p.count, pct)
		}
		wProj.Flush()
	}

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

	if len(recentCompleted) > 0 {
		fmt.Printf("\nRecent completions:\n")
		for _, r := range recentCompleted {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			age := ""
			if ts := r.GetStatus().GetCompletedAt(); ts != nil {
				age = "  " + relativeTime(ts.AsTime())
			}
			fmt.Printf("  %s  %-40s  %s%s\n", r.GetId(), title, colorPhase(phaseLabel(r.GetStatus().GetPhase())), age)
		}
	}

	if len(recentFailed) > 0 {
		fmt.Printf("\nRecent failures:\n")
		wf := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, r := range recentFailed {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			if len(title) > 30 {
				title = title[:27] + "..."
			}
			msg := r.GetStatus().GetMessage()
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			age := ""
			if ts := r.GetStatus().GetCompletedAt(); ts != nil {
				age = relativeTime(ts.AsTime())
			}
			fmt.Fprintf(wf, "  %s\t%-30s\t%s\t%s\n", r.GetId(), title, age, msg)
		}
		wf.Flush()
	}

		return nil
	} // end doSummary

	if !*watch {
		return doSummary()
	}

	for {
		if useColorSum {
			fmt.Print("\033[2J\033[H")
		}
		fmt.Printf("runs summary — %s  (every %ds, Ctrl+C to stop)\n\n", time.Now().Format("15:04:05"), *interval)
		if err := doSummary(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

// parseSinceDuration parses a human duration like "1h", "24h", "7d".
// Standard time.ParseDuration handles h/m/s; "d" is handled manually.
// ── wait ──────────────────────────────────────────────────────────────────────

func runRunsWait(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs wait", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	timeout := fs.Duration("timeout", 0, "Max time to wait (e.g. 10m, 1h); 0 = no limit")
	quiet := fs.Bool("quiet", false, "Suppress all output; use exit code only")
	log := fs.Bool("log", false, "Stream log lines while waiting (like logs --follow)")
	onSuccess := fs.String("on-success", "", "Shell command to run on success (run ID is passed as $RUN_ID)")
	onFailure := fs.String("on-failure", "", "Shell command to run on failure (run ID is passed as $RUN_ID, message as $RUN_MESSAGE)")
	notify := fs.Bool("notify", false, "Send a macOS desktop notification when the run completes")
	anyFlag := fs.Bool("any", false, "Return as soon as any one run completes (default: wait for all)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs wait <id> [<id2> ...] [flags]\n\nBlock until run(s) reach a terminal phase.\nExits 0 if all succeed, 1 if any fail or are cancelled.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *lastRun && fs.NArg() == 0 {
		client0, err0 := newClient(*server)
		if err0 != nil {
			return err0
		}
		resp0, err0 := client0.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(resp0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		latestID := resp0.Msg.GetAgentRuns()[0].GetId()
		newArgs := []string{latestID, "--server=" + *server}
		if *timeout > 0 {
			newArgs = append(newArgs, "--timeout="+timeout.String())
		}
		if *quiet {
			newArgs = append(newArgs, "--quiet")
		}
		if *log {
			newArgs = append(newArgs, "--log")
		}
		if *onSuccess != "" {
			newArgs = append(newArgs, "--on-success="+*onSuccess)
		}
		if *onFailure != "" {
			newArgs = append(newArgs, "--on-failure="+*onFailure)
		}
		if *notify {
			newArgs = append(newArgs, "--notify")
		}
		return runRunsWait(newArgs)
	}

	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	// Multi-run support: fan out goroutines when multiple IDs given.
	if fs.NArg() > 1 {
		type result struct {
			id  string
			err error
		}
		ids := fs.Args()
		ch := make(chan result, len(ids))
		for _, rid := range ids {
			rid := rid
			go func() {
				subArgs := []string{rid, "--server=" + *server}
				if *timeout > 0 {
					subArgs = append(subArgs, "--timeout="+timeout.String())
				}
				if *quiet {
					subArgs = append(subArgs, "--quiet")
				}
				if *log {
					subArgs = append(subArgs, "--log")
				}
				if *onSuccess != "" {
					subArgs = append(subArgs, "--on-success="+*onSuccess)
				}
				if *onFailure != "" {
					subArgs = append(subArgs, "--on-failure="+*onFailure)
				}
				if *notify {
					subArgs = append(subArgs, "--notify")
				}
				ch <- result{id: rid, err: runRunsWait(subArgs)}
			}()
		}
		var firstErr error
		for i := 0; i < len(ids); i++ {
			r := <-ch
			if r.err != nil && firstErr == nil {
				firstErr = r.err
			}
			if *anyFlag {
				return firstErr
			}
		}
		return firstErr
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

	runHook := func(shellCmd string, extraEnv ...string) {
		if shellCmd == "" {
			return
		}
		cmd := exec.Command("sh", "-c", shellCmd)
		cmd.Env = append(os.Environ(), append([]string{"RUN_ID=" + id}, extraEnv...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && !*quiet {
			fmt.Fprintf(os.Stderr, "hook error: %v\n", err)
		}
	}

	sendNotify := func(title, body string) {
		if !*notify {
			return
		}
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		_ = exec.Command("osascript", "-e", script).Run()
	}

	switch phase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		if !*quiet {
			fmt.Printf("[%s] done\n", id)
			if url := getResp.Msg.GetStatus().GetPrUrl(); url != "" {
				fmt.Printf("PR: %s\n", url)
			}
		}
		sendNotify("UNCWORKS: run succeeded", id)
		runHook(*onSuccess)
		return nil
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		runHook(*onFailure, "RUN_MESSAGE="+msg)
		sendNotify("UNCWORKS: run failed", id)
		if finalPayload != "" {
			return fmt.Errorf("run %s failed: %s", id, finalPayload)
		}
		return fmt.Errorf("run %s failed", id)
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		runHook(*onFailure, "RUN_MESSAGE=cancelled")
		sendNotify("UNCWORKS: run cancelled", id)
		return fmt.Errorf("run %s was cancelled", id)
	default:
		return fmt.Errorf("run %s ended in unexpected phase: %s", id, phaseLabel(phase))
	}
}

// ── top ───────────────────────────────────────────────────────────────────────

func runRunsTop(args []string) error {
	fs := flag.NewFlagSet("runs top", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	interval := fs.Int("interval", 5, "Refresh interval in seconds")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	titleContains := fs.String("title-contains", "", "Filter runs by display name substring (case-insensitive)")
	phase := fs.String("phase", "", "Filter by phase: running, pending, waiting (default: all active)")
	limit := fs.Int("limit", 30, "Max runs to show per refresh")
	noColor := fs.Bool("no-color", false, "Disable ANSI color in output")
	oneShot := fs.Bool("one-shot", false, "Print once and exit (useful for scripting)")
	jsonOut := fs.Bool("json", false, "Output active runs as JSON (implies --one-shot)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs top [flags]\n\nLive view of active runs sorted by elapsed time.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *jsonOut {
		*oneShot = true
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	phaseFilter := strings.ToUpper(*phase)

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
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
		}
		return label
	}

	for {
		if !*jsonOut {
			if useColor {
				fmt.Print("\033[H\033[2J")
			}
			header := fmt.Sprintf("uncworks runs top — %s  (Ctrl+C to stop)", time.Now().Format("15:04:05"))
			if phaseFilter != "" {
				header += "  [phase:" + phaseFilter + "]"
			}
			fmt.Println(header + "\n")
		}

		var allActive []*apiv1.AgentRun
		cursor := ""
		for {
			req := &apiv1.ListAgentRunsRequest{
				Limit:         100,
				ProjectFilter: *project,
				FeatureFilter: *feature,
				TagFilter:     *tag,
				Cursor:        cursor,
			}
			resp, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(req))
			if apiErr != nil {
				fmt.Printf("error: %s\n", humanizeErr(apiErr))
				break
			}
			titleNeedle := strings.ToLower(*titleContains)
			for _, r := range resp.Msg.GetAgentRuns() {
				p := r.GetStatus().GetPhase()
				active := p == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING ||
					p == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING ||
					p == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
				if !active {
					continue
				}
				if phaseFilter != "" {
					label := phaseLabel(p)
					if !strings.EqualFold(label, phaseFilter) && !strings.HasPrefix(strings.ToUpper(label), phaseFilter) {
						continue
					}
				}
				if titleNeedle != "" && !strings.Contains(strings.ToLower(r.GetSpec().GetDisplayName()), titleNeedle) {
					continue
				}
				allActive = append(allActive, r)
			}
			cursor = resp.Msg.GetNextCursor()
			if cursor == "" {
				break
			}
		}

		// Sort by start time (oldest first = longest running at top).
		sort.Slice(allActive, func(i, j int) bool {
			ti := allActive[i].GetStatus().GetStartedAt()
			tj := allActive[j].GetStatus().GetStartedAt()
			if ti == nil {
				return false
			}
			if tj == nil {
				return true
			}
			return ti.AsTime().Before(tj.AsTime())
		})

		if *jsonOut {
			type topRun struct {
				ID      string `json:"id"`
				Phase   string `json:"phase"`
				Elapsed string `json:"elapsed"`
				Stage   string `json:"stage"`
				Title   string `json:"title"`
				Project string `json:"project"`
				Feature string `json:"feature"`
				Model   string `json:"model_tier"`
			}
			shown := allActive
			if *limit > 0 && len(shown) > *limit {
				shown = shown[:*limit]
			}
			var out []topRun
			for _, r := range shown {
				title := r.GetSpec().GetDisplayName()
				if title == "" {
					title = r.GetSpec().GetProject()
				}
				out = append(out, topRun{
					ID:      r.GetId(),
					Phase:   phaseLabel(r.GetStatus().GetPhase()),
					Elapsed: runDuration(r),
					Stage:   r.GetStatus().GetStage(),
					Title:   title,
					Project: r.GetSpec().GetProject(),
					Feature: r.GetSpec().GetFeature(),
					Model:   r.GetSpec().GetModelTier(),
				})
			}
			if out == nil {
				out = []topRun{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		if len(allActive) == 0 {
			fmt.Println("No active runs.")
		} else {
			var buf bytes.Buffer
			w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tELAPSED\tPHASE\tSTAGE\tTITLE")
			shown := allActive
			if *limit > 0 && len(shown) > *limit {
				shown = shown[:*limit]
			}
			for _, r := range shown {
				title := r.GetSpec().GetDisplayName()
				if title == "" {
					title = r.GetSpec().GetProject()
				}
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				phase := phaseLabel(r.GetStatus().GetPhase())
				elapsed := runDuration(r)
				stage := r.GetStatus().GetStage()
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.GetId(), elapsed, phase, stage, title)
			}
			_ = w.Flush()
			output := buf.String()
			if useColor {
				output = strings.NewReplacer(
					"RUNNING", "\033[32mRUNNING\033[0m",
					"PENDING", "\033[33mPENDING\033[0m",
					"WAITING", "\033[36mWAITING\033[0m",
				).Replace(output)
			}
			fmt.Print(output)
			fmt.Printf("\nTotal active: %d\n", len(allActive))
		}
		_ = colorPhase
		if *oneShot {
			return nil
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

// ── multi-tail ────────────────────────────────────────────────────────────────

func runRunsMultiTail(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs multi-tail", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	grep := fs.String("grep", "", "Only show lines matching this substring (case-insensitive)")
	noColor := fs.Bool("no-color", false, "Disable per-run ANSI color coding")
	allActive := fs.Bool("active", false, "Automatically discover and tail all active runs")
	project := fs.String("project", "", "Filter auto-discovered active runs by project (requires --active)")
	feature := fs.String("feature", "", "Filter auto-discovered active runs by feature (requires --active)")
	tag := fs.String("tag", "", "Filter auto-discovered active runs by tag (requires --active)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs multi-tail <id> [<id> ...] [flags]\n\nTail logs from multiple runs simultaneously.\nEach log line is prefixed with the run ID.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	runIDs := fs.Args()

	// Auto-discover active runs if --active flag is set.
	if *allActive {
		var cursor string
		for {
			listReq := connect.NewRequest(&apiv1.ListAgentRunsRequest{
				Limit:         100,
				Cursor:        cursor,
				ProjectFilter: *project,
				FeatureFilter: *feature,
				TagFilter:     *tag,
			})
			listResp, listErr := client.ListAgentRuns(context.Background(), listReq)
			if listErr != nil {
				return fmt.Errorf("%s", humanizeErr(listErr))
			}
			for _, r := range listResp.Msg.GetAgentRuns() {
				ph := r.GetStatus().GetPhase()
				if ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING ||
					ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING ||
					ph == apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
					runIDs = append(runIDs, r.GetId())
				}
			}
			cursor = listResp.Msg.GetNextCursor()
			if cursor == "" {
				break
			}
		}
	}

	if len(runIDs) == 0 {
		if *allActive {
			fmt.Println("No active runs to tail.")
		} else {
			fs.Usage()
			return fmt.Errorf("at least one run ID required")
		}
		return nil
	}

	grepNeedle := strings.ToLower(*grep)
	useColorMT := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))

	// ANSI colors for cycling per-run labels.
	colorCodes := []string{"\033[36m", "\033[32m", "\033[33m", "\033[35m", "\033[34m", "\033[96m", "\033[92m", "\033[93m"}
	colorReset := "\033[0m"

	type logLine struct {
		id   string
		line string
	}
	ch := make(chan logLine, 256)

	var wg sync.WaitGroup
	for _, id := range runIDs {
		wg.Add(1)
		go func(runID string) {
			defer wg.Done()
			stream, streamErr := client.WatchAgentRun(context.Background(), connect.NewRequest(&apiv1.WatchAgentRunRequest{Id: runID}))
			if streamErr != nil {
				ch <- logLine{id: runID, line: fmt.Sprintf("[error: %s]", humanizeErr(streamErr))}
				return
			}
			for stream.Receive() {
				ev := stream.Msg()
				switch ev.GetType() {
				case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
					payload := ev.GetPayload()
					if payload == "" {
						continue
					}
					for _, line := range strings.Split(strings.TrimRight(payload, "\n"), "\n") {
						if grepNeedle != "" && !strings.Contains(strings.ToLower(line), grepNeedle) {
							continue
						}
						ch <- logLine{id: runID, line: line}
					}
				case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
					ch <- logLine{id: runID, line: fmt.Sprintf("[phase: %s]", ev.GetPayload())}
				case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
					ch <- logLine{id: runID, line: fmt.Sprintf("[completed: %s]", ev.GetPayload())}
				}
			}
		}(id)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Assign a short label (last 6 chars of ID) and a color per run.
	labels := make(map[string]string, len(runIDs))
	colors := make(map[string]string, len(runIDs))
	for i, id := range runIDs {
		label := id
		if len(id) > 6 {
			label = id[len(id)-6:]
		}
		labels[id] = label
		if useColorMT {
			colors[id] = colorCodes[i%len(colorCodes)]
		}
	}

	for ll := range ch {
		label := labels[ll.id]
		if useColorMT {
			fmt.Printf("%s[%s]%s %s\n", colors[ll.id], label, colorReset, ll.line)
		} else {
			fmt.Printf("[%s] %s\n", label, ll.line)
		}
	}
	return nil
}

// ── batch ─────────────────────────────────────────────────────────────────────

func runRunsBatch(args []string) error {
	fs := flag.NewFlagSet("runs batch", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	dryRun := fs.Bool("dry-run", false, "Print what would be submitted without creating runs")
	wait := fs.Bool("wait", false, "Wait for all runs to complete before exiting")
	outputIDs := fs.Bool("output-ids", false, "Print only run IDs (one per line) for scripting")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks runs batch <file.json> [flags]

Submit multiple runs from a JSON file. The file should contain a JSON array of run specs.

Example file:
  [
    {"prompt": "task 1", "project": "myproj", "model_tier": "deepseek-v3.2"},
    {"prompt": "task 2", "project": "myproj", "tags": ["ci"]}
  ]

Supported fields: prompt, repo, branch, name, project, feature, model_tier,
  auto_push, auto_pr, parent_run_id, tags, env_vars.

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("JSON file argument required (use '-' to read from stdin)")
	}

	var raw []byte
	var err error
	if fs.Arg(0) == "-" {
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		raw, err = os.ReadFile(fs.Arg(0))
		if err != nil {
			return fmt.Errorf("reading %s: %w", fs.Arg(0), err)
		}
	}

	type batchSpec struct {
		Prompt      string            `json:"prompt"`
		Repo        string            `json:"repo"`
		Branch      string            `json:"branch"`
		Name        string            `json:"name"`
		Project     string            `json:"project"`
		Feature     string            `json:"feature"`
		ModelTier   string            `json:"model_tier"`
		AutoPush    bool              `json:"auto_push"`
		AutoPR      bool              `json:"auto_pr"`
		ParentRunID string            `json:"parent_run_id"`
		Tags        []string          `json:"tags"`
		EnvVars     map[string]string `json:"env_vars"`
	}

	var specs []batchSpec
	if err := json.Unmarshal(raw, &specs); err != nil {
		return fmt.Errorf("parsing %s: %w", fs.Arg(0), err)
	}
	if len(specs) == 0 {
		return fmt.Errorf("no run specs found in %s", fs.Arg(0))
	}

	// Auto-detect repo from git if not specified in any spec.
	defaultRepo := ""
	if out, gitErr := exec.Command("git", "remote", "get-url", "origin").Output(); gitErr == nil {
		defaultRepo = strings.TrimSpace(string(out))
	}
	defaultBranch := "main"
	if out, gitErr := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); gitErr == nil {
		b := strings.TrimSpace(string(out))
		if b != "" && b != "HEAD" {
			defaultBranch = b
		}
	}

	// Apply config defaults to fields not set in each spec.
	defaultModelTier, defaultProject, defaultFeature := "", "", ""
	if cfg, cfgErr := loadConfig(); cfgErr == nil {
		defaultModelTier = cfg.DefaultModelTier
		defaultProject = cfg.DefaultProject
		defaultFeature = cfg.DefaultFeature
	}

	if *dryRun {
		fmt.Printf("Dry run — would submit %d run(s):\n", len(specs))
		for i, s := range specs {
			repo := s.Repo
			if repo == "" {
				repo = defaultRepo
			}
			branch := s.Branch
			if branch == "" {
				branch = defaultBranch
			}
			prompt := s.Prompt
			if len(prompt) > 80 {
				prompt = prompt[:77] + "..."
			}
			fmt.Printf("  %d. [%s@%s] %s\n", i+1, repo, branch, prompt)
		}
		return nil
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	var createdIDs []string
	for i, s := range specs {
		repo := s.Repo
		if repo == "" {
			repo = defaultRepo
		}
		if repo == "" {
			return fmt.Errorf("spec %d: repo is required (no git remote detected)", i+1)
		}
		if s.Prompt == "" {
			return fmt.Errorf("spec %d: prompt is required", i+1)
		}
		branch := s.Branch
		if branch == "" {
			branch = defaultBranch
		}

		model := s.ModelTier
		if model == "" {
			model = defaultModelTier
		}
		proj := s.Project
		if proj == "" {
			proj = defaultProject
		}
		feat := s.Feature
		if feat == "" {
			feat = defaultFeature
		}
		spec := &apiv1.AgentRunSpec{
			Backend:     apiv1.Backend_BACKEND_POD,
			Repos:       []*apiv1.Repository{{Url: repo, Branch: branch}},
			Prompt:      s.Prompt,
			DisplayName: s.Name,
			Project:     proj,
			Feature:     feat,
			ModelTier:   model,
			AutoPush:    s.AutoPush || s.AutoPR,
			AutoPr:      s.AutoPR,
			Tags:        s.Tags,
			ParentRunId: s.ParentRunID,
			EnvVars:     s.EnvVars,
		}

		resp, createErr := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{Spec: spec}))
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "  spec %d: %s\n", i+1, humanizeErr(createErr))
			continue
		}
		id := resp.Msg.GetAgentRun().GetId()
		createdIDs = append(createdIDs, id)
		if *outputIDs {
			fmt.Println(id)
		} else {
			fmt.Printf("  created: %s  (%s)\n", id, s.Prompt[:min(len(s.Prompt), 60)])
		}
	}

	if !*outputIDs {
		fmt.Printf("Submitted %d/%d run(s).\n", len(createdIDs), len(specs))
	}

	if *wait && len(createdIDs) > 0 {
		fmt.Printf("Waiting for %d run(s) to complete...\n", len(createdIDs))
		var wg sync.WaitGroup
		results := make([]error, len(createdIDs))
		for i, id := range createdIDs {
			wg.Add(1)
			go func(idx int, runID string) {
				defer wg.Done()
				results[idx] = runRunsWait([]string{runID, "--quiet", "--server=" + *server})
			}(i, id)
		}
		wg.Wait()
		failed := 0
		for _, e := range results {
			if e != nil {
				failed++
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d/%d run(s) failed", failed, len(createdIDs))
		}
		fmt.Printf("All %d run(s) completed successfully.\n", len(createdIDs))
	}

	return nil
}

// ── group ─────────────────────────────────────────────────────────────────────

func runRunsGroup(args []string) error {
	fs := flag.NewFlagSet("runs group", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	by := fs.String("by", "project", "Group dimension: project, feature, tag, or model")
	since := fs.String("since", "", "Filter to runs created within this window (e.g. 1h, 24h, 7d)")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, etc.)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	limit := fs.Int("limit", 200, "Max total runs to fetch (0 = no limit)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	titleWidth := fs.Int("title-width", 36, "Max characters for title column")
	jsonOut := fs.Bool("json", false, "Output grouped runs as JSON object")
	countOnly := fs.Bool("count-only", false, "Show only group names and run counts (no individual runs)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs group [flags]\n\nShow runs organized into groups.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *by {
	case "project", "feature", "tag", "model":
	default:
		return fmt.Errorf("--by must be one of: project, feature, tag, model")
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

	listReq := &apiv1.ListAgentRunsRequest{
		Limit:         100,
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
		case "WAITING":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
		case "CANCELLED":
			listReq.PhaseFilter = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		}
	}

	// group key → list of runs (preserve insertion order via separate keys slice)
	groups := map[string][]*apiv1.AgentRun{}
	var groupOrder []string
	total := 0
	cursor := ""
	for {
		listReq.Cursor = cursor
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(listReq))
		if err != nil {
			// On connection error mid-pagination, render with data collected so far.
			break
		}
		stop := false
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts == nil || !ts.AsTime().After(sinceTime) {
					stop = true
					break
				}
			}
			if *limit > 0 && total >= *limit {
				stop = true
				break
			}
			var keys []string
			switch *by {
			case "project":
				k := r.GetSpec().GetProject()
				if k == "" {
					k = "(no project)"
				}
				keys = []string{k}
			case "feature":
				k := r.GetSpec().GetFeature()
				if k == "" {
					k = "(no feature)"
				}
				keys = []string{k}
			case "tag":
				tags := r.GetSpec().GetTags()
				if len(tags) == 0 {
					keys = []string{"(untagged)"}
				} else {
					keys = tags
				}
			case "model":
				k := r.GetSpec().GetModelTier()
				if k == "" {
					k = "(no model)"
				}
				keys = []string{k}
			}
			for _, k := range keys {
				if _, ok := groups[k]; !ok {
					groupOrder = append(groupOrder, k)
				}
				groups[k] = append(groups[k], r)
			}
			total++
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || stop {
			break
		}
	}

	if len(groups) == 0 {
		if *jsonOut {
			fmt.Println(`{"groups":[]}`)
			return nil
		}
		fmt.Println("No runs found.")
		return nil
	}

	if *jsonOut {
		type jsonRun struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Phase    string `json:"phase"`
			Duration string `json:"duration"`
			Age      string `json:"age"`
		}
		type jsonGroup struct {
			Key  string    `json:"key"`
			Runs []jsonRun `json:"runs"`
		}
		type groupsOutput struct {
			Groups []jsonGroup `json:"groups"`
			Total  int         `json:"total"`
		}
		out := groupsOutput{Total: total}
		for _, key := range groupOrder {
			g := jsonGroup{Key: key}
			for _, r := range groups[key] {
				title := r.GetSpec().GetDisplayName()
				if title == "" {
					title = r.GetSpec().GetProject()
				}
				age := ""
				if ts := r.GetCreatedAt(); ts != nil {
					age = relativeTime(ts.AsTime())
				}
				g.Runs = append(g.Runs, jsonRun{
					ID:       r.GetId(),
					Title:    title,
					Phase:    phaseLabel(r.GetStatus().GetPhase()),
					Duration: runDuration(r),
					Age:      age,
				})
			}
			out.Groups = append(out.Groups, g)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if *countOnly {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "GROUP\tCOUNT\n")
		for _, key := range groupOrder {
			fmt.Fprintf(w, "%s\t%d\n", key, len(groups[key]))
		}
		w.Flush()
		fmt.Printf("\nTotal: %d run(s) in %d group(s)\n", total, len(groupOrder))
		return nil
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	tw := *titleWidth
	if tw < 10 {
		tw = 10
	}

	phaseColor := func(label string) string {
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

	for i, key := range groupOrder {
		runs := groups[key]
		if i > 0 {
			fmt.Println()
		}
		header := fmt.Sprintf("── %s (%d run", key, len(runs))
		if len(runs) != 1 {
			header += "s"
		}
		header += ")"
		if useColor {
			fmt.Printf("\033[1m%s\033[0m\n", header)
		} else {
			fmt.Println(header)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, r := range runs {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			if len(title) > tw {
				title = title[:tw-3] + "..."
			}
			ph := phaseLabel(r.GetStatus().GetPhase())
			dur := runDuration(r)
			age := ""
			if ts := r.GetCreatedAt(); ts != nil {
				age = relativeTime(ts.AsTime())
			}
			fmt.Fprintf(w, "  %s\t%-*s\t%s\t%s\t%s\n",
				r.GetId(), tw, title, phaseColor(ph), dur, age)
		}
		w.Flush()
	}
	fmt.Printf("\nTotal: %d run(s) in %d group(s)\n", total, len(groupOrder))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// ── search ────────────────────────────────────────────────────────────────────

func runRunsSearch(args []string) error {
	fs := flag.NewFlagSet("runs search", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	phase := fs.String("phase", "", "Filter by phase (RUNNING, DONE, FAILED, etc.)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	since := fs.String("since", "7d", "Time window to search (e.g. 1h, 24h, 7d)")
	limit := fs.Int("limit", 50, "Max number of matching runs to show")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	jsonOut := fs.Bool("json", false, "Output matching runs as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs search <term> [flags]\n\nSearch runs by prompt text, title, or project name.\nThe search term is matched case-insensitively against the run prompt, display name, and project.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("search term required")
	}
	query := strings.ToLower(strings.Join(fs.Args(), " "))

	d, err := parseSinceDuration(*since)
	if err != nil {
		return fmt.Errorf("--since %q: %w", *since, err)
	}
	sinceTime := time.Now().Add(-d)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))

	var phaseF apiv1.AgentRunPhase
	if *phase != "" {
		switch strings.ToUpper(*phase) {
		case "RUNNING":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
		case "DONE", "SUCCEEDED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
		case "FAILED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
		case "PENDING":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
		case "CANCELLED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		}
	}

	var matches []*apiv1.AgentRun
	cursor := ""
	for {
		req := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			PhaseFilter:   phaseF,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}
		resp, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(req))
		if apiErr != nil {
			break
		}
		passedSince := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts != nil && !ts.AsTime().After(sinceTime) {
				passedSince = true
				continue
			}
			prompt := strings.ToLower(r.GetSpec().GetPrompt())
			title := strings.ToLower(r.GetSpec().GetDisplayName())
			proj := strings.ToLower(r.GetSpec().GetProject())
			if strings.Contains(prompt, query) || strings.Contains(title, query) || strings.Contains(proj, query) {
				matches = append(matches, r)
			}
		}
		if len(matches) >= *limit {
			matches = matches[:*limit]
			break
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || passedSince {
			break
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		type jsonRun struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Project string `json:"project"`
			Phase   string `json:"phase"`
			Created string `json:"created_at"`
		}
		var out []jsonRun
		for _, r := range matches {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			out = append(out, jsonRun{
				ID:      r.GetId(),
				Title:   title,
				Project: r.GetSpec().GetProject(),
				Phase:   phaseLabel(r.GetStatus().GetPhase()),
				Created: r.GetCreatedAt().AsTime().Format(time.RFC3339),
			})
		}
		return enc.Encode(out)
	}

	if len(matches) == 0 {
		fmt.Printf("No runs found matching %q in the last %s.\n", query, *since)
		return nil
	}

	fmt.Printf("Found %d run(s) matching %q:\n\n", len(matches), query)
	colorPhase := func(label string) string {
		if !useColor {
			return label
		}
		switch label {
		case "RUNNING":
			return "\033[32m" + label + "\033[0m"
		case "PENDING":
			return "\033[33m" + label + "\033[0m"
		case "FAILED":
			return "\033[31m" + label + "\033[0m"
		case "DONE":
			return "\033[90m" + label + "\033[0m"
		case "CANCELLED":
			return "\033[35m" + label + "\033[0m"
		}
		return label
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPHASE\tPROJECT\tAGE\tTITLE")
	for _, r := range matches {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if len(title) > 45 {
			title = title[:42] + "..."
		}
		age := ""
		if ts := r.GetCreatedAt(); ts != nil {
			age = relativeTime(ts.AsTime())
		}
		phase := colorPhase(phaseLabel(r.GetStatus().GetPhase()))
		proj := r.GetSpec().GetProject()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.GetId(), phase, proj, age, title)
	}
	w.Flush()
	return nil
}

// ── timeline ──────────────────────────────────────────────────────────────────

func runRunsTimeline(args []string) error {
	fs := flag.NewFlagSet("runs timeline", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	since := fs.String("since", "24h", "Time window to show (e.g. 1h, 24h, 7d)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	limit := fs.Int("limit", 100, "Max runs to show")
	phase := fs.String("phase", "", "Filter by phase (DONE, FAILED, CANCELLED; default: all terminal)")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	jsonOut := fs.Bool("json", false, "Output as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs timeline [flags]\n\nShow a chronological view of completed runs with durations.\n\nFlags:")
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

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))

	var phaseF apiv1.AgentRunPhase
	terminalOnly := true
	if *phase != "" {
		switch strings.ToUpper(*phase) {
		case "DONE", "SUCCEEDED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
		case "FAILED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
		case "CANCELLED":
			phaseF = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
		default:
			return fmt.Errorf("--phase must be DONE, FAILED, or CANCELLED")
		}
		terminalOnly = false
	}

	var runs []*apiv1.AgentRun
	cursor := ""
	for {
		req := &apiv1.ListAgentRunsRequest{
			Limit:         100,
			PhaseFilter:   phaseF,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}
		resp, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(req))
		if apiErr != nil {
			break
		}
		for _, r := range resp.Msg.GetAgentRuns() {
			completedAt := r.GetStatus().GetCompletedAt()
			if completedAt == nil || !completedAt.AsTime().After(sinceTime) {
				continue
			}
			ph := r.GetStatus().GetPhase()
			if terminalOnly {
				if ph != apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED &&
					ph != apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED &&
					ph != apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
					continue
				}
			}
			runs = append(runs, r)
		}
		if len(runs) >= *limit {
			runs = runs[:*limit]
			break
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	// Sort by completion time ascending (oldest first = chronological).
	sort.Slice(runs, func(i, j int) bool {
		ti := runs[i].GetStatus().GetCompletedAt()
		tj := runs[j].GetStatus().GetCompletedAt()
		if ti == nil {
			return true
		}
		if tj == nil {
			return false
		}
		return ti.AsTime().Before(tj.AsTime())
	})

	if len(runs) == 0 {
		if *jsonOut {
			fmt.Println("[]")
			return nil
		}
		fmt.Printf("No completed runs in the last %s.\n", *since)
		return nil
	}

	if *jsonOut {
		type timelineRun struct {
			ID          string `json:"id"`
			Phase       string `json:"phase"`
			Duration    string `json:"duration"`
			Project     string `json:"project"`
			Feature     string `json:"feature"`
			Title       string `json:"title"`
			CompletedAt string `json:"completed_at,omitempty"`
		}
		var out []timelineRun
		for _, r := range runs {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			completedStr := ""
			if ts := r.GetStatus().GetCompletedAt(); ts != nil {
				completedStr = ts.AsTime().Format(time.RFC3339)
			}
			out = append(out, timelineRun{
				ID:          r.GetId(),
				Phase:       phaseLabel(r.GetStatus().GetPhase()),
				Duration:    runDuration(r),
				Project:     r.GetSpec().GetProject(),
				Feature:     r.GetSpec().GetFeature(),
				Title:       title,
				CompletedAt: completedStr,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	colorPhase := func(label string) string {
		if !useColor {
			return label
		}
		switch label {
		case "DONE":
			return "\033[32m" + label + "\033[0m"
		case "FAILED":
			return "\033[31m" + label + "\033[0m"
		case "CANCELLED":
			return "\033[35m" + label + "\033[0m"
		}
		return label
	}

	fmt.Printf("Timeline: %d completed run(s) in the last %s\n\n", len(runs), *since)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMPLETED\tID\tPHASE\tDURATION\tPROJECT\tTITLE")
	for _, r := range runs {
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		completedStr := "—"
		if ts := r.GetStatus().GetCompletedAt(); ts != nil {
			completedStr = relativeTime(ts.AsTime())
		}
		ph := colorPhase(phaseLabel(r.GetStatus().GetPhase()))
		dur := runDuration(r)
		proj := r.GetSpec().GetProject()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", completedStr, r.GetId(), ph, dur, proj, title)
	}
	w.Flush()
	return nil
}

// ── compare ───────────────────────────────────────────────────────────────────

func runRunsCompare(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs compare", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	jsonOut := fs.Bool("json", false, "Output comparison as JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs compare <id1> <id2> [flags]\n\nShow a side-by-side field comparison of two runs.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("exactly two run IDs required")
	}
	id1, id2 := fs.Arg(0), fs.Arg(1)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	r1resp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id1}))
	if err != nil {
		return fmt.Errorf("fetching %s: %s", id1, humanizeErr(err))
	}
	r2resp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id2}))
	if err != nil {
		return fmt.Errorf("fetching %s: %s", id2, humanizeErr(err))
	}
	r1, r2 := r1resp.Msg, r2resp.Msg

	getTitle := func(r *apiv1.AgentRun) string {
		t := r.GetSpec().GetDisplayName()
		if t == "" {
			t = r.GetSpec().GetProject()
		}
		return t
	}
	getAge := func(r *apiv1.AgentRun) string {
		if ts := r.GetCreatedAt(); ts != nil {
			return relativeTime(ts.AsTime())
		}
		return "—"
	}
	getDur := func(r *apiv1.AgentRun) string { return runDuration(r) }
	getBranch := func(r *apiv1.AgentRun) string {
		if repos := r.GetSpec().GetRepos(); len(repos) > 0 {
			return repos[0].GetBranch()
		}
		return "—"
	}

	if *jsonOut {
		type runSummary struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Phase    string `json:"phase"`
			Duration string `json:"duration"`
			Age      string `json:"age"`
			Model    string `json:"model"`
			Project  string `json:"project"`
			Feature  string `json:"feature"`
			Branch   string `json:"branch"`
			PRUrl    string `json:"pr_url,omitempty"`
		}
		out := struct {
			A runSummary `json:"a"`
			B runSummary `json:"b"`
		}{
			A: runSummary{
				ID:       r1.GetId(),
				Title:    getTitle(r1),
				Phase:    phaseLabel(r1.GetStatus().GetPhase()),
				Duration: getDur(r1),
				Age:      getAge(r1),
				Model:    r1.GetSpec().GetModelTier(),
				Project:  r1.GetSpec().GetProject(),
				Feature:  r1.GetSpec().GetFeature(),
				Branch:   getBranch(r1),
				PRUrl:    r1.GetStatus().GetPrUrl(),
			},
			B: runSummary{
				ID:       r2.GetId(),
				Title:    getTitle(r2),
				Phase:    phaseLabel(r2.GetStatus().GetPhase()),
				Duration: getDur(r2),
				Age:      getAge(r2),
				Model:    r2.GetSpec().GetModelTier(),
				Project:  r2.GetSpec().GetProject(),
				Feature:  r2.GetSpec().GetFeature(),
				Branch:   getBranch(r2),
				PRUrl:    r2.GetStatus().GetPrUrl(),
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	type row struct{ field, a, b string }
	rows := []row{
		{"ID", r1.GetId(), r2.GetId()},
		{"Title", getTitle(r1), getTitle(r2)},
		{"Phase", phaseLabel(r1.GetStatus().GetPhase()), phaseLabel(r2.GetStatus().GetPhase())},
		{"Duration", getDur(r1), getDur(r2)},
		{"Age", getAge(r1), getAge(r2)},
		{"Model", r1.GetSpec().GetModelTier(), r2.GetSpec().GetModelTier()},
		{"Project", r1.GetSpec().GetProject(), r2.GetSpec().GetProject()},
		{"Feature", r1.GetSpec().GetFeature(), r2.GetSpec().GetFeature()},
		{"Branch", getBranch(r1), getBranch(r2)},
		{"PR URL", r1.GetStatus().GetPrUrl(), r2.GetStatus().GetPrUrl()},
		{"Message", r1.GetStatus().GetMessage(), r2.GetStatus().GetMessage()},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "FIELD\t%-22s\t%-22s\n", id1, id2)
	fmt.Fprintf(w, "─────\t%s\t%s\n", strings.Repeat("─", 22), strings.Repeat("─", 22))
	for _, r := range rows {
		if r.a == "" && r.b == "" {
			continue
		}
		a, b := r.a, r.b
		if len(a) > 40 {
			a = a[:37] + "..."
		}
		if len(b) > 40 {
			b = b[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.field, a, b)
	}
	w.Flush()
	return nil
}

// ── alias ────────────────────────────────────────────────────────────────────

func runRunsAlias(args []string) error {
	type aliasEntry struct{ alias, expandsTo string }
	flagAliases := []aliasEntry{
		{"runs list --running", "runs list --phase RUNNING"},
		{"runs list --failed", "runs list --phase FAILED"},
		{"runs list --pending", "runs list --phase PENDING"},
		{"runs list --waiting", "runs list --phase WAITING"},
		{"runs list --done", "runs list --phase DONE"},
		{"runs list --cancelled", "runs list --phase CANCELLED"},
		{"runs list --active", "runs list (RUNNING + PENDING + WAITING)"},
		{"runs list --recent", "runs list --since 24h"},
		{"runs list --all", "runs list (all pages, no limit)"},
		{"runs list --title <text>", "runs list --title-contains <text>"},
		{"run --model <tier>", "run --model-tier <tier>"},
		{"runs retry --model <tier>", "runs retry --model-tier <tier>"},
	}
	cmdAliases := []aliasEntry{
		{"uncworks jobs", "uncworks runs list --active"},
		{"uncworks top", "uncworks runs top"},
		{"uncworks watch", "uncworks runs watch"},
		{"uncworks last", "uncworks runs get --last"},
		{"uncworks tail", "uncworks runs tail --last"},
		{"uncworks wait", "uncworks runs wait --last"},
		{"uncworks summary", "uncworks runs summary"},
		{"uncworks score", "uncworks runs score"},
		{"uncworks tally", "uncworks runs tally"},
		{"uncworks stats", "uncworks runs stats"},
		{"uncworks kill <id>", "uncworks cancel <id>"},
		{"runs show <id>", "runs get <id>"},
		{"runs rerun <id>", "runs retry <id>"},
		{"runs copy <id>", "runs retry <id>"},
		{"runs duplicate <id>", "runs retry <id>"},
		{"runs open-pr <id>", "runs open <id>"},
		{"runs pr <id>", "runs open <id>"},
		{"runs kill <id>", "runs cancel <id>"},
		{"runs kill-all", "runs cancel-all"},
		{"runs multi-logs", "runs multi-tail"},
		{"runs aliases", "runs alias"},
		{"runs retry-last", "runs retry --last"},
		{"runs tail-last", "runs tail --last"},
	}

	fmt.Println("Flag aliases:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, a := range flagAliases {
		fmt.Fprintf(w, "  %s\t→  %s\n", a.alias, a.expandsTo)
	}
	w.Flush()

	fmt.Println("\nCommand aliases:")
	w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, a := range cmdAliases {
		fmt.Fprintf(w2, "  %s\t→  %s\n", a.alias, a.expandsTo)
	}
	w2.Flush()
	return nil
}

// ── env ──────────────────────────────────────────────────────────────────────

func runRunsEnv(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("runs env", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	export := fs.Bool("export", false, "Output as shell export statements (eval-friendly)")
	lastRun := fs.Bool("last", false, "Use the most recent run (auto-detect ID)")
	jsonOut := fs.Bool("json", false, "Output as JSON object")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs env <id> [flags]\n\nShow environment variables configured for a run.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	var id string
	if *lastRun {
		resp0, err0 := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if err0 != nil {
			return fmt.Errorf("%s", humanizeErr(err0))
		}
		if len(resp0.Msg.GetAgentRuns()) == 0 {
			return fmt.Errorf("no runs found")
		}
		id = resp0.Msg.GetAgentRuns()[0].GetId()
	} else {
		if fs.NArg() != 1 {
			fs.Usage()
			return fmt.Errorf("run ID argument required")
		}
		id = fs.Arg(0)
	}

	resp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	envVars := resp.Msg.GetSpec().GetEnvVars()
	if *jsonOut {
		if envVars == nil {
			envVars = map[string]string{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(envVars)
	}
	if len(envVars) == 0 {
		fmt.Println("(no env vars set)")
		return nil
	}

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if *export {
			fmt.Printf("export %s=%q\n", k, envVars[k])
		} else {
			fmt.Printf("%s=%s\n", k, envVars[k])
		}
	}
	return nil
}

// ── slow ─────────────────────────────────────────────────────────────────────

func runRunsSlow(args []string) error {
	fs := flag.NewFlagSet("runs slow", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	limit := fs.Int("limit", 10, "Number of slowest runs to show")
	since := fs.String("since", "7d", "Time window to search (e.g. 1h, 24h, 7d)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	jsonOut := fs.Bool("json", false, "Output as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs slow [flags]\n\nShow the slowest completed runs sorted by duration.\n\nFlags:")
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

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	var runs []*apiv1.AgentRun
	cursor := ""
	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		}))
		if err != nil {
			break
		}
		passedSince := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil {
				continue
			}
			if !ts.AsTime().After(sinceTime) {
				passedSince = true
				continue
			}
			if r.GetStatus().GetStartedAt() == nil || r.GetStatus().GetCompletedAt() == nil {
				continue
			}
			runs = append(runs, r)
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || passedSince {
			break
		}
	}

	getDurationSecs := func(r *apiv1.AgentRun) float64 {
		return r.GetStatus().GetCompletedAt().AsTime().Sub(r.GetStatus().GetStartedAt().AsTime()).Seconds()
	}

	sort.Slice(runs, func(i, j int) bool {
		return getDurationSecs(runs[i]) > getDurationSecs(runs[j])
	})

	if *limit > 0 && len(runs) > *limit {
		runs = runs[:*limit]
	}

	if len(runs) == 0 {
		if *jsonOut {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No completed runs found.")
		return nil
	}

	if *jsonOut {
		type slowRun struct {
			ID       string  `json:"id"`
			Phase    string  `json:"phase"`
			Duration string  `json:"duration"`
			Seconds  float64 `json:"duration_seconds"`
			Project  string  `json:"project"`
			Feature  string  `json:"feature"`
			Title    string  `json:"title"`
		}
		var out []slowRun
		for _, r := range runs {
			title := r.GetSpec().GetDisplayName()
			if title == "" {
				title = r.GetSpec().GetProject()
			}
			out = append(out, slowRun{
				ID:       r.GetId(),
				Phase:    phaseLabel(r.GetStatus().GetPhase()),
				Duration: runDuration(r),
				Seconds:  getDurationSecs(r),
				Project:  r.GetSpec().GetProject(),
				Feature:  r.GetSpec().GetFeature(),
				Title:    title,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	_ = !*noColor && term.IsTerminal(int(os.Stdout.Fd())) // reserved for future coloring
	fmt.Printf("Slowest %d run(s) in the last %s:\n\n", len(runs), *since)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DURATION\tID\tPHASE\tPROJECT\tTITLE")
	for _, r := range runs {
		dur := runDuration(r)
		title := r.GetSpec().GetDisplayName()
		if title == "" {
			title = r.GetSpec().GetProject()
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		ph := phaseLabel(r.GetStatus().GetPhase())
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", dur, r.GetId(), ph, r.GetSpec().GetProject(), title)
	}
	w.Flush()
	return nil
}

// ── score ─────────────────────────────────────────────────────────────────────

func runRunsScore(args []string) error {
	fs := flag.NewFlagSet("runs score", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	includeArchived := fs.Bool("include-archived", false, "Include archived runs in the score calculation")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs score [flags]\n\nShow success rate across multiple time windows (1h, 24h, 7d, 30d).\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	type windowResult struct {
		Window  string  `json:"window"`
		Total   int     `json:"total"`
		Done    int     `json:"done"`
		Failed  int     `json:"failed"`
		Rate    float64 `json:"success_rate"`
	}

	windows := []struct {
		label string
		dur   time.Duration
	}{
		{"1h", time.Hour},
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
	}

	// Fetch once for the largest window (30d) and compute all smaller windows
	// from the same dataset — avoids 4 separate pagination loops.
	longestCutoff := time.Now().Add(-windows[len(windows)-1].dur)
	type runRecord struct {
		createdAt time.Time
		phase     apiv1.AgentRunPhase
	}
	var allRuns []runRecord
	cursor := ""
	for {
		listReq := connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		})
		if *includeArchived {
			listReq.Header().Set("X-Include-Archived", "true")
		}
		resp, err2 := c.ListAgentRuns(context.Background(), listReq)
		if err2 != nil {
			break
		}
		passedCutoff := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil {
				continue
			}
			if !ts.AsTime().After(longestCutoff) {
				passedCutoff = true
				continue
			}
			allRuns = append(allRuns, runRecord{
				createdAt: ts.AsTime(),
				phase:     r.GetStatus().GetPhase(),
			})
		}
		cursor = resp.Msg.GetNextCursor()
		if cursor == "" || (!*includeArchived && passedCutoff) {
			break
		}
	}

	now := time.Now()
	var results []windowResult
	for _, win := range windows {
		cutoff := now.Add(-win.dur)
		done, failed, total := 0, 0, 0
		for _, r := range allRuns {
			if !r.createdAt.After(cutoff) {
				continue
			}
			total++
			switch r.phase {
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
				done++
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
				failed++
			}
		}
		rate := 0.0
		if done+failed > 0 {
			rate = float64(done) / float64(done+failed) * 100
		}
		results = append(results, windowResult{Window: win.label, Total: total, Done: done, Failed: failed, Rate: rate})
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	useColor := term.IsTerminal(int(os.Stdout.Fd()))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "WINDOW\tTOTAL\tDONE\tFAILED\tSUCCESS RATE")
	for _, r := range results {
		rateStr := "—"
		if r.Done+r.Failed > 0 {
			rateStr = fmt.Sprintf("%.1f%%", r.Rate)
			if useColor {
				if r.Rate >= 80 {
					rateStr = "\033[32m" + rateStr + "\033[0m"
				} else if r.Rate >= 50 {
					rateStr = "\033[33m" + rateStr + "\033[0m"
				} else {
					rateStr = "\033[31m" + rateStr + "\033[0m"
				}
			}
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n", r.Window, r.Total, r.Done, r.Failed, rateStr)
	}
	return w.Flush()
}

// ── tally ─────────────────────────────────────────────────────────────────────

func runRunsTally(args []string) error {
	fs := flag.NewFlagSet("runs tally", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	days := fs.Int("days", 14, "Number of past days to show")
	project := fs.String("project", "", "Filter by project name")
	feature := fs.String("feature", "", "Filter by feature name")
	tag := fs.String("tag", "", "Filter by tag")
	includeArchived := fs.Bool("include-archived", false, "Include archived runs in the counts")
	noColor := fs.Bool("no-color", false, "Disable ANSI color")
	jsonOut := fs.Bool("json", false, "Output as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs tally [flags]\n\nShow daily run counts for the past N days.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *days < 1 {
		return fmt.Errorf("--days must be >= 1")
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -*days)
	type dayBucket struct {
		Date   string `json:"date"`
		Total  int    `json:"total"`
		Done   int    `json:"done"`
		Failed int    `json:"failed"`
	}

	// Build a map keyed by YYYY-MM-DD in local time.
	buckets := map[string]*dayBucket{}
	now := time.Now()
	for i := 0; i < *days; i++ {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		buckets[d] = &dayBucket{Date: d}
	}

	cursor := ""
	for {
		listReq := connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			FeatureFilter: *feature,
			TagFilter:     *tag,
			Cursor:        cursor,
		})
		if *includeArchived {
			listReq.Header().Set("X-Include-Archived", "true")
		}
		resp, err2 := c.ListAgentRuns(context.Background(), listReq)
		if err2 != nil {
			break
		}
		passedCutoff := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil {
				continue
			}
			t := ts.AsTime().Local()
			if t.Before(cutoff) {
				passedCutoff = true
				continue
			}
			d := t.Format("2006-01-02")
			b, ok := buckets[d]
			if !ok {
				continue
			}
			b.Total++
			switch r.GetStatus().GetPhase() {
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
				b.Done++
			case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
				b.Failed++
			}
		}
		cursor = resp.Msg.GetNextCursor()
		// When not including archived runs, stop once we've passed the cutoff date
		// (all remaining pages will be older). With --include-archived, archived runs
		// appear after non-archived in the API response, so we must paginate fully.
		if cursor == "" || (!*includeArchived && passedCutoff) {
			break
		}
	}

	// Sort dates descending.
	var sorted []string
	for d := range buckets {
		sorted = append(sorted, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(sorted)))

	if *jsonOut {
		var out []dayBucket
		for _, d := range sorted {
			out = append(out, *buckets[d])
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	useColor := !*noColor && term.IsTerminal(int(os.Stdout.Fd()))
	maxTotal := 0
	for _, b := range buckets {
		if b.Total > maxTotal {
			maxTotal = b.Total
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTOTAL\tDONE\tFAILED\tBAR")
	for _, d := range sorted {
		b := buckets[d]
		barLen := 0
		if maxTotal > 0 {
			barLen = int(float64(b.Total) / float64(maxTotal) * 20)
		}
		bar := strings.Repeat("█", barLen)
		if useColor && b.Total > 0 {
			if b.Failed > b.Done {
				bar = "\033[31m" + bar + "\033[0m"
			} else if b.Failed > 0 {
				bar = "\033[33m" + bar + "\033[0m"
			} else {
				bar = "\033[32m" + bar + "\033[0m"
			}
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n", d, b.Total, b.Done, b.Failed, bar)
	}
	return w.Flush()
}

func runRunsCost(args []string) error {
	fs := flag.NewFlagSet("runs cost", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	since := fs.String("since", "", "Filter window (e.g. 24h, 7d, 30d)")
	model := fs.String("model", "", "Filter by model tier substring")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs cost [flags]\n\nShow cost and diff summary across agent runs.\n\nFlags:")
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

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	type modelStats struct {
		runs      int
		additions int32
		deletions int32
	}
	byModel := map[string]*modelStats{}
	totalRuns := 0
	totalAdditions := int32(0)
	totalDeletions := int32(0)
	runsWithCost := 0
	cursor := ""
	modelNeedle := strings.ToLower(*model)

	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			Cursor:        cursor,
		}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		done := false
		for _, r := range resp.Msg.GetAgentRuns() {
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts != nil {
					t := time.Unix(ts.Seconds, int64(ts.Nanos))
					if t.Before(sinceTime) {
						done = true
						break
					}
				}
			}
			tier := r.GetSpec().GetModelTier()
			if tier == "" {
				tier = "default"
			}
			if modelNeedle != "" && !strings.Contains(strings.ToLower(tier), modelNeedle) {
				continue
			}
			totalRuns++
			totalAdditions += r.GetStatus().GetTotalAdditions()
			totalDeletions += r.GetStatus().GetTotalDeletions()
			if r.GetStatus().GetTotalCost() != "" {
				runsWithCost++
			}
			ms := byModel[tier]
			if ms == nil {
				ms = &modelStats{}
				byModel[tier] = ms
			}
			ms.runs++
			ms.additions += r.GetStatus().GetTotalAdditions()
			ms.deletions += r.GetStatus().GetTotalDeletions()
		}
		if done || resp.Msg.GetNextCursor() == "" {
			break
		}
		cursor = resp.Msg.GetNextCursor()
	}

	label := "all time"
	if *since != "" {
		label = "last " + *since
	}
	if *project != "" {
		label += " · " + *project
	}
	fmt.Printf("Cost summary (%s) — %d runs\n\n", label, totalRuns)
	fmt.Printf("  Total diff:  +%d -%d lines\n", totalAdditions, totalDeletions)
	if runsWithCost > 0 {
		fmt.Printf("  Cost data:   %d/%d runs have cost estimates\n", runsWithCost, totalRuns)
	}
	if len(byModel) > 1 {
		fmt.Println("\n  By model tier:")
		tiers := make([]string, 0, len(byModel))
		for tier := range byModel {
			tiers = append(tiers, tier)
		}
		sort.Strings(tiers)
		for _, tier := range tiers {
			ms := byModel[tier]
			fmt.Printf("    %-22s  %d runs  +%d -%d lines\n", tier, ms.runs, ms.additions, ms.deletions)
		}
	}
	return nil
}

func runRunsVelocity(args []string) error {
	fs := flag.NewFlagSet("runs velocity", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	buckets := fs.Int("buckets", 24, "Number of hourly buckets to show (default 24 = last 24h)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs velocity [flags]\n\nShow runs-per-hour for the past N hours.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	bucketCount := *buckets
	if bucketCount < 1 {
		bucketCount = 24
	}
	counts := make([]int, bucketCount)
	sinceTime := now.Add(-time.Duration(bucketCount) * time.Hour)

	cursor := ""
	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			Cursor:        cursor,
		}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		done := false
		for _, r := range resp.Msg.GetAgentRuns() {
			ts := r.GetCreatedAt()
			if ts == nil {
				continue
			}
			t := time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
			if t.Before(sinceTime) {
				done = true
				break
			}
			idx := int(now.Sub(t) / time.Hour)
			if idx >= 0 && idx < bucketCount {
				counts[bucketCount-1-idx]++
			}
		}
		if done || resp.Msg.GetNextCursor() == "" {
			break
		}
		cursor = resp.Msg.GetNextCursor()
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	const barWidth = 20
	fmt.Printf("Runs per hour — last %dh\n\n", bucketCount)
	for i, count := range counts {
		hoursAgo := bucketCount - i
		barLen := 0
		if maxCount > 0 {
			barLen = count * barWidth / maxCount
		}
		fmt.Printf("  %-8s  %s %d\n", fmt.Sprintf("%dh ago", hoursAgo), strings.Repeat("█", barLen), count)
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	fmt.Printf("\n  Total: %d runs in last %dh (avg %.1f/h)\n", total, bucketCount, float64(total)/float64(bucketCount))
	return nil
}

func runRunsPercentiles(args []string) error {
	fs := flag.NewFlagSet("runs percentiles", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	since := fs.String("since", "", "Filter window (e.g. 24h, 7d, 30d)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs percentiles [flags]\n\nShow p50/p95/p99 duration for completed runs.\n\nFlags:")
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

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	var durations []time.Duration
	cursor := ""
	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			Cursor:        cursor,
		}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		done := false
		for _, r := range resp.Msg.GetAgentRuns() {
			if r.GetStatus().GetPhase() != apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED {
				continue
			}
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts != nil {
					t := time.Unix(ts.Seconds, int64(ts.Nanos))
					if t.Before(sinceTime) {
						done = true
						break
					}
				}
			}
			started := r.GetStatus().GetStartedAt()
			completed := r.GetStatus().GetCompletedAt()
			if started == nil || completed == nil {
				continue
			}
			s := time.Unix(started.Seconds, int64(started.Nanos))
			e := time.Unix(completed.Seconds, int64(completed.Nanos))
			if e.After(s) {
				durations = append(durations, e.Sub(s))
			}
		}
		if done || resp.Msg.GetNextCursor() == "" {
			break
		}
		cursor = resp.Msg.GetNextCursor()
	}

	if len(durations) == 0 {
		fmt.Println("No completed runs with duration data.")
		return nil
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	n := len(durations)
	pct := func(p float64) time.Duration {
		return durations[int(float64(n-1)*p/100)]
	}

	label := "all time"
	if *since != "" {
		label = "last " + *since
	}
	if *project != "" {
		label += " · " + *project
	}
	fmt.Printf("Duration percentiles (%s) — %d completed runs\n\n", label, n)
	fmt.Printf("  p50  %s\n", formatDuration(pct(50)))
	fmt.Printf("  p75  %s\n", formatDuration(pct(75)))
	fmt.Printf("  p95  %s\n", formatDuration(pct(95)))
	fmt.Printf("  p99  %s\n", formatDuration(pct(99)))
	fmt.Printf("  min  %s\n", formatDuration(durations[0]))
	fmt.Printf("  max  %s\n", formatDuration(durations[n-1]))
	return nil
}

func runRunsAnomalies(args []string) error {
	fs := flag.NewFlagSet("runs anomalies", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	project := fs.String("project", "", "Filter by project name")
	since := fs.String("since", "7d", "Filter window (default 7d)")
	threshold := fs.Float64("threshold", 2.0, "Flag runs with duration > threshold * p75")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks runs anomalies [flags]\n\nShow succeeded runs with unusually long duration.\n\nFlags:")
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

	c, err := newClient(*server)
	if err != nil {
		return err
	}

	type runEntry struct {
		id       string
		title    string
		duration time.Duration
		started  time.Time
	}
	var entries []runEntry
	var durations []time.Duration
	cursor := ""
	for {
		resp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:         100,
			ProjectFilter: *project,
			Cursor:        cursor,
		}))
		if err != nil {
			return fmt.Errorf("%s", humanizeErr(err))
		}
		done := false
		for _, r := range resp.Msg.GetAgentRuns() {
			if r.GetStatus().GetPhase() != apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED {
				continue
			}
			if !sinceTime.IsZero() {
				ts := r.GetCreatedAt()
				if ts != nil {
					t := time.Unix(ts.Seconds, int64(ts.Nanos))
					if t.Before(sinceTime) {
						done = true
						break
					}
				}
			}
			s := r.GetStatus().GetStartedAt()
			e := r.GetStatus().GetCompletedAt()
			if s == nil || e == nil {
				continue
			}
			start := time.Unix(s.Seconds, int64(s.Nanos))
			end := time.Unix(e.Seconds, int64(e.Nanos))
			dur := end.Sub(start)
			if dur <= 0 {
				continue
			}
			durations = append(durations, dur)
			title := r.GetSpec().GetDisplayName()
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			entries = append(entries, runEntry{
				id:       r.GetId(),
				title:    title,
				duration: dur,
				started:  start,
			})
		}
		if done || resp.Msg.GetNextCursor() == "" {
			break
		}
		cursor = resp.Msg.GetNextCursor()
	}

	if len(durations) < 4 {
		fmt.Printf("Not enough data (%d runs) to detect anomalies.\n", len(durations))
		return nil
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p75 := durations[int(float64(len(durations)-1)*0.75)]
	cutoff := time.Duration(float64(p75) * *threshold)

	var anomalies []runEntry
	for _, e := range entries {
		if e.duration > cutoff {
			anomalies = append(anomalies, e)
		}
	}
	sort.Slice(anomalies, func(i, j int) bool { return anomalies[i].duration > anomalies[j].duration })

	if len(anomalies) == 0 {
		fmt.Printf("No anomalies detected (p75=%s, cutoff=%s)\n", formatDuration(p75), formatDuration(cutoff))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DURATION\tID\tSTARTED\tTITLE")
	for _, a := range anomalies {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", formatDuration(a.duration), a.id, relativeTime(a.started), a.title)
	}
	w.Flush()
	plural := "ies"
	if len(anomalies) == 1 {
		plural = "y"
	}
	fmt.Printf("\n%d anomal%s (>%.1fx p75=%s, cutoff=%s)\n",
		len(anomalies), plural, *threshold, formatDuration(p75), formatDuration(cutoff))
	return nil
}
