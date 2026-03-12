package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/uncworks/aot/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: aot <command>\nCommands: open, dashboard\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "open":
		cmdOpen()
	case "dashboard":
		cmdDashboard()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func cmdDashboard() {
	args := []string{"packages/tui/src/main.ts"}
	// Pass --server flag if provided
	for i := 2; i < len(os.Args); i++ {
		args = append(args, os.Args[i])
	}

	cmd := exec.Command("tsx", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running dashboard: %v\n", err)
		os.Exit(1)
	}
}

func cmdOpen() {
	dir := "."
	if len(os.Args) > 2 {
		dir = os.Args[2]
	}

	worktrees, err := cli.FindAOTWorktrees(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding worktrees: %v\n", err)
		os.Exit(1)
	}

	if len(worktrees) == 0 {
		fmt.Println("No AOT worktrees found.")
		os.Exit(0)
	}

	fmt.Println("AOT Worktrees:")
	for i, wt := range worktrees {
		fmt.Printf("  [%d] %s\n", i+1, wt)
	}

	// If only one, open it directly
	if len(worktrees) == 1 {
		fmt.Printf("\nOpening %s in $EDITOR...\n", worktrees[0])
		if err := cli.OpenInEditor(worktrees[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening editor: %v\n", err)
			os.Exit(1)
		}
	}
}
