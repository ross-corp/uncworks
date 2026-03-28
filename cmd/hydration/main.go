package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/uncworks/aot/internal/hydration"
)

func main() {
	config := hydration.ConfigFromEnv()

	if len(config.Repos) == 0 && config.SpecContent == "" {
		slog.Error("AOT_REPOS or AOT_REPO_URL is required (unless AOT_SPEC_CONTENT is set)")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	h := hydration.NewHydrator(config, nil)

	slog.Info("hydrating workspace", "repos", len(config.Repos), "dir", config.WorkspaceDir)
	if err := h.Run(ctx); err != nil {
		cancel()
		slog.Error("hydration failed", "err", err)
		os.Exit(1)
	}

	cancel()
	slog.Info("workspace ready", "path", h.WorktreePath())
}
