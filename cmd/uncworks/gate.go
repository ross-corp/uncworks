// gate.go — uncworks gate: the factory-gate verdicts for a pull request.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/uncworks/aot/internal/gate"
)

var errGateUsage = errors.New("gate usage")

const gateUsage = `Compute the factory-gate verdicts for a pull request.

Usage:
  uncworks gate check [flags]

Flags:
  --change <id>   the openspec change this pull request implements
  --base <ref>    the base ref (default origin/main)
  --head <ref>    the head ref (default HEAD)
  --root <path>   the repository root (default .)
  --branch <name> derive --change from a branch named change/<id> or aot/<id>
  --json          machine-readable output

Two verdicts are required, factory/tier and factory/citations, and two are
advisory, factory/spec-conformance and factory/order. check exits 1 when a
required verdict fails.
`

// changeFromBranch reads the change id out of a branch name. This is a
// convenience for CI, never an input to the tier: a tier read from a branch
// name could be lowered by renaming the branch.
var changeFromBranch = regexp.MustCompile(`^(?:change|aot|spec)/([A-Za-z0-9._-]+)`)

func runGate(args []string) error {
	sub := "check"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "-h", "--help", "help":
		fmt.Print(gateUsage)
		return nil
	case "check":
	default:
		return fmt.Errorf("%w: unknown subcommand %q", errGateUsage, sub)
	}

	opts := gate.Options{Base: "origin/main", Head: "HEAD", Root: "."}
	asJSON := false
	branch := ""
	for i := 0; i < len(args); i++ {
		needsValue := args[i] != "--json"
		if needsValue && i+1 >= len(args) {
			return fmt.Errorf("%w: %s needs a value", errGateUsage, args[i])
		}
		switch args[i] {
		case "--json":
			asJSON = true
			continue
		case "--change":
			opts.ChangeID = args[i+1]
		case "--base":
			opts.Base = args[i+1]
		case "--head":
			opts.Head = args[i+1]
		case "--root":
			opts.Root = args[i+1]
		case "--branch":
			branch = args[i+1]
		default:
			return fmt.Errorf("%w: unknown flag %q", errGateUsage, args[i])
		}
		i++
	}
	if opts.ChangeID == "" && branch != "" {
		if m := changeFromBranch.FindStringSubmatch(branch); m != nil {
			opts.ChangeID = m[1]
		}
	}

	result, checkErr := gate.Check(context.Background(), gate.GitRepo{Root: opts.Root}, opts)
	if checkErr != nil && !errors.Is(checkErr, gate.ErrFailed) {
		return fmt.Errorf("gate check: %w", checkErr)
	}

	if asJSON {
		if err := writeJSON(result); err != nil {
			return err
		}
	} else {
		printGateResult(result)
	}
	if checkErr != nil {
		return fmt.Errorf("gate check: %w", checkErr)
	}
	return nil
}

func printGateResult(result gate.Result) {
	change := result.Change
	if change == "" {
		change = "(no change named)"
	}
	fmt.Printf("change: %s\ntier:   %s, because %s\n\n", change, result.Tier, result.Reason)
	for _, v := range result.Verdicts {
		status := "PASS"
		if !v.Pass {
			status = "FAIL"
		}
		kind := "advisory"
		if v.Required {
			kind = "required"
		}
		fmt.Printf("%-4s %-8s %-26s %s\n", status, kind, v.Name, v.Summary)
		for _, line := range v.Detail {
			fmt.Printf("              %s\n", line)
		}
	}
	if result.Failed() {
		fmt.Fprintln(os.Stderr, "\na required verdict failed")
	}
}
