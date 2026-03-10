// Package temporal implements Temporal workflows and activities for agent lifecycle orchestration.
package temporal

import (
	"context"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

const (
	defaultAgentImage = "ghcr.io/uncworks/aot-agent:latest"
	sidecarImage      = "ghcr.io/uncworks/aot-sidecar:latest"
	initImage         = "ghcr.io/uncworks/aot-init:latest"
	sidecarPort       = 50052
)

// Activities holds the dependencies needed by Temporal activity implementations.
type Activities struct {
	K8sClient client.Client
}

// CreateAgentPodInput contains the parameters for creating an agent pod.
type CreateAgentPodInput struct {
	Name         string
	Namespace    string
	AgentRunName string
	RepoURL      string
	Branch       string
	Prompt       string
	DevboxConfig string
	Image        string
	EnvVars      map[string]string
}

// CreateAgentPodOutput contains the result of creating an agent pod.
type CreateAgentPodOutput struct {
	PodName string
}

// CreateAgentPod creates the agent pod with init container, agent, and sidecar.
func (a *Activities) CreateAgentPod(ctx context.Context, input CreateAgentPodInput) (*CreateAgentPodOutput, error) {
	pod := BuildAgentPod(input)

	if err := a.K8sClient.Create(ctx, pod); err != nil {
		return nil, fmt.Errorf("create agent pod: %w", err)
	}

	return &CreateAgentPodOutput{PodName: pod.Name}, nil
}

// WaitForHydrationInput contains the parameters for waiting on hydration.
type WaitForHydrationInput struct {
	PodName   string
	Namespace string
}

// WaitForHydration polls the pod's init container status until hydration completes.
func (a *Activities) WaitForHydration(ctx context.Context, input WaitForHydrationInput) error {
	for {
		var pod corev1.Pod
		if err := a.K8sClient.Get(ctx, client.ObjectKey{
			Namespace: input.Namespace,
			Name:      input.PodName,
		}, &pod); err != nil {
			return fmt.Errorf("get pod: %w", err)
		}

		for _, initStatus := range pod.Status.InitContainerStatuses {
			if initStatus.Name == "hydration" {
				if initStatus.State.Terminated != nil {
					if initStatus.State.Terminated.ExitCode == 0 {
						return nil
					}
					return fmt.Errorf("hydration failed with exit code %d: %s",
						initStatus.State.Terminated.ExitCode,
						initStatus.State.Terminated.Message)
				}
			}
		}

		// Pod is running (past init) — hydration succeeded
		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return fmt.Errorf("pod failed before hydration completed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// SidecarRPCInput contains the parameters for calling the sidecar.
type SidecarRPCInput struct {
	PodName   string
	Namespace string
}

// StartAgentInput contains the parameters for starting the agent.
type StartAgentInput struct {
	PodName   string
	Namespace string
	Prompt    string
}

// StartAgent calls the sidecar StartAgent RPC.
func (a *Activities) StartAgent(ctx context.Context, input StartAgentInput) error {
	sidecarClient := a.sidecarClient(input.PodName, input.Namespace)

	resp, err := sidecarClient.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
		Prompt: input.Prompt,
	}))
	if err != nil {
		return fmt.Errorf("start agent RPC: %w", err)
	}
	if !resp.Msg.Started {
		return fmt.Errorf("agent did not start")
	}
	return nil
}

// GetAgentStatusInput contains the parameters for getting agent status.
type GetAgentStatusInput struct {
	PodName   string
	Namespace string
}

// GetAgentStatusOutput contains the agent's current status.
type GetAgentStatusOutput struct {
	State string
	Error string
}

// GetAgentStatus calls the sidecar GetStatus RPC.
func (a *Activities) GetAgentStatus(ctx context.Context, input GetAgentStatusInput) (*GetAgentStatusOutput, error) {
	sidecarClient := a.sidecarClient(input.PodName, input.Namespace)

	resp, err := sidecarClient.GetStatus(ctx, connect.NewRequest(&agentv1.GetStatusRequest{}))
	if err != nil {
		return nil, fmt.Errorf("get status RPC: %w", err)
	}

	status := resp.Msg
	return &GetAgentStatusOutput{
		State: status.State.String(),
		Error: status.Error,
	}, nil
}

// ForwardHumanInputInput contains the parameters for forwarding human input.
type ForwardHumanInputInput struct {
	PodName   string
	Namespace string
	Input     string
}

// ForwardHumanInput calls the sidecar SendInput RPC.
func (a *Activities) ForwardHumanInput(ctx context.Context, input ForwardHumanInputInput) error {
	sidecarClient := a.sidecarClient(input.PodName, input.Namespace)

	_, err := sidecarClient.SendInput(ctx, connect.NewRequest(&agentv1.SendInputRequest{
		Data: []byte(input.Input),
	}))
	if err != nil {
		return fmt.Errorf("send input RPC: %w", err)
	}
	return nil
}

// StopAgentInput contains the parameters for stopping the agent.
type StopAgentInput struct {
	PodName   string
	Namespace string
	Force     bool
}

// StopAgent calls the sidecar StopAgent RPC.
func (a *Activities) StopAgent(ctx context.Context, input StopAgentInput) error {
	sidecarClient := a.sidecarClient(input.PodName, input.Namespace)

	_, err := sidecarClient.StopAgent(ctx, connect.NewRequest(&agentv1.StopAgentRequest{
		Force: input.Force,
	}))
	if err != nil {
		return fmt.Errorf("stop agent RPC: %w", err)
	}
	return nil
}

// CleanupPodInput contains the parameters for deleting an agent pod.
type CleanupPodInput struct {
	PodName   string
	Namespace string
}

// CleanupPod deletes the agent pod.
func (a *Activities) CleanupPod(ctx context.Context, input CleanupPodInput) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.PodName,
			Namespace: input.Namespace,
		},
	}

	if err := a.K8sClient.Delete(ctx, pod); err != nil {
		if errors.IsNotFound(err) {
			return nil // Already gone
		}
		return fmt.Errorf("delete pod: %w", err)
	}
	return nil
}

