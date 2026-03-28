// connect.go — uncworks connect: store remote gRPC server address.
package main

import (
	"flag"
	"fmt"
)

func runConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks connect <address>\n\nStore a gRPC server address for use by 'uncworks tui'.\nExample: uncworks connect grpc.example.com:50055")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("address argument required")
	}
	addr := fs.Arg(0)

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.Server.Address = addr
	if err := saveConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("Server address set to %s\n", addr)
	return nil
}
