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

	if len(config.Repos) == 0 && config.SpecContent == "" {
		log.Fatal("AOT_REPOS or AOT_REPO_URL is required (unless AOT_SPEC_CONTENT is set)")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	h := hydration.NewHydrator(config, nil)

	log.Printf("Hydrating workspace: %d repo(s), dir=%s", len(config.Repos), config.WorkspaceDir)
	if err := h.Run(ctx); err != nil {
		cancel()
		log.Fatalf("Hydration failed: %v", err)
	}

	cancel()
	log.Printf("Workspace ready at %s", h.WorktreePath())
}
