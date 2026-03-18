package server

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// lookupRunningPod finds the actual running pod name for an AgentRun.
// The CRD status stores the deployment name (e.g. "agentrun-ar-xyz"),
// but the actual pod has a ReplicaSet hash suffix. We find it by listing
// pods matching the agentrun label.
func lookupRunningPod(ctx context.Context, k8sClient runtimeclient.Client, namespace, runID string) (string, error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: namespace,
		Name:      runID,
	}, crd); err != nil {
		return "", err
	}

	deployName := crd.Status.PodName
	if deployName == "" {
		deployName = crd.Status.DeploymentName
	}
	if deployName == "" {
		return "", nil
	}

	// Find the actual pod via the agentrun label.
	var podList corev1.PodList
	if err := k8sClient.List(ctx, &podList,
		runtimeclient.InNamespace(namespace),
		runtimeclient.MatchingLabels{"aot.uncworks.io/agentrun": runID},
	); err != nil {
		return "", fmt.Errorf("list pods for run %s: %w", runID, err)
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	// Fallback: return the CRD value (may work if it's the actual pod name).
	return deployName, nil
}
