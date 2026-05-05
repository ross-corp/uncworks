// input.go — uncworks input: send human-in-the-loop response to a paused agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runInput(args []string) error {
	fs := flag.NewFlagSet("input", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks input <run-id> <text> [flags]

Send a human-in-the-loop response to a paused agent run.

Example:
  uncworks input abc123 "approved, proceed"

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("run ID and input text arguments are required")
	}
	id := fs.Arg(0)
	text := fs.Arg(1)

	// Validate input length
	if len(text) > 10000 {
		return fmt.Errorf("input too long: %d chars (max 10000)", len(text))
	}

	client, err := newClient(*server)
	if err != nil {
		return err
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
	return nil
}
