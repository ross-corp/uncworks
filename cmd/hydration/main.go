package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/uncworks/aot/internal/hydration"
)

func main() {
	config := hydration.ConfigFromEnv()

	if config.RepoURL == "" {
		log.Fatal("AOT_REPO_URL is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	h := hydration.NewHydrator(config, nil)

	log.Printf("Hydrating workspace: repo=%s branch=%s dir=%s", config.RepoURL, config.Branch, config.WorkspaceDir)
	if err := h.Run(ctx); err != nil {
		cancel()
		log.Fatalf("Hydration failed: %v", err)
	}

	cancel()
	log.Printf("Workspace ready at %s", h.WorktreePath())
}
