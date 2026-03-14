package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// Initialize K8s client
	restConfig := ctrl.GetConfigOrDie()
	k8sClient, err := runtimeclient.New(restConfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}
	log.Printf("K8s client initialized (namespace: %s)", namespace)

	bus := eventbus.NewChannelBus()
	svc := server.NewAOTServiceHandler(k8sClient, bus, namespace)

	// Connect to Temporal if configured
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost != "" {
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

	// Register GitHub integration REST endpoints
	ghClient := server.NewGitHubClient()
	ghClient.RegisterHandlers(mux)

	// Register GitHub webhook receiver
	webhookHandler := server.NewWebhookHandler(k8sClient, namespace)
	mux.Handle("/api/v1/webhooks/github", webhookHandler)

	// Register AOTService handler
	path, handler := apiv1connect.NewAOTServiceHandler(svc,
		connect.WithInterceptors(validateInterceptor),
	)
	mux.Handle(path, handler)

	// Health check
	checker := grpchealth.NewStaticChecker(apiv1connect.AOTServiceName)
	mux.Handle(grpchealth.NewHandler(checker))

	// Reflection for grpcurl compatibility
	reflector := grpcreflect.NewStaticReflector(apiv1connect.AOTServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	httpServer := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(withCORS(mux), &http2.Server{}),
	}

	go func() {
		log.Printf("AOT API server listening on %s (gRPC + Connect + gRPC-Web)", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// withCORS wraps a handler to allow cross-origin requests from web UIs.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent")
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

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
