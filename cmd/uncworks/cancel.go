// cancel.go — uncworks cancel: request cancellation of a running agent.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// Sentinel errors, so a caller can match on what went wrong rather than
// on a message.
var (
	// errFailed reports an operation that did not complete.
	errFailed = errors.New("failed")
	// errInvalidInput reports a caller mistake: a bad argument, flag, or value.
	errInvalidInput = errors.New("invalid input")
	// errNotFound reports that a named thing is absent.
	errNotFound = errors.New("not found")
	// errUnavailable reports that a dependency did not answer.
	errUnavailable = errors.New("unavailable")
)

func runCancel(args []string) error {
	args = normalizeRunArgs(args)
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	lastRun := fs.Bool("last", false, "Cancel the most recent active run (auto-detect ID)")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(fs.Output(), "Usage: uncworks cancel <run-id> [<run-id> ...] [flags]\n\nRequest cancellation of one or more running agents.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	ids := fs.Args()
	if *lastRun && len(ids) == 0 {
		resp, listErr := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
			Limit:       1,
			PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
		}))
		if listErr != nil {
			return fmt.Errorf("%w: %s", errFailed, humanizeErr(listErr))
		}
		if len(resp.Msg.GetAgentRuns()) == 0 {
			// Fall back to any active run.
			resp2, listErr2 := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
			if listErr2 != nil || len(resp2.Msg.GetAgentRuns()) == 0 {
				return fmt.Errorf("%w: no active runs found to cancel", errFailed)
			}
			ids = []string{resp2.Msg.GetAgentRuns()[0].GetId()}
		} else {
			ids = []string{resp.Msg.GetAgentRuns()[0].GetId()}
		}
	}

	if len(ids) == 0 {
		fs.Usage()
		return fmt.Errorf("%w: run ID argument required", errInvalidInput)
	}

	var errs []string
	for _, id := range ids {
		req := connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: id})
		_, cancelErr := client.CancelAgentRun(context.Background(), req)
		if cancelErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", id, humanizeErr(cancelErr)))
		} else {
			fmt.Printf("Run %s cancellation requested\n", id)
		}
	}
	if len(errs) > 0 {
		for _, e := range errs {
			_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("%w: %d cancellation(s) failed", errFailed, len(errs))
	}
	return nil
}
