// config_cmd.go — uncworks config: show or manage CLI configuration.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config <subcommand> [flags]\n\nSubcommands:\n  show             Print the current CLI configuration\n  set-server       Set the gRPC server address\n  set-web-url      Set the web dashboard URL (used by 'runs ui')\n  set-model        Set the default model tier (used when --model-tier is not specified)\n  set-project      Set the default project name (used when --project is not specified)\n  set-feature      Set the default feature name (used when --feature is not specified)\n  unset <field>    Clear a config field (server, web-url, model, project, feature)\n  path             Print the config file path\n  edit             Open the config file in $EDITOR\n  reset            Reset the config to defaults\n\nFlags:")
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
	case "set-model":
		return runConfigSetModel(fs.Args()[1:])
	case "set-project":
		return runConfigSetStringField(fs.Args()[1:], "set-project", "default_project", func(cfg *Config, v string) { cfg.DefaultProject = v }, "default project name")
	case "set-feature":
		return runConfigSetStringField(fs.Args()[1:], "set-feature", "default_feature", func(cfg *Config, v string) { cfg.DefaultFeature = v }, "default feature name")
	case "unset":
		return runConfigUnset(fs.Args()[1:])
	case "set-auto-push":
		return runConfigSetAutoPush(fs.Args()[1:])
	case "path":
		p, err := configPath()
		if err != nil {
			return err
		}
		fmt.Println(p)
		return nil
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

func runConfigSetModel(args []string) error {
	fs := flag.NewFlagSet("config set-model", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config set-model <tier>\n\nSet the default model tier used when --model-tier is not specified.\nUse 'default' or '' to clear the default.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("model tier argument required")
	}
	tier := fs.Arg(0)
	if tier == "default" || tier == "none" {
		tier = ""
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.DefaultModelTier = tier
	if err := saveConfig(cfg); err != nil {
		return err
	}
	path, _ := configPath()
	if tier == "" {
		fmt.Printf("default_model_tier cleared (config: %s)\n", path)
	} else {
		fmt.Printf("default_model_tier set to %s (config: %s)\n", tier, path)
	}
	return nil
}

func runConfigSetStringField(args []string, cmd, field string, setter func(*Config, string), label string) error {
	fs := flag.NewFlagSet("config "+cmd, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: uncworks config %s <value>\n\nSet the %s. Use 'none' to clear.\n", cmd, label)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("value argument required")
	}
	val := fs.Arg(0)
	if val == "none" || val == "default" {
		val = ""
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	setter(&cfg, val)
	if err := saveConfig(cfg); err != nil {
		return err
	}
	path, _ := configPath()
	if val == "" {
		fmt.Printf("%s cleared (config: %s)\n", field, path)
	} else {
		fmt.Printf("%s set to %s (config: %s)\n", field, val, path)
	}
	return nil
}

func runConfigSetAutoPush(args []string) error {
	fs := flag.NewFlagSet("config set-auto-push", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config set-auto-push <true|false>\n\nSet default_auto_push. When true, --auto-push is applied to all new runs.\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("value argument required (true or false)")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch strings.ToLower(fs.Arg(0)) {
	case "true", "yes", "1", "on":
		cfg.DefaultAutoPush = true
	case "false", "no", "0", "off":
		cfg.DefaultAutoPush = false
	default:
		return fmt.Errorf("invalid value %q: must be true or false", fs.Arg(0))
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	path, _ := configPath()
	fmt.Printf("default_auto_push set to %v (config: %s)\n", cfg.DefaultAutoPush, path)
	return nil
}

func runConfigUnset(args []string) error {
	fs := flag.NewFlagSet("config unset", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks config unset <field>\n\nClear a config field. Valid fields: server, web-url, model, project, feature\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("field argument required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	field := fs.Arg(0)
	switch field {
	case "server", "server-address":
		cfg.Server.Address = ""
	case "web-url", "web_url":
		cfg.WebURL = ""
	case "model", "model-tier", "default-model":
		cfg.DefaultModelTier = ""
	case "project", "default-project":
		cfg.DefaultProject = ""
	case "feature", "default-feature":
		cfg.DefaultFeature = ""
	case "auto-push", "default-auto-push":
		cfg.DefaultAutoPush = false
	default:
		return fmt.Errorf("unknown field %q: must be server, web-url, model, project, feature, or auto-push", field)
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	path, _ := configPath()
	fmt.Printf("%s cleared (config: %s)\n", field, path)
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
			"server_address":     cfg.Server.Address,
			"web_url":            cfg.WebURL,
			"default_model_tier": cfg.DefaultModelTier,
			"default_project":    cfg.DefaultProject,
			"default_feature":    cfg.DefaultFeature,
			"default_auto_push":  cfg.DefaultAutoPush,
			"config_file":        path,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	notSet := func(key, hint string) string {
		return fmt.Sprintf("(not set — use 'config %s')", hint)
	}
	addr := cfg.Server.Address
	if addr == "" {
		addr = notSet("server.address", "set-server <addr>")
	}
	fmt.Printf("server.address:      %s\n", addr)
	webURL := cfg.WebURL
	if webURL == "" {
		webURL = notSet("web_url", "set-web-url <url>")
	}
	fmt.Printf("web_url:             %s\n", webURL)
	model := cfg.DefaultModelTier
	if model == "" {
		model = notSet("default_model_tier", "set-model <tier>")
	}
	fmt.Printf("default_model_tier:  %s\n", model)
	proj := cfg.DefaultProject
	if proj == "" {
		proj = notSet("default_project", "set-project <name>")
	}
	fmt.Printf("default_project:     %s\n", proj)
	feat := cfg.DefaultFeature
	if feat == "" {
		feat = notSet("default_feature", "set-feature <name>")
	}
	fmt.Printf("default_feature:     %s\n", feat)
	fmt.Printf("default_auto_push:   %v\n", cfg.DefaultAutoPush)
	fmt.Printf("config file:         %s\n", path)
	return nil
}
