// connect.go — uncworks connect: store remote gRPC server address.
package main

import (
	"context"
	"flag"
	"fmt"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func runConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	test := fs.Bool("test", false, "Verify connectivity after saving the address")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks connect <address> [flags]

Store a gRPC server address for use by 'uncworks tui'.
Pass "local" to reset to the default local port-forward.

Examples:
  uncworks connect http://100.81.3.110:30055 --test
  uncworks connect local

Flags:`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("address argument required")
	}
	addr := fs.Arg(0)
	if addr == "local" {
		addr = ""
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.Server.Address = addr
	if err := saveConfig(cfg); err != nil {
		return err
	}

	path, _ := configPath()
	if addr == "" {
		fmt.Printf("Server address reset to local port-forward (config: %s)\n", path)
	} else {
		fmt.Printf("Server address set to %s (config: %s)\n", addr, path)
	}

	if *test {
		client, _ := newClient(addr)
		_, apiErr := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
		if apiErr != nil {
			fmt.Printf("Connection test failed: %s\n", humanizeErr(apiErr))
		} else {
			fmt.Println("Connection OK")
		}
	}
	return nil
}
