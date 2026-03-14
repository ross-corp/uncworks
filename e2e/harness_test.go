//go:build e2e

package e2e

import (
	"fmt"
	"os"
)

// getSoftServeRepoURL returns the git:// URL for a repo hosted on the local Soft-Serve instance.
func getSoftServeRepoURL(repoName string) string {
	addr := os.Getenv("SOFT_SERVE_ADDR")
	if addr == "" {
		addr = "localhost:9418"
	}
	return fmt.Sprintf("git://%s/%s", addr, repoName)
}
