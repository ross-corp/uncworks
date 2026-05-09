package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	temporalclient "go.temporal.io/sdk/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	aotgithub "github.com/uncworks/aot/internal/github"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/internal/softserve"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

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
	if os.Getenv("LOG_FORMAT") == "json" || !isTerminal(os.Stdout) {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	initLogger()

	serverCtx, serverCancel := context.WithCancel(context.Background())

	addr := envOrDefault("LISTEN_ADDR", ":50055")
	namespace := envOrDefault("NAMESPACE", "default")

	// Parse allowed CORS origins. Default to localhost dev URLs only.
	allowedOrigins := parseAllowedOrigins(os.Getenv("AOT_ALLOWED_ORIGINS"))

	// API key authentication (optional but strongly recommended for production).
	apiKey := os.Getenv("AOT_API_KEY")
	if apiKey == "" {
		slog.Warn("AOT_API_KEY not set — API server is unauthenticated. Set AOT_API_KEY for production use.")
	}

	// Initialize K8s client
	restConfig := ctrl.GetConfigOrDie()
	k8sClient, err := runtimeclient.New(restConfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		serverCancel()
		slog.Error("failed to create K8s client", "err", err)
		os.Exit(1)
	}
	slog.Info("K8s client initialized", "namespace", namespace)

	bus := eventbus.NewChannelBus()
	svc := server.NewAOTServiceHandler(k8sClient, bus, namespace)

	// Build rate limiter instances from env-var config.
	globalRLCfg, llmRLCfg, webhookRLCfg := rateLimitConfigs()
	globalRL := server.NewRateLimiter(globalRLCfg)
	llmRL := server.NewRateLimiter(llmRLCfg)
	webhookRL := server.NewRateLimiter(webhookRLCfg)
	llmMiddleware := server.RateLimitMiddleware(llmRL)
	webhookMiddleware := server.RateLimitMiddleware(webhookRL)
	if globalRLCfg.Enabled {
		slog.Info("rate limiting enabled",
			"globalRPS", globalRLCfg.RPS, "llmRPS", llmRLCfg.RPS, "webhookRPS", webhookRLCfg.RPS)
	}

	// Connect to Temporal (required for production)
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		slog.Warn("TEMPORAL_HOST not set — agent run creation, cancellation, and human input will fail. Set TEMPORAL_HOST for production use.")
	} else {
		temporalNamespace := envOrDefault("TEMPORAL_NAMESPACE", "default")
		tc, err := temporalclient.Dial(temporalclient.Options{
			HostPort:  temporalHost,
			Namespace: temporalNamespace,
		})
		if err != nil {
			slog.Warn("failed to connect to Temporal", "host", temporalHost, "err", err)
		} else {
			defer tc.Close()
			svc.TemporalClient = tc
			slog.Info("connected to Temporal", "host", temporalHost, "namespace", temporalNamespace)
		}
	}

	// Protovalidate interceptor rejects invalid requests with INVALID_ARGUMENT
	validateInterceptor := validate.NewInterceptor()

	mux := http.NewServeMux()

	// Health check endpoints (unauthenticated)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ready := true
		checks := map[string]string{"k8s": "ok"}

		if svc.TemporalClient == nil {
			ready = false
			checks["temporal"] = "not configured"
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if _, err := svc.TemporalClient.CheckHealth(ctx, &temporalclient.CheckHealthRequest{}); err != nil {
				ready = false
				checks["temporal"] = "unreachable"
			} else {
				checks["temporal"] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		status := "ok"
		if !ready {
			status = "degraded"
		}
		_ = writeJSONResponse(w, map[string]interface{}{"status": status, "checks": checks})
	})

	// Metrics endpoint (exposes expvar metrics)
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		expvar.Do(func(kv expvar.KeyValue) {
			fmt.Fprintf(w, "%s %s\n", kv.Key, kv.Value)
		})
	})

	// Create GitHub token provider from environment
	ghProvider := aotgithub.NewPATProvider(os.Getenv("GITHUB_TOKEN"))

	// Register GitHub integration REST endpoints
	ghClient := server.NewGitHubClient(ghProvider)
	ghClient.RegisterHandlers(mux)

	// Register file explorer REST endpoints (dual-mode: exec or disk)
	fileHandler := server.NewFileHandler(k8sClient, restConfig, namespace)
	fileHandler.RegisterFileHandlers(mux)

	// Register debug pod endpoints
	debugHandler := server.NewDebugHandler(k8sClient, restConfig, namespace)
	debugHandler.RegisterDebugHandlers(mux)

	// Register trace endpoints
	traceHandler := server.NewTraceHandler(k8sClient, restConfig, namespace)
	traceHandler.RegisterTraceHandlers(mux)

	// Register classify endpoint (wrapped with LLM rate limiter)
	classifyHandler := server.NewClassifyRunHandler(k8sClient, namespace, svc.LiteLLMBaseURL)
	classifyHandler.RegisterClassifyHandlersWithMiddleware(mux, llmMiddleware)

	// Register chat streaming endpoint (wrapped with LLM rate limiter)
	chatHandler := server.NewChatHandler(svc.LiteLLMBaseURL)
	chatHandler.RegisterChatHandlersWithMiddleware(mux, llmMiddleware)

	// Register SSE endpoints for real-time graph and trace updates
	sseHandler := server.NewSSEHandler(k8sClient, bus, namespace)
	sseHandler.RegisterSSEHandlers(mux)

	// Register interactive shell WebSocket endpoint
	execHandler := server.NewExecHandler(k8sClient, restConfig, namespace, allowedOrigins)
	execHandler.RegisterExecHandlers(mux)

	// Register archive endpoints
	archiveHandler := &server.ArchiveHandler{K8sClient: k8sClient, Namespace: namespace}
	archiveHandler.RegisterArchiveHandlers(mux)

	// Register lightweight counts endpoint (used by GlobalNav)
	countsHandler := &server.CountsHandler{K8sClient: k8sClient, Namespace: namespace}
	countsHandler.RegisterCountsHandlers(mux)

	// Register project endpoints
	softServeAddr := envOrDefault("SOFT_SERVE_ADDR", "soft-serve.aot.svc:23231")
	softServeKeyPath := envOrDefault("SOFT_SERVE_KEY_PATH", "/etc/soft-serve/id_ed25519")
	var ssClient *softserve.Client
	if _, err := os.Stat(softServeKeyPath); err == nil {
		ssClient = &softserve.Client{SSHAddr: softServeAddr, KeyPath: softServeKeyPath}
	}
	projectHandler := &server.ProjectHandler{K8sClient: k8sClient, Namespace: namespace, SoftServe: ssClient}
	projectHandler.RegisterProjectHandlers(mux)

	// Register chain/schedule/template endpoints
	chainHandler := &server.ChainHandler{K8sClient: k8sClient, Namespace: namespace}
	chainHandler.RegisterChainHandlers(mux)

	// Register GitHub webhook receiver (webhooks use their own HMAC auth, skip API key)
	webhookHandler := server.NewWebhookHandler(serverCtx, k8sClient, namespace, ghProvider)
	mux.Handle("/api/v1/webhooks/github", webhookMiddleware(webhookHandler))

	// Register AOTService handler
	path, handler := apiv1connect.NewAOTServiceHandler(svc,
		connect.WithInterceptors(validateInterceptor),
	)
	mux.Handle(path, handler)

	// Health check (gRPC)
	checker := grpchealth.NewStaticChecker(apiv1connect.AOTServiceName)
	mux.Handle(grpchealth.NewHandler(checker))

	// Reflection for grpcurl compatibility
	reflector := grpcreflect.NewStaticReflector(apiv1connect.AOTServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	// Build middleware chain: CORS → Auth → Global Rate Limit → Handler
	var finalHandler http.Handler = mux
	finalHandler = server.RateLimitMiddleware(globalRL)(finalHandler)
	finalHandler = withAuth(finalHandler, apiKey)
	finalHandler = withCORS(finalHandler, allowedOrigins)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(finalHandler, &http2.Server{}),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		slog.Info("UNCWORKS API server listening", "addr", addr, "protocols", "gRPC+Connect+gRPC-Web")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	// Create context that will be cancelled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	
	// Wait for shutdown signal
	<-ctx.Done()
	
	slog.Info("shutting down UNCWORKS API server...")
	serverCancel()
	
	// Give in-flight requests up to 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	} else {
		slog.Info("shutdown complete")
	}
}

