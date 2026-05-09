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
	count := fs.Int("count", 3, "Number of pings to send (0 = infinite with --watch)")
	interval := fs.Duration("interval", 0, "Delay between pings (e.g. 500ms, 1s; 0 = no delay)")
	watch := fs.Bool("watch", false, "Continuous mode: ping every --interval until Ctrl+C")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Print only the average RTT or 'error' (for scripting)")
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
	var minRTT, maxRTT time.Duration
	var failures int

	if *watch {
		watchInterval := *interval
		if watchInterval == 0 {
			watchInterval = time.Second
		}
		if !*jsonOut && !*quiet {
			fmt.Printf("PING %s (Ctrl+C to stop)\n", addr)
		}
		for seq := 1; ; seq++ {
			if seq > 1 {
				time.Sleep(watchInterval)
			}
			start := time.Now()
			_, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
			rtt := time.Since(start)
			ts := time.Now().Format("15:04:05")
			if apiErr != nil {
				if *quiet {
					fmt.Println("error")
				} else {
					fmt.Printf("[%s] seq=%d error: %s\n", ts, seq, humanizeErr(apiErr))
				}
			} else {
				if *quiet {
					fmt.Println(rtt.Round(time.Millisecond))
				} else {
					fmt.Printf("[%s] seq=%d rtt=%s\n", ts, seq, rtt.Round(time.Millisecond))
				}
			}
		}
	}

	if !*jsonOut && !*quiet {
		fmt.Printf("PING %s\n", addr)
	}

	for i := 0; i < *count; i++ {
		if i > 0 && *interval > 0 {
			time.Sleep(*interval)
		}
		start := time.Now()
		_, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		rtt := time.Since(start)
		if apiErr != nil {
			msg := humanizeErr(apiErr)
			results = append(results, result{Seq: i + 1, Error: msg, OK: false})
			if !*jsonOut && !*quiet {
				fmt.Printf("seq=%d error: %s\n", i+1, msg)
			}
			failures++
		} else {
			rttMS := float64(rtt.Milliseconds())
			results = append(results, result{Seq: i + 1, RTTMS: rttMS, OK: true})
			if !*jsonOut && !*quiet {
				fmt.Printf("seq=%d rtt=%s\n", i+1, rtt.Round(time.Millisecond))
			}
			total += rtt
			if minRTT == 0 || rtt < minRTT {
				minRTT = rtt
			}
			if rtt > maxRTT {
				maxRTT = rtt
			}
		}
	}

	sent := *count
	received := sent - failures

	if *quiet {
		if received == 0 {
			fmt.Println("error")
		} else {
			fmt.Println((total / time.Duration(received)).Round(time.Millisecond))
		}
		if failures == sent {
			return fmt.Errorf("all pings failed")
		}
		return nil
	}

	if *jsonOut {
		avgRTTMS := 0.0
		minRTTMS := 0.0
		maxRTTMS := 0.0
		if received > 0 {
			avgRTTMS = float64((total / time.Duration(received)).Milliseconds())
			minRTTMS = float64(minRTT.Milliseconds())
			maxRTTMS = float64(maxRTT.Milliseconds())
		}
		lossPct := 0
		if sent > 0 {
			lossPct = failures * 100 / sent
		}
		out := map[string]interface{}{
			"address":    addr,
			"count":      sent,
			"received":   received,
			"failed":     failures,
			"loss_pct":   lossPct,
			"avg_rtt_ms": avgRTTMS,
			"min_rtt_ms": minRTTMS,
			"max_rtt_ms": maxRTTMS,
			"results":    results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		lossPct := 0
		if sent > 0 {
			lossPct = failures * 100 / sent
		}
		fmt.Printf("\n%d sent, %d received, %d%% loss", sent, received, lossPct)
		if received > 0 {
			avg := total / time.Duration(received)
			fmt.Printf(", rtt min/avg/max %s/%s/%s",
				minRTT.Round(time.Millisecond),
				avg.Round(time.Millisecond),
				maxRTT.Round(time.Millisecond))
		}
		fmt.Println()
	}

	if failures == sent {
		return fmt.Errorf("all pings failed")
	}
	return nil
}
