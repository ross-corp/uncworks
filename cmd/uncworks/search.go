// search.go — uncworks search: search the knowledge base for past work.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	limit := fs.Int("limit", 10, "Maximum number of results to return")
	repo := fs.String("repo", "", "Filter results to a specific repository URL")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks search <query> [flags]

Search the knowledge base for relevant past work.

Example:
  uncworks search "implement OAuth in Go"

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("search query argument required")
	}
	query := fs.Arg(0)

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:  query,
		Limit:  int32(*limit),
		RepoUrl: *repo,
	})
	resp, err := client.SearchPastWork(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	results := resp.Msg.GetResults()
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, r := range results {
		snippet := r.GetChunkText()
		if len(snippet) > 120 {
			snippet = snippet[:117] + "..."
		}
		fmt.Printf("%d. [%.3f] %s: %s\n", i+1, r.GetSimilarityScore(), r.GetRunId(), snippet)
	}
	return nil
}
