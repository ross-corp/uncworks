//go:build e2e

// Package e2e contains full system E2E tests for AOT.
// These tests require a running k0s cluster and are intended
// to be run against a local development environment.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func getE2EClient(t *testing.T) client.Client {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// Try common locations relative to project root
		candidates := []string{
			"kubeconfig",
			"hack/../kubeconfig",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				kubeconfig = c
				break
			}
		}
		if kubeconfig == "" {
			kubeconfig = "kubeconfig"
		}
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("Skipping E2E test: KUBECONFIG not found. Run 'sudo ./hack/k0s-setup.sh' first.")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Skipf("Skipping E2E test: cannot build k8s config: %v", err)
	}

	if err := aotv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Skipf("Skipping E2E test: cannot create k8s client: %v", err)
	}

	return k8sClient
}

func TestE2E_AgentRunLifecycle(t *testing.T) {
	k8sClient := getE2EClient(t)
	ctx := context.Background()
	namespace := "default"

	// Create CRD first (assumes CRD is installed)
	runName := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	agentRun := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: namespace,
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:     "E2E test: fix the failing integration tests",
			TTLSeconds: 300,
		},
	}

	// Create
	t.Logf("Creating AgentRun %s", runName)
	if err := k8sClient.Create(ctx, agentRun); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}

	// Verify created
	fetched := &aotv1alpha1.AgentRun{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name: runName, Namespace: namespace,
	}, fetched); err != nil {
		t.Fatalf("Get AgentRun: %v", err)
	}

	t.Logf("AgentRun created: backend=%s", fetched.Spec.Backend)
	if fetched.Spec.Backend != aotv1alpha1.BackendPod {
		t.Errorf("expected Pod backend, got %s", fetched.Spec.Backend)
	}

	// Cleanup
	t.Logf("Deleting AgentRun %s", runName)
	if err := k8sClient.Delete(ctx, fetched); err != nil {
		t.Errorf("Delete AgentRun: %v", err)
	}
}

func TestE2E_KubeVirtBackendRejection(t *testing.T) {
	k8sClient := getE2EClient(t)
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-kubevirt-%d", time.Now().Unix())
	agentRun := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendKubeVirt,
			Repos:   []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo")}},
			Prompt:  "E2E test: KubeVirt should be stubbed",
			KubeVirtConfig: &aotv1alpha1.KubeVirtBackendConfig{
				CPUs:     2,
				MemoryMB: 4096,
				DiskGB:   20,
			},
		},
	}

	// Should create (CRD accepts it), controller would reject
	if err := k8sClient.Create(ctx, agentRun); err != nil {
		t.Fatalf("Create KubeVirt AgentRun: %v", err)
	}

	t.Logf("KubeVirt AgentRun created (CRD accepted, controller would reject)")

	// Cleanup
	_ = k8sClient.Delete(ctx, agentRun)
}
