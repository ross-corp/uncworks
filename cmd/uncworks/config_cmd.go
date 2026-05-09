// config_cmd.go — uncworks config: show or manage CLI configuration.
package main

import (
	"flag"
	"fmt"
	"os"
)

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config <subcommand> [flags]\n\nSubcommands:\n  show    Print the current CLI configuration\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("subcommand required")
	}
	switch fs.Arg(0) {
	case "show":
		return runConfigShow(fs.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", fs.Arg(0))
		fs.Usage()
		return fmt.Errorf("unknown subcommand %q", fs.Arg(0))
	}
}

func runConfigShow(_ []string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	addr := cfg.Server.Address
	if addr == "" {
		addr = "(not set — using local port-forward)"
	}
	fmt.Printf("server.address:  %s\n", addr)
	fmt.Printf("config file:     %s\n", path)
	return nil
}
