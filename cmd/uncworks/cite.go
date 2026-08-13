// cite.go — uncworks cite: capture and verify pinned external-factual claims.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/uncworks/aot/internal/citelock"
)

// errCiteUsage reports that the command line was wrong, as opposed to the
// citations themselves.
var errCiteUsage = errors.New("cite usage")

// report writes one finding per line to stderr, prefixed so it is obvious which
// tool produced it.
func report(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cite: "+format+"\n", args...)
}

func runCite(args []string) error {
	sub := "verify"
	if len(args) > 0 {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "-h", "--help", "help":
		fmt.Print(citelock.UsageText)
		return nil
	case "verify":
		if err := citelock.Verify(firstOr(args, "."), report); err != nil {
			return fmt.Errorf("cite verify: %w", err)
		}
		return nil
	case "capture":
		return runCiteCapture(args)
	case "recheck":
		if err := citelock.Recheck(context.Background(), firstOr(args, "."), report); err != nil {
			return fmt.Errorf("cite recheck: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q, use verify, capture, or recheck", errCiteUsage, sub)
	}
}

func runCiteCapture(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: uncworks cite capture <url> --id <id> --quote <text> --class <class>",
			errCiteUsage)
	}
	opts := citelock.CaptureOptions{URL: args[0], LockDir: "."}
	rest := args[1:]
	for i := 0; i < len(rest); i += 2 {
		if i+1 >= len(rest) {
			return fmt.Errorf("%w: %s needs a value", errCiteUsage, rest[i])
		}
		value := rest[i+1]
		switch rest[i] {
		case "--id":
			opts.ID = value
		case "--quote":
			opts.Quote = value
		case "--class":
			opts.ClaimClass = value
		case "--doi":
			opts.DOI = value
		case "--lockdir":
			opts.LockDir = value
		default:
			return fmt.Errorf("%w: unknown capture flag %q", errCiteUsage, rest[i])
		}
	}
	if err := citelock.Capture(context.Background(), opts, report); err != nil {
		return fmt.Errorf("cite capture: %w", err)
	}
	return nil
}

func firstOr(args []string, fallback string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return fallback
}
