// search.go — uncworks search: search the knowledge base for past work.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	limit := fs.Int("limit", 10, "Maximum number of results to return")
	repo := fs.String("repo", "", "Filter results to a specific repository URL")
	since := fs.String("since", "", "Filter to results created within this window (e.g. 1h, 24h, 7d)")
	source := fs.String("source", "", "Filter by source type (code, trace, source-code; default: all)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	snippetLen := fs.Int("snippet-length", 200, "Maximum length of result snippets")
	minScore := fs.Float64("min-score", 0, "Minimum similarity score threshold (0.0-1.0; 0 = no filter)")
	idsOnly := fs.Bool("ids-only", false, "Print only matching run IDs (one per line)")
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

	searchReq := &apiv1.SearchPastWorkRequest{
		Query:   query,
		Limit:   int32(*limit),
		RepoUrl: *repo,
	}

	if *since != "" {
		d, err := parseSinceDuration(*since)
		if err != nil {
			return fmt.Errorf("--since %q: %w", *since, err)
		}
		searchReq.CreatedAfter = timestamppb.New(time.Now().Add(-d))
	}

	if *source != "" {
		switch strings.ToLower(*source) {
		case "code":
			searchReq.SourceFilter = apiv1.SourceFilter_SOURCE_FILTER_CODE
		case "trace":
			searchReq.SourceFilter = apiv1.SourceFilter_SOURCE_FILTER_TRACE
		case "source-code":
			searchReq.SourceFilter = apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE
		case "all":
			searchReq.SourceFilter = apiv1.SourceFilter_SOURCE_FILTER_ALL
		default:
			return fmt.Errorf("--source %q: must be code, trace, source-code, or all", *source)
		}
	}

	req := connect.NewRequest(searchReq)
	resp, err := client.SearchPastWork(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", humanizeErr(err))
	}

	results := resp.Msg.GetResults()
	if *minScore > 0 {
		filtered := results[:0]
		for _, r := range results {
			if float64(r.GetSimilarityScore()) >= *minScore {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}
	if len(results) == 0 {
		if *jsonOut {
			fmt.Println("[]")
		} else {
			fmt.Println("No results found.")
		}
		return nil
	}

	if *idsOnly {
		seen := map[string]bool{}
		for _, r := range results {
			if id := r.GetRunId(); id != "" && !seen[id] {
				fmt.Println(id)
				seen[id] = true
			}
		}
		return nil
	}

	if *jsonOut {
		type resultJSON struct {
			Rank    int     `json:"rank"`
			Score   float64 `json:"score"`
			RunID   string  `json:"run_id"`
			RepoURL string  `json:"repo_url,omitempty"`
			Snippet string  `json:"snippet"`
			Age     string  `json:"age,omitempty"`
		}
		out := make([]resultJSON, 0, len(results))
		for i, r := range results {
			age := ""
			if ts := r.GetCreatedAt(); ts != nil {
				age = time.Since(ts.AsTime()).Round(time.Hour).String() + " ago"
			}
			out = append(out, resultJSON{
				Rank:    i + 1,
				Score:   float64(r.GetSimilarityScore()),
				RunID:   r.GetRunId(),
				RepoURL: r.GetRepoUrl(),
				Snippet: strings.Join(strings.Fields(r.GetChunkText()), " "),
				Age:     age,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	for i, r := range results {
		// Header line: rank, score, run ID, and age if available.
		age := ""
		if ts := r.GetCreatedAt(); ts != nil {
			age = " · " + time.Since(ts.AsTime()).Round(time.Hour).String() + " ago"
		}
		fmt.Printf("%d. [%.3f] %s%s\n", i+1, r.GetSimilarityScore(), r.GetRunId(), age)

		// Repo URL on its own line when present.
		if u := r.GetRepoUrl(); u != "" {
			fmt.Printf("   repo: %s\n", u)
		}

		// Snippet: collapse whitespace, then truncate at word boundary.
		snippet := strings.Join(strings.Fields(r.GetChunkText()), " ")
		max := *snippetLen
		if len(snippet) > max {
			cutAt := strings.LastIndex(snippet[:max], " ")
			if cutAt < max/2 {
				cutAt = max
			}
			snippet = snippet[:cutAt] + "..."
		}
		fmt.Printf("   %s\n\n", snippet)
	}
	return nil
}
