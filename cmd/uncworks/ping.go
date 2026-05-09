// ping.go — uncworks ping: quick API connectivity check with latency.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	server := fs.String("server", "", "gRPC server address (overrides config)")
	count := fs.Int("count", 3, "Number of pings to send")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks ping [flags]\n\nCheck API connectivity and measure round-trip latency.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newClient(*server)
	if err != nil {
		return err
	}

	cfg, _ := loadConfig()
	addr := *server
	if addr == "" {
		addr = cfg.Server.Address
	}
	if addr == "" {
		addr = "localhost:30055 (port-forward)"
	}

	type result struct {
		Seq   int     `json:"seq"`
		RTTMS float64 `json:"rtt_ms,omitempty"`
		Error string  `json:"error,omitempty"`
		OK    bool    `json:"ok"`
	}

	var results []result
	var total time.Duration
	var failures int

	if !*jsonOut {
		fmt.Printf("PING %s\n", addr)
	}

	for i := 0; i < *count; i++ {
		start := time.Now()
		_, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		rtt := time.Since(start)
		if apiErr != nil {
			msg := humanizeErr(apiErr)
			results = append(results, result{Seq: i + 1, Error: msg, OK: false})
			if !*jsonOut {
				fmt.Printf("seq=%d error: %s\n", i+1, msg)
			}
			failures++
		} else {
			rttMS := float64(rtt.Milliseconds())
			results = append(results, result{Seq: i + 1, RTTMS: rttMS, OK: true})
			if !*jsonOut {
				fmt.Printf("seq=%d rtt=%s\n", i+1, rtt.Round(time.Millisecond))
			}
			total += rtt
		}
	}

	sent := *count
	received := sent - failures

	if *jsonOut {
		avgRTTMS := 0.0
		if received > 0 {
			avgRTTMS = float64((total / time.Duration(received)).Milliseconds())
		}
		out := map[string]interface{}{
			"address":    addr,
			"count":      sent,
			"received":   received,
			"failed":     failures,
			"loss_pct":   failures * 100 / sent,
			"avg_rtt_ms": avgRTTMS,
			"results":    results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		fmt.Printf("\n%d sent, %d received, %d%% loss", sent, received, failures*100/sent)
		if received > 0 {
			avg := total / time.Duration(received)
			fmt.Printf(", avg rtt %s", avg.Round(time.Millisecond))
		}
		fmt.Println()
	}

	if failures == sent {
		return fmt.Errorf("all pings failed")
	}
	return nil
}
