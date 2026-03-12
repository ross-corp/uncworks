package main

import (
	"fmt"
	"log"
	"os"

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
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Controller failed: %v", err)
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
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsconfig.Options{
			BindAddress: metricsAddr,
		},
	})
	if err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	bus := eventbus.NewChannelBus()
	if err = (&controller.AgentRunReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		TemporalClient: tc,
		TaskQueue:      taskQueue,
		LiteLLMBaseURL: litellmBaseURL,
		EventBus:       bus,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create controller: %w", err)
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
