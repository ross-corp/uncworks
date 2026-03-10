// Package testutil provides shared test helpers for the AOT project.
package testutil

import (
	"os"
	"os/exec"
	"strings"
)

// EnsureEnvtestAssets sets KUBEBUILDER_ASSETS if not already set,
// by running setup-envtest to locate the binaries.
func EnsureEnvtestAssets() {
	if os.Getenv("KUBEBUILDER_ASSETS") != "" {
		return
	}

	// Try common locations for setup-envtest
	candidates := []string{
		"setup-envtest",
		os.ExpandEnv("$HOME/go/bin/setup-envtest"),
		os.ExpandEnv("$GOPATH/bin/setup-envtest"),
	}

	for _, bin := range candidates {
		out, err := exec.Command(bin, "use", "--print", "path").Output()
		if err == nil {
			path := strings.TrimSpace(string(out))
			if path != "" {
				_ = os.Setenv("KUBEBUILDER_ASSETS", path)
				return
			}
		}
	}
}
