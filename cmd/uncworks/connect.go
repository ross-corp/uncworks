// connect.go — uncworks connect: store remote gRPC server address.
package main

import (
	"flag"
	"fmt"
)

func runConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: uncworks connect <address>

Store a gRPC server address for use by 'uncworks tui'.
Pass "local" to reset to the default local port-forward.

Examples:
  uncworks connect grpc.example.com:50055
  uncworks connect local`)
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
	return nil
}
