package main

import (
	"fmt"
	"os"

	"github.com/uncworks/aot/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: aot <command>\nCommands: open\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "open":
		cmdOpen()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
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
