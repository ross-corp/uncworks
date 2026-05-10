// cmd/uncworks — UNCWORKS CLI: setup, manage, and connect to UNCWORKS deployments.
package main

import (
	"fmt"
	"os"
)

// Version and Commit are set at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
)

const usage = `uncworks — manage UNCWORKS deployments

Usage:
  uncworks <command> [flags]

Commands:
  setup      Deploy UNCWORKS into a local Kubernetes cluster
  teardown   Uninstall UNCWORKS from the current cluster (prompts for confirmation)
  status     Show health of the UNCWORKS stack (exits non-zero if unhealthy)
  open       Start port-forward and open the web UI in a browser
  connect    Set the gRPC server address for tui and remote commands
  tui        Launch the terminal UI
  run        Submit a new agent run non-interactively
  runs       List, inspect, and stream logs for agent runs (list/get/logs)
  jobs       Show active (RUNNING + PENDING + WAITING) runs (alias for runs list --active)
  top        Live view of active runs sorted by elapsed time (alias for runs top)
  watch      Auto-refresh the run list (alias for runs watch)
  cancel     Request cancellation of a running agent
  kill       Alias for cancel
  input      Send human-in-the-loop response to a paused agent
  graph      Print the run execution tree
  ping       Check API connectivity and measure round-trip latency
  search     Search the knowledge base for past work
  config     Show or edit the CLI configuration

Flags:
  --version  Print the build version and exit
  --help     Show this help message

Run 'uncworks <command> --help' for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "setup":
		err = runSetup(args)
	case "teardown":
		err = runTeardown(args)
	case "status":
		err = runStatus(args)
	case "open":
		err = runOpen(args)
	case "connect":
		err = runConnect(args)
	case "tui":
		err = runTUI(args)
	case "run":
		err = runRun(args)
	case "runs":
		err = runRuns(args)
	case "jobs":
		err = runRunsList(append([]string{"--active"}, args...))
	case "top":
		err = runRunsTop(args)
	case "watch":
		err = runRunsWatch(args)
	case "cancel", "kill":
		err = runCancel(args)
	case "input":
		err = runInput(args)
	case "graph":
		err = runGraph(args)
	case "ping":
		err = runPing(args)
	case "search":
		err = runSearch(args)
	case "config":
		err = runConfig(args)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
	case "-v", "--version", "version":
		fmt.Printf("uncworks %s (commit %s)\n", Version, Commit)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n%s", cmd, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
