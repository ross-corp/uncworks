package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsconfig "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	temporalclient "go.temporal.io/sdk/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/controller"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/softserve"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

func main() {
	if err := run(); err != nil {
		slog.Error("controller failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	temporalHost := envOrDefault("TEMPORAL_HOST", "localhost:7233")
	temporalNamespace := envOrDefault("TEMPORAL_NAMESPACE", "default")
	taskQueue := envOrDefault("TEMPORAL_TASK_QUEUE", aottemporal.TaskQueue)
	litellmBaseURL := envOrDefault("LITELLM_BASE_URL", "http://litellm:4000")

	// Create Temporal client for workflow management
	tc, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  temporalHost,
		Namespace: temporalNamespace,
	})
	if err != nil {
		return fmt.Errorf("create Temporal client: %w", err)
	}
	defer tc.Close()

	metricsAddr := envOrDefault("METRICS_ADDR", ":8090")
	// LeaderElection is required for safe multi-replica deployments.
	// The controller.replicas value in the Helm chart may be >1, so we must
	// elect a single active replica to prevent split-brain reconciliation.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsconfig.Options{
			BindAddress: metricsAddr,
		},
		LeaderElection:          true,
		LeaderElectionID:        "aot-controller-leader",
		LeaderElectionNamespace: os.Getenv("POD_NAMESPACE"),
	})
	if err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	bus := eventbus.NewChannelBus()
	ghTokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_NAME")
	retentionDays := controller.DefaultRetentionDays
	if v := os.Getenv("AOT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retentionDays = n
		}
	}
	// Set up soft-serve client (shared by AgentRun + Project controllers).
	// When SOFT_SERVE_ADDR is unset the client is nil; controllers skip soft-serve
	// operations gracefully (projects remain ConfigRepoReady=false, runs proceed
	// without a config repo).
	var ssClient softserve.RepoManager
	if softServeAddr := os.Getenv("SOFT_SERVE_ADDR"); softServeAddr != "" {
		ssClient = &softserve.Client{
			SSHAddr: softServeAddr,
			KeyPath: envOrDefault("SOFT_SERVE_KEY_PATH", "/etc/soft-serve/id_ed25519"),
		}
	}

	if err = (&controller.AgentRunReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		TemporalClient:        tc,
		TaskQueue:             taskQueue,
		LiteLLMBaseURL:        litellmBaseURL,
		GitHubTokenSecretName: ghTokenSecretName,
		EventBus:              bus,
		RetentionDays:         retentionDays,
		SoftServe:             ssClient,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create controller: %w", err)
	}
	if err = (&controller.ProjectReconciler{
		Client:    mgr.GetClient(),
		SoftServe: ssClient,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create project controller: %w", err)
	}

	// Set up Schedule controller (cron-triggered runs)
	if err = (&controller.ScheduleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create schedule controller: %w", err)
	}

	// Set up ChainRun controller (DAG executor)
	if err = (&controller.ChainRunReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create chain run controller: %w", err)
	}

	ctrl.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("run manager: %w", err)
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
