package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Test 1: Check that "uncworks runs --help" includes "cancel"
	cmd := exec.Command("/tmp/uncworks-test", "runs", "--help")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running uncworks runs --help: %v\n", err)
		os.Exit(1)
	}
	
	if !strings.Contains(string(output), "cancel <id>") {
		fmt.Printf("ERROR: 'uncworks runs --help' doesn't include 'cancel' subcommand\n")
		fmt.Printf("Output:\n%s\n", output)
		os.Exit(1)
	}
	
	fmt.Println("✓ 'uncworks runs --help' includes 'cancel' subcommand")
	
	// Test 2: Check that "uncworks runs cancel --help" shows same as "uncworks cancel --help"
	cmd1 := exec.Command("/tmp/uncworks-test", "runs", "cancel", "--help")
	output1, err1 := cmd1.Output()
	if err1 != nil {
		// Exit code 2 is expected for --help
		if exitErr, ok := err1.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			// This is expected
		} else {
			fmt.Printf("Error running uncworks runs cancel --help: %v\n", err1)
			os.Exit(1)
		}
	}
	
	cmd2 := exec.Command("/tmp/uncworks-test", "cancel", "--help")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		if exitErr, ok := err2.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			// This is expected
		} else {
			fmt.Printf("Error running uncworks cancel --help: %v\n", err2)
			os.Exit(1)
		}
	}
	
	if string(output1) != string(output2) {
		fmt.Printf("ERROR: 'uncworks runs cancel --help' and 'uncworks cancel --help' produce different output\n")
		fmt.Printf("'runs cancel --help':\n%s\n", output1)
		fmt.Printf("'cancel --help':\n%s\n", output2)
		os.Exit(1)
	}
	
	fmt.Println("✓ 'uncworks runs cancel --help' matches 'uncworks cancel --help'")
	
	// Test 3: Check that invalid subcommand shows proper error
	cmd3 := exec.Command("/tmp/uncworks-test", "runs", "invalidcmd")
	output3, err3 := cmd3.CombinedOutput()
	if err3 == nil {
		fmt.Printf("ERROR: 'uncworks runs invalidcmd' should fail but didn't\n")
		os.Exit(1)
	}
	
	if !strings.Contains(string(output3), "unknown subcommand") {
		fmt.Printf("ERROR: 'uncworks runs invalidcmd' doesn't show 'unknown subcommand' error\n")
		fmt.Printf("Output:\n%s\n", output3)
		os.Exit(1)
	}
	
	fmt.Println("✓ Invalid subcommand shows proper error")
	fmt.Println("\nAll tests passed!")
}