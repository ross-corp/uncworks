// Binary temporal-worker runs the Temporal worker that executes AgentRun workflows
// and activities. It connects to the Temporal Frontend service and registers all
// workflow and activity implementations.
package main

import (
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(scheme))
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func run() error {
	temporalHost := envOrDefault("TEMPORAL_HOST", "localhost:7233")
	temporalNamespace := envOrDefault("TEMPORAL_NAMESPACE", "default")
	taskQueue := envOrDefault("TEMPORAL_TASK_QUEUE", aottemporal.TaskQueue)

	log.Printf("Connecting to Temporal at %s (namespace: %s, queue: %s)",
		temporalHost, temporalNamespace, taskQueue)

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

	// Create activities with dependencies
	activities := &aottemporal.Activities{
		K8sClient: k8sClient,
	}

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)

	// Register activities
	w.RegisterActivity(activities)

	// Start worker (blocks until interrupted)
	log.Printf("Starting Temporal worker on queue %s", taskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("run worker: %w", err)
	}

	log.Println("Worker stopped")
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
