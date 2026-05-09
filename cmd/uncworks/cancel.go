// cancel.go — uncworks cancel: request cancellation of a running agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runCancel(args []string) error {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks cancel <run-id> [<run-id> ...] [flags]\n\nRequest cancellation of one or more running agents.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("run ID argument required")
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	var errs []string
	for _, id := range fs.Args() {
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
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("%d cancellation(s) failed", len(errs))
	}
	return nil
}
