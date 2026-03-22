package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uncworks/aot/internal/bff"
)

//go:embed dist/*
var staticFS embed.FS

func main() {
	port := envOrDefault("BFF_PORT", "3000")
	apiserverURL := envOrDefault("APISERVER_URL", "http://localhost:50055")
	sessionSecret := envOrDefault("SESSION_SECRET", "dev-secret-change-in-prod")
	authMode := envOrDefault("AUTH_MODE", "open")

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
		bff.CORSMiddleware("*"),
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
		Addr:    ":" + port,
		Handler: handler,
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

	log.Printf("BFF listening on :%s (apiserver: %s, auth: %s)", port, apiserverURL, authMode)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("BFF server error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
