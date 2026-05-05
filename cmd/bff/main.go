package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/uncworks/aot/internal/bff"
)

//go:embed dist/*
var staticFS embed.FS

func initLogger() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func main() {
	initLogger()

	port := envOrDefault("BFF_PORT", "3000")
	apiserverURL := envOrDefault("APISERVER_URL", "http://localhost:50055")
	sessionSecret := envOrDefault("SESSION_SECRET", "dev-secret-change-in-prod")
	authMode := envOrDefault("AUTH_MODE", "open")
	// BFF_ALLOWED_ORIGIN controls the CORS Access-Control-Allow-Origin header.
	// Default to localhost dev URL only. Production deployments MUST set this
	// to the actual frontend origin (e.g. "https://app.example.com").
	allowedOrigin := envOrDefault("BFF_ALLOWED_ORIGIN", "http://localhost:5173")

	proxy := bff.NewProxy(apiserverURL)

	mux := http.NewServeMux()

	// API routes — proxy to apiserver
	proxy.RegisterRoutes(mux)

	// Static files — SPA fallback
	staticRoot, _ := fs.Sub(staticFS, "dist")
	mux.Handle("/", bff.SPAHandler(http.FS(staticRoot)))

	// Middleware chain
	handler := bff.Chain(
		bff.RequestIDMiddleware(),
		bff.CORSMiddleware(allowedOrigin),
		bff.RateLimitMiddleware(100),
	)(mux)

	if authMode != "open" {
		// Session + CSRF middleware are applied when auth is enabled.
		// Session must wrap CSRF so the session cookie exists by the time
		// CSRFMiddleware reads it.
		handler = bff.Chain(
			bff.SessionMiddleware(sessionSecret),
			bff.CSRFMiddleware(),
		)(handler)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	slog.Info("BFF listening",
		"port", port,
		"apiserver", apiserverURL,
		"auth", authMode,
		"allowedOrigin", allowedOrigin,
		"log_level", os.Getenv("LOG_LEVEL"),
	)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("BFF server error", "err", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