// BuildAgentPod creates a pod spec for an agent run. This is a shared function
// used by both the Temporal activity and potentially direct controller usage.
func BuildAgentPod(input CreateAgentPodInput) *corev1.Pod {
	image := input.Image
	if image == "" {
		image = defaultAgentImage
	}

	envVars := []corev1.EnvVar{
		{Name: "AOT_AGENT_RUN_ID", Value: input.AgentRunName},
		{Name: "AOT_REPO_URL", Value: input.RepoURL},
		{Name: "AOT_BRANCH", Value: input.Branch},
		{Name: "AOT_PROMPT", Value: input.Prompt},
	}
	if input.DevboxConfig != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "AOT_DEVBOX_CONFIG", Value: input.DevboxConfig})
	}
	for k, v := range input.EnvVars {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "aot-agent",
				"app.kubernetes.io/managed-by": "aot-controller",
				"aot.uncworks.io/agentrun":     input.AgentRunName,
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
						{Name: "grpc", ContainerPort: sidecarPort, Protocol: corev1.ProtocolTCP},
					},
					Env: []corev1.EnvVar{
						{Name: "AOT_AGENT_RUN_ID", Value: input.AgentRunName},
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

// sidecarClient creates a ConnectRPC client for the sidecar running in the given pod.
// In a real cluster, this resolves to the pod's IP. For simplicity, we use the pod
// DNS name within the cluster: <pod-name>.<namespace>.svc.cluster.local.
func (a *Activities) sidecarClient(podName, namespace string) agentv1connect.AgentSidecarServiceClient {
	addr := fmt.Sprintf("http://%s.%s:%d", podName, namespace, sidecarPort)
	return agentv1connect.NewAgentSidecarServiceClient(
		&http.Client{},
		addr,
	)
}
