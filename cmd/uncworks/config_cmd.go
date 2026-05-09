// config_cmd.go — uncworks config: show or manage CLI configuration.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config <subcommand> [flags]\n\nSubcommands:\n  show           Print the current CLI configuration\n  set-server     Set the gRPC server address\n  set-web-url    Set the web dashboard URL (used by 'runs ui')\n  edit           Open the config file in $EDITOR\n  reset          Reset the config to defaults\n\nFlags:")
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
	case "set-server":
		return runConfigSetServer(fs.Args()[1:])
	case "set-web-url":
		return runConfigSetWebURL(fs.Args()[1:])
	case "edit":
		return runConfigEdit(fs.Args()[1:])
	case "reset":
		return runConfigReset(fs.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", fs.Arg(0))
		fs.Usage()
		return fmt.Errorf("unknown subcommand %q", fs.Arg(0))
	}
}

func runConfigSetServer(args []string) error {
	fs := flag.NewFlagSet("config set-server", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config set-server <address>\n\nSet the gRPC server address. Use 'local' to reset to port-forward mode.")
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
		fmt.Printf("server.address reset to local port-forward (config: %s)\n", path)
	} else {
		fmt.Printf("server.address set to %s (config: %s)\n", addr, path)
	}
	return nil
}

func runConfigSetWebURL(args []string) error {
	fs := flag.NewFlagSet("config set-web-url", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config set-web-url <url>\n\nSet the UNCWORKS web dashboard base URL (e.g. http://192.168.1.10:30080).\nUsed by 'uncworks runs ui' to open run detail pages.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("URL argument required")
	}
	url := fs.Arg(0)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.WebURL = url
	if err := saveConfig(cfg); err != nil {
		return err
	}
	path, _ := configPath()
	fmt.Printf("web_url set to %s (config: %s)\n", url, path)
	return nil
}

func runConfigEdit(args []string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	// Ensure the file exists so the editor has something to open.
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		if err := saveConfig(Config{}); err != nil {
			return fmt.Errorf("creating default config: %w", err)
		}
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runConfigReset(args []string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := saveConfig(Config{}); err != nil {
		return err
	}
	fmt.Printf("Config reset to defaults (%s)\n", path)
	return nil
}

func runConfigShow(args []string) error {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	path, err := configPath()
	if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if *jsonOut {
		out := map[string]interface{}{
			"server_address": cfg.Server.Address,
			"web_url":        cfg.WebURL,
			"config_file":    path,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	addr := cfg.Server.Address
	if addr == "" {
		addr = "(not set — using local port-forward)"
	}
	fmt.Printf("server.address:  %s\n", addr)
	webURL := cfg.WebURL
	if webURL == "" {
		webURL = "(not set — use 'config set-web-url <url>')"
	}
	fmt.Printf("web_url:         %s\n", webURL)
	fmt.Printf("config file:     %s\n", path)
	return nil
}
