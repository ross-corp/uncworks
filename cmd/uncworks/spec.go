// spec.go — uncworks spec: the deterministic rubric lint and the ready set.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/uncworks/aot/internal/specutil"
)

var errSpecUsage = errors.New("spec usage")

const specUsage = `Deterministic checks over an OpenSpec change.

Usage:
  uncworks spec check [<change>...]    rubric-lint every change, or the named ones
  uncworks spec next [<change>]        the runnable tasks in the active phase
  uncworks spec status [<change>]      task completion
  uncworks spec graph [<change>]       the phases and edges as Mermaid
  uncworks spec rules                  the rubric this build enforces

Flags:
  --json          machine-readable output
  --dir <path>    openspec/changes directory (default openspec/changes)

check exits 1 when a rule fails at error severity. next exits 2 when work
remains but nothing is runnable, which means the declared dependencies form a
cycle: fix tasks.md rather than retrying.
`

type specFlags struct {
	asJSON bool
	dir    string
	names  []string
}

func parseSpecFlags(args []string) (specFlags, error) {
	flags := specFlags{dir: filepath.Join("openspec", "changes")}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.asJSON = true
		case "--dir":
			if i+1 >= len(args) {
				return flags, fmt.Errorf("%w: --dir needs a value", errSpecUsage)
			}
			i++
			flags.dir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return flags, fmt.Errorf("%w: unknown flag %q", errSpecUsage, args[i])
			}
			flags.names = append(flags.names, args[i])
		}
	}
	return flags, nil
}

// load resolves the named changes, or every active change when none is named.
func (f specFlags) load() ([]*specutil.Change, error) {
	if len(f.names) == 0 {
		changes, err := specutil.LoadAll(f.dir)
		if err != nil {
			return nil, fmt.Errorf("loading changes: %w", err)
		}
		return changes, nil
	}
	changes := make([]*specutil.Change, 0, len(f.names))
	for _, name := range f.names {
		dir := name
		if _, err := os.Stat(dir); err != nil {
			dir = filepath.Join(f.dir, name)
		}
		change, err := specutil.Load(dir)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func runSpec(args []string) error {
	sub := "check"
	if len(args) > 0 {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "-h", "--help", "help":
		fmt.Print(specUsage)
		return nil
	case "rules":
		for _, rule := range specutil.Rules {
			fmt.Printf("%-26s %s\n", rule.ID, rule.What)
		}
		return nil
	}

	flags, err := parseSpecFlags(args)
	if err != nil {
		return err
	}
	changes, err := flags.load()
	if err != nil {
		return err
	}

	switch sub {
	case "check":
		return specCheck(changes, flags)
	case "next":
		return specNext(changes, flags)
	case "status":
		return specStatus(changes, flags)
	case "graph":
		for _, change := range changes {
			fmt.Printf("%%%% %s\n%s\n", change.Name, specutil.Mermaid(change))
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q", errSpecUsage, sub)
	}
}

func specCheck(changes []*specutil.Change, flags specFlags) error {
	reports := make([]specutil.Report, 0, len(changes))
	failed := false
	for _, change := range changes {
		report := specutil.Check(change)
		reports = append(reports, report)
		if report.Failed() {
			failed = true
		}
	}

	if flags.asJSON {
		if err := writeJSON(reports); err != nil {
			return err
		}
	} else {
		for _, report := range reports {
			if len(report.Findings) == 0 {
				fmt.Printf("%s: 0 findings\n", report.Change)
				continue
			}
			fmt.Printf("%s:\n", report.Change)
			for _, f := range report.Findings {
				location := f.File
				if f.Line > 0 {
					location = fmt.Sprintf("%s:%d", f.File, f.Line)
				}
				fmt.Printf("  %-5s %-26s %s\n           %s\n", f.Severity, f.Rule, location, f.Message)
			}
		}
	}
	if failed {
		return fmt.Errorf("spec check: %w", specutil.ErrRubric)
	}
	return nil
}

func specNext(changes []*specutil.Change, flags specFlags) error {
	for _, change := range changes {
		ready, err := specutil.Next(change)
		if err != nil {
			return fmt.Errorf("spec next: %w", err)
		}
		if flags.asJSON {
			if err := writeJSON(ready); err != nil {
				return err
			}
			continue
		}
		if ready.Done {
			fmt.Printf("%s: every task is done\n", change.Name)
			continue
		}
		fmt.Printf("%s: phase %d %q (shape %s)\n", change.Name, ready.Phase, ready.Name, ready.Shape)
		if ready.Stop != "" {
			// Printed verbatim, and deliberately not turned into a command. A
			// stop condition often names a file in backticks, and a guessed
			// command looks right and proves nothing.
			fmt.Printf("  stop: %s\n", ready.Stop)
		}
		if ready.Merge != "" {
			fmt.Printf("  merge: %s\n", ready.Merge)
		}
		fmt.Println("  ready:")
		for _, task := range ready.Tasks {
			fmt.Printf("    %s %s\n", task.ID, task.Text)
		}
		if ready.Concurrent {
			fmt.Println("  runnable concurrently")
			if conflicts := specutil.WriteConflicts(ready); len(conflicts) > 0 {
				paths := make([]string, 0, len(conflicts))
				for path := range conflicts {
					paths = append(paths, path)
				}
				sort.Strings(paths)
				fmt.Println("  but these write sets intersect, so run them in one context:")
				for _, path := range paths {
					fmt.Printf("    %s: %s\n", path, strings.Join(conflicts[path], ", "))
				}
			}
		}
		for _, blocked := range ready.Blocked {
			fmt.Printf("  blocked: %s waits on %s\n", blocked.ID, strings.Join(blocked.Waiting, ", "))
		}
		if len(ready.Tasks) == 0 {
			return fmt.Errorf("spec next: %w", specutil.ErrCycle)
		}
	}
	return nil
}

func specStatus(changes []*specutil.Change, flags specFlags) error {
	if flags.asJSON {
		all := make([]specutil.Progress, 0, len(changes))
		for _, change := range changes {
			all = append(all, specutil.Status(change))
		}
		return writeJSON(all)
	}
	for _, change := range changes {
		progress := specutil.Status(change)
		fmt.Printf("%-40s %d/%d\n", progress.Change, progress.Done, progress.Total)
	}
	return nil
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}
