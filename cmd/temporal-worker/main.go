// Binary temporal-worker runs the Temporal worker that executes AgentRun workflows
// and activities. It connects to the Temporal Frontend service and registers all
// workflow and activity implementations.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/brain"
	"github.com/uncworks/aot/internal/embeddings"
	aotgithub "github.com/uncworks/aot/internal/github"
	"github.com/uncworks/aot/internal/litellm"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

func main() {
	if err := run(); err != nil {
		slog.Error("worker failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// TEMPORAL_HOST: required in production; defaults to localhost:7233 for local dev only.
	temporalHost := envOrDefault("TEMPORAL_HOST", "localhost:7233")
	if os.Getenv("TEMPORAL_HOST") == "" {
		slog.Warn("TEMPORAL_HOST not set — using localhost:7233; set TEMPORAL_HOST for production")
	}
	// TEMPORAL_NAMESPACE: optional; defaults to "default".
	temporalNamespace := envOrDefault("TEMPORAL_NAMESPACE", "default")
	// TEMPORAL_TASK_QUEUE: optional; defaults to the compiled-in task queue constant.
	taskQueue := envOrDefault("TEMPORAL_TASK_QUEUE", aottemporal.TaskQueue)

	slog.Info("connecting to Temporal", "host", temporalHost, "namespace", temporalNamespace, "queue", taskQueue)

	// Initialize controller-runtime K8s client for pod management activities
	restConfig := ctrl.GetConfigOrDie()
	k8sClient, err := runtimeclient.New(restConfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("create K8s client: %w", err)
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: temporalNamespace,
	})
	if err != nil {
		return fmt.Errorf("create Temporal client: %w", err)
	}
	defer c.Close()

	// LITELLM_BASE_URL: optional; base URL of the LiteLLM proxy.
	// LITELLM_MASTER_KEY: optional; master key for the LiteLLM proxy.
	// Both must be set for the LiteLLM client to be active; otherwise LiteLLM
	// activities (model provisioning) are no-ops.
	var litellmClient *litellm.Client
	litellmBaseURL := os.Getenv("LITELLM_BASE_URL")
	litellmMasterKey := os.Getenv("LITELLM_MASTER_KEY")
	if litellmBaseURL != "" && litellmMasterKey != "" {
		litellmClient = litellm.NewClient(litellmBaseURL, litellmMasterKey)
		slog.Info("LiteLLM client configured", "baseURL", litellmBaseURL)
	} else if litellmBaseURL != "" || litellmMasterKey != "" {
		slog.Warn("LiteLLM client not configured: both LITELLM_BASE_URL and LITELLM_MASTER_KEY must be set",
			"baseURLSet", litellmBaseURL != "", "masterKeySet", litellmMasterKey != "")
	}

	// AOT_AGENT_IMAGE, AOT_SIDECAR_IMAGE, AOT_INIT_IMAGE: optional; container image refs for
	// agent pods. Defaults to ghcr.io/uncworks/aot-{agent,sidecar,init}:latest.
	// Note: actual values are read by internal/temporal/activities.go at package init time.
	slog.Info("agent images",
		"agent", envOrDefault("AOT_AGENT_IMAGE", "ghcr.io/uncworks/aot-agent:latest"),
		"sidecar", envOrDefault("AOT_SIDECAR_IMAGE", "ghcr.io/uncworks/aot-sidecar:latest"),
		"init", envOrDefault("AOT_INIT_IMAGE", "ghcr.io/uncworks/aot-init:latest"),
	)

	// GITHUB_TOKEN: optional; PAT for GitHub API calls made inside workflows.
	// GITHUB_TOKEN_SECRET_NAME: optional; k8s Secret name holding the token for agent pods.
	ghProvider := aotgithub.NewPATProvider(os.Getenv("GITHUB_TOKEN"))
	ghTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_NAME")

	// Create activities with dependencies
	activities := &aottemporal.Activities{
		K8sClient:             k8sClient,
		LiteLLMClient:         litellmClient,
		HTTPClient:            &http.Client{Timeout: 30 * time.Second},
		GitHubProvider:        ghProvider,
		GitHubTokenSecretName: ghTokenSecretName,
		RESTConfig:            restConfig,
	}

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterWorkflow(aottemporal.SpawnJuniorWorkflow)

	// Register activities
	w.RegisterActivity(activities)

	// Register knowledge activities (context hydration, run data persistence, embedding).
	// Wire brain store and embedder when BRAIN_DATABASE_URL and EMBEDDER_BASE_URL are set.
	knowledgeActivities := &aottemporal.KnowledgeActivities{}
	if brainDSN := os.Getenv("BRAIN_DATABASE_URL"); brainDSN != "" {
		brainPool, err := brain.NewPool(context.Background(), brainDSN)
		if err != nil {
			slog.Warn("failed to connect to brain DB — knowledge activities will be no-ops", "err", err)
		} else {
			defer brainPool.Close()
			store := brain.NewStore(brainPool)
			if err := store.Migrate(context.Background()); err != nil {
				slog.Warn("brain DB migration failed — knowledge activities will be no-ops", "err", err)
			} else {
				knowledgeActivities.BrainStore = store
				slog.Info("brain store connected and migrated")
			}
		}
	} else {
		slog.Warn("BRAIN_DATABASE_URL not set — knowledge activities (persist/embed/hydrate) are no-ops")
	}
	if embedderURL := os.Getenv("EMBEDDER_BASE_URL"); embedderURL != "" && knowledgeActivities.BrainStore != nil {
		model := os.Getenv("EMBEDDER_MODEL") // empty string falls back to DefaultModel
		knowledgeActivities.Embedder = embeddings.NewEmbedder(embedderURL, model, &http.Client{Timeout: 30 * time.Second})
		slog.Info("embedder configured", "baseURL", embedderURL)
	}
	w.RegisterActivity(knowledgeActivities)

	// Health probe server — liveness/readiness check Temporal connectivity.
	healthAddr := envOrDefault("HEALTH_PROBE_ADDR", ":8091")
	mux := http.NewServeMux()
	checkTemporal := func(w http.ResponseWriter, _ *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := c.CheckHealth(ctx, &client.CheckHealthRequest{}); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc("/healthz", checkTemporal)
	mux.HandleFunc("/readyz", checkTemporal)
	healthSrv := &http.Server{Addr: healthAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("health probe server stopped", "err", err)
		}
	}()

	// Start worker (blocks until interrupted)
	slog.Info("starting Temporal worker", "queue", taskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("run worker: %w", err)
	}

	slog.Info("worker stopped")
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