// parseAllowedOrigins parses a comma-separated list of allowed origins.
// If the input is empty, defaults to "*" (permissive dev mode).
// Production deployments should set AOT_ALLOWED_ORIGINS explicitly.
func parseAllowedOrigins(raw string) []string {
	if raw == "" || raw == "*" {
		return []string{"*"}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// isOriginAllowed checks whether an origin is in the allowed list.
func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

// withCORS wraps a handler to allow cross-origin requests from configured origins only.
func withCORS(h http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent")
			w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
			w.Header().Set("Access-Control-Max-Age", "7200")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// withAuth adds bearer token authentication to all non-exempt endpoints.
func withAuth(h http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return h // No auth configured — pass through.
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks, gRPC health, reflection, and webhooks (which use HMAC auth).
		p := r.URL.Path
		if p == "/healthz" || p == "/readyz" ||
			strings.HasPrefix(p, "/grpc.health.") ||
			strings.HasPrefix(p, "/grpc.reflection.") ||
			p == "/api/v1/webhooks/github" {
			h.ServeHTTP(w, r)
			return
		}

		// Check Authorization header (or "token" query param for WebSocket).
		auth := r.Header.Get("Authorization")
		token := ""
		if auth != "" {
			token = strings.TrimPrefix(auth, "Bearer ")
			if token == auth {
				token = "" // Not a Bearer token
			}
		}
		// Fallback: check query parameter (for WebSocket connections where
		// browsers cannot set custom headers).
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		// Use constant-time comparison to prevent timing-oracle attacks.
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// rateLimitConfigs reads rate limit configuration from environment variables.
func rateLimitConfigs() (global, llm, webhook server.RateLimiterConfig) {
	enabled := os.Getenv("RATE_LIMIT_ENABLED") == "true"
	trustProxy := os.Getenv("RATE_LIMIT_TRUST_PROXY") == "true"
	ttl := envIntOrDefault("RATE_LIMIT_TTL_MINUTES", 10)

	global = server.RateLimiterConfig{
		Enabled:    enabled,
		RPS:        envFloatOrDefault("RATE_LIMIT_RPS", 100),
		Burst:      envIntOrDefault("RATE_LIMIT_BURST", 20),
		TTLMinutes: ttl,
		TrustProxy: trustProxy,
	}
	llm = server.RateLimiterConfig{
		Enabled:    enabled,
		RPS:        envFloatOrDefault("RATE_LIMIT_LLM_RPS", 10),
		Burst:      envIntOrDefault("RATE_LIMIT_LLM_BURST", 5),
		TTLMinutes: ttl,
		TrustProxy: trustProxy,
	}
	webhook = server.RateLimiterConfig{
		Enabled:    enabled,
		RPS:        envFloatOrDefault("RATE_LIMIT_WEBHOOK_RPS", 5),
		Burst:      envIntOrDefault("RATE_LIMIT_WEBHOOK_BURST", 2),
		TTLMinutes: ttl,
		TrustProxy: trustProxy,
	}
	return
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			slog.Warn("invalid integer env var, using default", "key", key, "value", v, "default", def)
			return def
		}
		return n
	}
	return def
}

func envFloatOrDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			slog.Warn("invalid float env var, using default", "key", key, "value", v, "default", def)
			return def
		}
		return f
	}
	return def
}

func writeJSONResponse(w http.ResponseWriter, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
