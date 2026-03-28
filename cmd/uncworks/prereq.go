// prereq.go — prerequisite validation for uncworks CLI.
package main

import (
	"fmt"
	"os/exec"
)

type prereqError struct {
	tool    string
	installURL string
}

func (e *prereqError) Error() string {
	return fmt.Sprintf("%s not found in PATH\nInstall from: %s", e.tool, e.installURL)
}

// checkPrereqs validates that kubectl and helm are available.
func checkPrereqs() error {
	tools := []struct {
		name string
		url  string
	}{
		{"kubectl", "https://kubernetes.io/docs/tasks/tools/"},
		{"helm", "https://helm.sh/docs/intro/install/"},
	}
	for _, t := range tools {
		if _, err := exec.LookPath(t.name); err != nil {
			return &prereqError{tool: t.name, installURL: t.url}
		}
	}
	return nil
}
