// Package controller implements the Kubernetes controller for AgentRun CRDs.
package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

const (
	defaultAgentImage = "ghcr.io/uncworks/aot-agent:latest"
	sidecarImage      = "ghcr.io/uncworks/aot-sidecar:latest"
	initImage         = "ghcr.io/uncworks/aot-init:latest"
)

// AgentRunReconciler reconciles AgentRun objects.
type AgentRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aot.uncworks.io,resources=agentruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aot.uncworks.io,resources=agentruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles changes to AgentRun resources.
func (r *AgentRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agentRun aotv1alpha1.AgentRun
	if err := r.Get(ctx, req.NamespacedName, &agentRun); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle based on backend type
	switch agentRun.Spec.Backend {
	case aotv1alpha1.BackendPod:
		return r.reconcilePod(ctx, &agentRun)
	case aotv1alpha1.BackendKubeVirt:
		return r.handleNotImplemented(ctx, &agentRun, "KubeVirt")
	case aotv1alpha1.BackendExternal:
		return r.handleNotImplemented(ctx, &agentRun, "External")
	default:
		logger.Error(fmt.Errorf("unknown backend: %s", agentRun.Spec.Backend), "unsupported backend")
		return ctrl.Result{}, nil
	}
}

func (r *AgentRunReconciler) reconcilePod(ctx context.Context, agentRun *aotv1alpha1.AgentRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if pod already exists
	podName := fmt.Sprintf("agentrun-%s", agentRun.Name)
	var existingPod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{
		Namespace: agentRun.Namespace,
		Name:      podName,
	}, &existingPod)

	if err == nil {
		// Pod exists, sync status
		return r.syncPodStatus(ctx, agentRun, &existingPod)
	}

	if !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	// Check TTL
	if agentRun.Status.Phase == aotv1alpha1.AgentRunPhaseRunning && agentRun.Status.StartedAt != nil {
		elapsed := time.Since(agentRun.Status.StartedAt.Time)
		ttl := time.Duration(agentRun.Spec.TTLSeconds) * time.Second
		if elapsed > ttl {
			logger.Info("AgentRun exceeded TTL, marking failed")
			agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
			agentRun.Status.Message = "Exceeded TTL"
			now := metav1.Now()
			agentRun.Status.CompletedAt = &now
			return ctrl.Result{}, r.Status().Update(ctx, agentRun)
		}
	}

	// Create the pod
	pod := r.buildAgentPod(agentRun, podName)

	if err := ctrl.SetControllerReference(agentRun, pod, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Creating Agent Pod", "pod", podName)
	if err := r.Create(ctx, pod); err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	now := metav1.Now()
	agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	agentRun.Status.PodName = podName
	agentRun.Status.StartedAt = &now
	agentRun.Status.Message = "Pod created"
	return ctrl.Result{RequeueAfter: 30 * time.Second}, r.Status().Update(ctx, agentRun)
}

func (r *AgentRunReconciler) buildAgentPod(agentRun *aotv1alpha1.AgentRun, podName string) *corev1.Pod {
	image := agentRun.Spec.Image
	if image == "" {
		image = defaultAgentImage
	}

	envVars := []corev1.EnvVar{
		{Name: "AOT_AGENT_RUN_ID", Value: agentRun.Name},
		{Name: "AOT_REPO_URL", Value: agentRun.Spec.RepoURL},
		{Name: "AOT_BRANCH", Value: agentRun.Spec.Branch},
		{Name: "AOT_PROMPT", Value: agentRun.Spec.Prompt},
	}
	for k, v := range agentRun.Spec.EnvVars {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: agentRun.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "aot-agent",
				"app.kubernetes.io/managed-by": "aot-controller",
				"aot.uncworks.io/agentrun":     agentRun.Name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				{
					Name:  "hydration",
					Image: initImage,
					Env:   envVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "agent",
					Image: image,
					Env:   envVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
				},
				{
					Name:  "rpc-gateway",
					Image: sidecarImage,
					Ports: []corev1.ContainerPort{
						{Name: "grpc", ContainerPort: 50052, Protocol: corev1.ProtocolTCP},
					},
					Env: []corev1.EnvVar{
						{Name: "AOT_AGENT_RUN_ID", Value: agentRun.Name},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func (r *AgentRunReconciler) syncPodStatus(ctx context.Context, agentRun *aotv1alpha1.AgentRun, pod *corev1.Pod) (ctrl.Result, error) {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		if agentRun.Status.Phase != aotv1alpha1.AgentRunPhaseSucceeded {
			agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
			agentRun.Status.Message = "Agent completed successfully"
			now := metav1.Now()
			agentRun.Status.CompletedAt = &now
			return ctrl.Result{}, r.Status().Update(ctx, agentRun)
		}
	case corev1.PodFailed:
		if agentRun.Status.Phase != aotv1alpha1.AgentRunPhaseFailed {
			agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
			agentRun.Status.Message = "Agent pod failed"
			now := metav1.Now()
			agentRun.Status.CompletedAt = &now
			return ctrl.Result{}, r.Status().Update(ctx, agentRun)
		}
	case corev1.PodRunning:
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *AgentRunReconciler) handleNotImplemented(ctx context.Context, agentRun *aotv1alpha1.AgentRun, backend string) (ctrl.Result, error) {
	agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
	agentRun.Status.Message = fmt.Sprintf("%s backend is not yet implemented", backend)
	return ctrl.Result{}, r.Status().Update(ctx, agentRun)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aotv1alpha1.AgentRun{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
