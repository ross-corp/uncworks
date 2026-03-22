package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
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
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

func main() {
	addr := envOrDefault("LISTEN_ADDR", ":50055")
	namespace := envOrDefault("NAMESPACE", "default")

	// Parse allowed CORS origins. Default to localhost dev URLs only.
	allowedOrigins := parseAllowedOrigins(os.Getenv("AOT_ALLOWED_ORIGINS"))

	// API key authentication (optional but strongly recommended for production).
	apiKey := os.Getenv("AOT_API_KEY")
	if apiKey == "" {
		log.Println("WARNING: AOT_API_KEY not set — API server is unauthenticated. Set AOT_API_KEY for production use.")
	}

	// Initialize K8s client
	restConfig := ctrl.GetConfigOrDie()
	k8sClient, err := runtimeclient.New(restConfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}
	log.Printf("K8s client initialized (namespace: %s)", namespace)

	bus := eventbus.NewChannelBus()
	svc := server.NewAOTServiceHandler(k8sClient, bus, namespace)

	// Connect to Temporal (required for production)
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		log.Println("WARNING: TEMPORAL_HOST not set — agent run creation, cancellation, and human input will fail. Set TEMPORAL_HOST for production use.")
	} else {
		temporalNamespace := envOrDefault("TEMPORAL_NAMESPACE", "default")
		tc, err := temporalclient.Dial(temporalclient.Options{
			HostPort:  temporalHost,
			Namespace: temporalNamespace,
		})
		if err != nil {
			log.Printf("WARNING: Failed to connect to Temporal at %s: %v", temporalHost, err)
		} else {
			defer tc.Close()
			svc.TemporalClient = tc
			log.Printf("Connected to Temporal at %s (namespace: %s)", temporalHost, temporalNamespace)
		}
	}

	// Protovalidate interceptor rejects invalid requests with INVALID_ARGUMENT
	validateInterceptor := validate.NewInterceptor()

	mux := http.NewServeMux()

	// Health check endpoints (unauthenticated)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		ready := true
		checks := map[string]string{"k8s": "ok"}

		if svc.TemporalClient == nil {
			ready = false
			checks["temporal"] = "not connected"
		} else {
			checks["temporal"] = "ok"
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

	// Register classify endpoint
	classifyHandler := server.NewClassifyRunHandler(k8sClient, namespace, svc.LiteLLMBaseURL)
	classifyHandler.RegisterClassifyHandlers(mux)

	// Register SSE endpoints for real-time graph and trace updates
	sseHandler := server.NewSSEHandler(k8sClient, bus, namespace)
	sseHandler.RegisterSSEHandlers(mux)

	// Register interactive shell WebSocket endpoint
	execHandler := server.NewExecHandler(k8sClient, restConfig, namespace, allowedOrigins)
	execHandler.RegisterExecHandlers(mux)

	// Register GitHub webhook receiver (webhooks use their own HMAC auth, skip API key)
	webhookHandler := server.NewWebhookHandler(k8sClient, namespace, ghProvider)
	mux.Handle("/api/v1/webhooks/github", webhookHandler)

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

	// Build middleware chain: CORS → Auth → Rate Limit → Handler
	var finalHandler http.Handler = mux
	finalHandler = withRateLimit(finalHandler)
	finalHandler = withAuth(finalHandler, apiKey)
	finalHandler = withCORS(finalHandler, allowedOrigins)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(finalHandler, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		log.Printf("UNCWORKS API server listening on %s (gRPC + Connect + gRPC-Web)", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
		if token != apiKey {
			http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// rateLimiter is a simple per-IP token bucket rate limiter.
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*tokenBucket
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

const (
	rateLimit  = 60.0  // requests per second
	burstLimit = 120.0 // max burst
)

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{clients: make(map[string]*tokenBucket)}
	// Clean up stale entries periodically.
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.mu.Lock()
			now := time.Now()
			for ip, tb := range rl.clients {
				if now.Sub(tb.lastRefill) > 10*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tb, ok := rl.clients[ip]
	if !ok {
		tb = &tokenBucket{tokens: burstLimit, lastRefill: time.Now()}
		rl.clients[ip] = tb
	}

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * rateLimit
	if tb.tokens > burstLimit {
		tb.tokens = burstLimit
	}
	tb.lastRefill = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// withRateLimit applies per-IP rate limiting.
func withRateLimit(h http.Handler) http.Handler {
	rl := newRateLimiter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health checks.
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			h.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.SplitN(fwd, ",", 2)[0]
		}
		ip = strings.TrimSpace(ip)

		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		h.ServeHTTP(w, r)
	})
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
