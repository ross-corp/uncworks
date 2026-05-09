// input.go — uncworks input: send human-in-the-loop response to a paused agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runInput(args []string) error {
	fs := flag.NewFlagSet("input", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	follow := fs.Bool("follow", false, "Stream logs after sending input until the run completes")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks input <run-id> [<text>] [flags]

Send a human-in-the-loop response to a paused agent run.
If <text> is omitted, reads from stdin.

Examples:
  uncworks input abc123 "approved, proceed"
  echo "approved" | uncworks input abc123
  uncworks input abc123 --follow

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}
	id := fs.Arg(0)

	var text string
	if fs.NArg() >= 2 {
		text = strings.Join(fs.Args()[1:], " ")
	} else {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		text = strings.TrimRight(string(raw), "\n")
		if text == "" {
			return fmt.Errorf("input text is empty")
		}
	}

	if len(text) > 10000 {
		return fmt.Errorf("input too long: %d chars (max 10000)", len(text))
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	// Warn if run is not in WAITING_FOR_INPUT phase.
	getResp, getErr := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: id}))
	if getErr == nil {
		phase := getResp.Msg.GetStatus().GetPhase()
		if phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
			fmt.Fprintf(os.Stderr, "warning: run %s is in phase %s, not WAITING — input may be ignored\n", id, phaseLabel(phase))
		} else if prompt := getResp.Msg.GetStatus().GetMessage(); prompt != "" {
			fmt.Printf("Agent is asking: %s\n", prompt)
		}
	}

	req := connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: id,
		Input:      text,
	})
	_, err = client.SendHumanInput(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	fmt.Printf("Input sent to run %s\n", id)

	if *follow {
		return runRunsTail([]string{id, "--server=" + *server})
	}
	return nil
}
