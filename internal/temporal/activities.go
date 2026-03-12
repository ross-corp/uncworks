// Package temporal implements Temporal workflows and activities for agent lifecycle orchestration.
package temporal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
	"github.com/uncworks/aot/internal/litellm"
)

const sidecarPort = 50052

// Image names — configurable via environment variables for local development.
var (
	agentImage   = envOrDefault("AOT_AGENT_IMAGE", "ghcr.io/uncworks/aot-agent:latest")
	sidecarImage = envOrDefault("AOT_SIDECAR_IMAGE", "ghcr.io/uncworks/aot-sidecar:latest")
	initImage    = envOrDefault("AOT_INIT_IMAGE", "ghcr.io/uncworks/aot-init:latest")
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// imagePullPolicy returns Never for local images (no registry prefix), Always otherwise.
func imagePullPolicy(image string) corev1.PullPolicy {
	if !strings.Contains(image, "/") {
		return corev1.PullNever
	}
	return corev1.PullAlways
}

// Activities holds the dependencies needed by Temporal activity implementations.
type Activities struct {
	K8sClient     client.Client
	LiteLLMClient *litellm.Client
	HTTPClient    *http.Client
}

// CreateAgentPodInput contains the parameters for creating an agent pod.
type CreateAgentPodInput struct {
	Name           string
	Namespace      string
	AgentRunName   string
	RepoURL        string
	Branch         string
	Prompt         string
	DevboxConfig   string
	Image          string
	EnvVars        map[string]string
	LLMKey         string
	LiteLLMBaseURL string
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

// WaitForHydrationOutput contains the result of waiting for hydration.
type WaitForHydrationOutput struct {
	PodIP string
}

// WaitForHydration polls the pod's init container status until hydration completes.
// Returns the pod IP so subsequent activities can reach the sidecar directly.
func (a *Activities) WaitForHydration(ctx context.Context, input WaitForHydrationInput) (*WaitForHydrationOutput, error) {
	for {
		var pod corev1.Pod
		if err := a.K8sClient.Get(ctx, client.ObjectKey{
			Namespace: input.Namespace,
			Name:      input.PodName,
		}, &pod); err != nil {
			return nil, fmt.Errorf("get pod: %w", err)
		}

		for _, initStatus := range pod.Status.InitContainerStatuses {
			if initStatus.Name == "hydration" {
				if initStatus.State.Terminated != nil {
					if initStatus.State.Terminated.ExitCode == 0 {
						return &WaitForHydrationOutput{PodIP: pod.Status.PodIP}, nil
					}
					return nil, fmt.Errorf("hydration failed with exit code %d: %s",
						initStatus.State.Terminated.ExitCode,
						initStatus.State.Terminated.Message)
				}
			}
		}

		// Pod is running (past init) — hydration succeeded
		if pod.Status.Phase == corev1.PodRunning {
			return &WaitForHydrationOutput{PodIP: pod.Status.PodIP}, nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return nil, fmt.Errorf("pod failed before hydration completed")
		}

		activity.RecordHeartbeat(ctx, fmt.Sprintf("waiting for hydration: pod %s", input.PodName))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// SidecarRPCInput contains the parameters for calling the sidecar.
type SidecarRPCInput struct {
	PodName   string
	Namespace string
	PodIP     string
}

// StartAgentInput contains the parameters for starting the agent.
type StartAgentInput struct {
	PodName   string
	Namespace string
	PodIP     string
	Prompt    string
}

// StartAgent calls the sidecar StartAgent RPC, retrying until the sidecar is ready.
func (a *Activities) StartAgent(ctx context.Context, input StartAgentInput) error {
	sc := a.sidecarClient(input.PodIP)

	// Retry until sidecar is ready (it may still be starting)
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("waiting for sidecar readiness: attempt %d", attempt+1))

		resp, err := sc.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
			Prompt: input.Prompt,
		}))
		if err == nil {
			if !resp.Msg.Started {
				return fmt.Errorf("agent did not start")
			}
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("start agent RPC (sidecar not ready after 60s): %w", lastErr)
}

// GetAgentStatusInput contains the parameters for getting agent status.
type GetAgentStatusInput struct {
	PodName   string
	Namespace string
	PodIP     string
}

// GetAgentStatusOutput contains the agent's current status.
type GetAgentStatusOutput struct {
	State string
	Error string
}

// GetAgentStatus calls the sidecar GetStatus RPC.
func (a *Activities) GetAgentStatus(ctx context.Context, input GetAgentStatusInput) (*GetAgentStatusOutput, error) {
	sidecarClient := a.sidecarClient(input.PodIP)

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
	PodIP     string
	Input     string
}

// ForwardHumanInput calls the sidecar SendInput RPC.
func (a *Activities) ForwardHumanInput(ctx context.Context, input ForwardHumanInputInput) error {
	sidecarClient := a.sidecarClient(input.PodIP)

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
	PodIP     string
	Force     bool
}

// StopAgent calls the sidecar StopAgent RPC.
func (a *Activities) StopAgent(ctx context.Context, input StopAgentInput) error {
	sidecarClient := a.sidecarClient(input.PodIP)

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
		image = agentImage
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

	// LiteLLM gateway env vars — inject into agent container only
	var llmEnvVars []corev1.EnvVar
	if input.LiteLLMBaseURL != "" {
		llmEnvVars = append(llmEnvVars, corev1.EnvVar{
			Name:  "OPENAI_BASE_URL",
			Value: input.LiteLLMBaseURL + "/v1",
		})
	}
	if input.LLMKey != "" {
		llmEnvVars = append(llmEnvVars, corev1.EnvVar{
			Name:  "OPENAI_API_KEY",
			Value: input.LLMKey,
		})
	}
	agentEnvVars := append(envVars, llmEnvVars...) //nolint:gocritic // intentional new slice

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
					Name:            "hydration",
					Image:           initImage,
					ImagePullPolicy: imagePullPolicy(initImage),
					Env:             envVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            "agent",
					Image:           image,
					ImagePullPolicy: imagePullPolicy(image),
					Env:             agentEnvVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
				},
				{
					Name:            "rpc-gateway",
					Image:           sidecarImage,
					ImagePullPolicy: imagePullPolicy(sidecarImage),
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

// ProvisionLLMKeyInput contains the parameters for provisioning a LiteLLM virtual key.
type ProvisionLLMKeyInput struct {
	AgentRunName string
	Namespace    string
	ModelTier    string
	MaxBudget    float64
}

// ProvisionLLMKeyOutput contains the provisioned virtual key.
type ProvisionLLMKeyOutput struct {
	Key string
}

// modelsForTier returns the LiteLLM model names a tier is authorized to use.
func modelsForTier(tier string) []string {
	switch tier {
	case "premium":
		return []string{"default", "default-cloud", "premium"}
	case "default-cloud":
		return []string{"default", "default-cloud"}
	default: // "default" or empty
		return []string{"default", "default-cloud"}
	}
}

// ProvisionLLMKey provisions a LiteLLM virtual key for an agent run.
func (a *Activities) ProvisionLLMKey(ctx context.Context, input ProvisionLLMKeyInput) (*ProvisionLLMKeyOutput, error) {
	if a.LiteLLMClient == nil {
		return &ProvisionLLMKeyOutput{}, nil
	}

	tier := input.ModelTier
	if tier == "" {
		tier = "default"
	}

	budget := input.MaxBudget
	if budget <= 0 {
		budget = 1.0 // Default $1 budget
	}

	resp, err := a.LiteLLMClient.GenerateKey(ctx, litellm.GenerateKeyRequest{
		KeyAlias:  fmt.Sprintf("aot-%s-%s", input.Namespace, input.AgentRunName),
		MaxBudget: &budget,
		Models:    modelsForTier(tier),
		Metadata: map[string]string{
			"agent_run": input.AgentRunName,
			"namespace": input.Namespace,
			"tier":      tier,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("provision LLM key: %w", err)
	}

	return &ProvisionLLMKeyOutput{Key: resp.Key}, nil
}

// RevokeLLMKeyInput contains the parameters for revoking a LiteLLM virtual key.
type RevokeLLMKeyInput struct {
	Key string
}

// RevokeLLMKey revokes a LiteLLM virtual key.
func (a *Activities) RevokeLLMKey(ctx context.Context, input RevokeLLMKeyInput) error {
	if a.LiteLLMClient == nil || input.Key == "" {
		return nil
	}

	_, err := a.LiteLLMClient.DeleteKey(ctx, []string{input.Key})
	if err != nil {
		return fmt.Errorf("revoke LLM key: %w", err)
	}
	return nil
}

// sidecarClient creates a ConnectRPC client for the sidecar running in the given pod.
// Uses the pod IP directly since pod DNS names don't resolve without a headless Service.
func (a *Activities) sidecarClient(podIP string) agentv1connect.AgentSidecarServiceClient {
	addr := fmt.Sprintf("http://%s:%d", podIP, sidecarPort)
	httpClient := a.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return agentv1connect.NewAgentSidecarServiceClient(
		httpClient,
		addr,
	)
}
