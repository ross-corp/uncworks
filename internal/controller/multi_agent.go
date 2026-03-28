package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// SpawnJunior creates a child AgentRun from a parent (senior) agent.
// The junior is set as owned by the parent so it is GC'd when the parent is deleted.
func SpawnJunior(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, parentRun *aotv1alpha1.AgentRun, task string) (*aotv1alpha1.AgentRun, error) {
	logger := log.FromContext(ctx)

	juniorName := fmt.Sprintf("%s-junior-%d", parentRun.Name, time.Now().UnixMilli()%100000)

	junior := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      juniorName,
			Namespace: parentRun.Namespace,
			Labels: map[string]string{
				"aot.uncworks.io/parent":  parentRun.Name,
				"aot.uncworks.io/role":    "junior",
				"aot.uncworks.io/managed": "true",
			},
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:      parentRun.Spec.Backend,
			Repos:        parentRun.Spec.Repos,
			Prompt:       task,
			DevboxConfig: parentRun.Spec.DevboxConfig,
			TTLSeconds:   parentRun.Spec.TTLSeconds,
			Image:        parentRun.Spec.Image,
		},
	}

	// Set owner reference so the junior AgentRun is garbage collected when the parent is deleted.
	if err := controllerutil.SetControllerReference(parentRun, junior, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference on junior AgentRun")
	}

	logger.Info("Spawning junior agent", "parent", parentRun.Name, "junior", juniorName)
	if err := k8sClient.Create(ctx, junior); err != nil {
		return nil, fmt.Errorf("create junior AgentRun: %w", err)
	}

	return junior, nil
}

// ListJuniors returns all junior AgentRuns for a given parent.
func ListJuniors(ctx context.Context, k8sClient client.Client, parentName, namespace string) ([]aotv1alpha1.AgentRun, error) {
	var list aotv1alpha1.AgentRunList
	if err := k8sClient.List(ctx, &list, client.InNamespace(namespace), client.MatchingLabels{
		"aot.uncworks.io/parent": parentName,
		"aot.uncworks.io/role":   "junior",
	}); err != nil {
		return nil, err
	}
	return list.Items, nil
}
