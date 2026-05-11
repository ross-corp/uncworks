// Package temporal implements Temporal workflows and activities for agent lifecycle orchestration.
package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	temporalsdk "go.temporal.io/sdk/temporal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aotgithub "github.com/uncworks/aot/internal/github"
	"github.com/uncworks/aot/internal/litellm"
)

const sidecarPort = 50052

// Image names — configurable via environment variables for local development.
var (
	agentImage   = envOrDefault("AOT_AGENT_IMAGE", "ghcr.io/uncworks/aot-agent:latest")
	sidecarImage = envOrDefault("AOT_SIDECAR_IMAGE", "ghcr.io/uncworks/aot-sidecar:latest")
	initImage    = envOrDefault("AOT_INIT_IMAGE", "ghcr.io/uncworks/aot-init:latest")
	// cudgelEndpoint is the base URL of the cudgel HTTP shim, injected into agent and init pods.
	// Sourced from CUDGEL_ENDPOINT env var on the worker/controller process.
	cudgelEndpoint = os.Getenv("CUDGEL_ENDPOINT")
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// imagePullPolicy returns Never for local images (no registry prefix or :local tag),
// IfNotPresent otherwise. PullAlways is intentionally avoided: it causes ErrImagePull
// in air-gapped or local-only clusters where a versioned image (e.g. docker.io/library/aot-init:v1.0)
// is already present but not reachable on the public registry.
func imagePullPolicy(image string) corev1.PullPolicy {
	if !strings.Contains(image, "/") {
		return corev1.PullNever
	}
	if strings.HasSuffix(image, ":local") {
		return corev1.PullNever
	}
	return corev1.PullIfNotPresent
}

// Activities holds the dependencies needed by Temporal activity implementations.
type Activities struct {
	K8sClient      client.Client
	LiteLLMClient  *litellm.Client
	HTTPClient     *http.Client
	GitHubProvider aotgithub.TokenProvider
	// GitHubTokenSecretName is the k8s Secret name containing the GitHub token.
	// When set, the init container gets GITHUB_TOKEN from this Secret.
	GitHubTokenSecretName string
}

// WaitForHydrationInput contains the parameters for waiting on hydration.
type WaitForHydrationInput struct {
	PodName      string
	Namespace    string
	AgentRunName string // Used for label-based pod discovery (Deployment-managed pods)
}

// WaitForHydrationOutput contains the result of waiting for hydration.
type WaitForHydrationOutput struct {
	PodIP         string
	PodName       string
	WorkspacePath string
}

// WaitForHydration polls the pod's init container status until hydration completes.
// Returns the pod IP so subsequent activities can reach the sidecar directly.
// When AgentRunName is set, discovers the pod via label selector (Deployment-managed).
func (a *Activities) WaitForHydration(ctx context.Context, input WaitForHydrationInput) (*WaitForHydrationOutput, error) {
	slog.Debug("WaitForHydration started", "agentRun", input.AgentRunName, "namespace", input.Namespace, "podName", input.PodName)
	
	iteration := 0
	for {
		pod, err := a.findPod(ctx, input.Namespace, input.AgentRunName, input.PodName)
		if err != nil {
			// Pod may not exist yet if Deployment is still creating it
			activity.RecordHeartbeat(ctx, fmt.Sprintf("waiting for pod: %v", err))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
			continue
		}

		// Check for eviction specifically
		if pod.Status.Reason == "Evicted" {
			return nil, temporalsdk.NewApplicationError(
				fmt.Sprintf("pod was evicted before hydration completed: %s", pod.Status.Message),
				"eviction",
			)
		}

		for _, initStatus := range pod.Status.InitContainerStatuses {
			if initStatus.Name == "hydration" {
				if initStatus.State.Terminated != nil {
					if initStatus.State.Terminated.ExitCode == 0 {
						slog.Info("hydration complete", "agentRun", input.AgentRunName, "podIP", pod.Status.PodIP, "pod", pod.Name)
						return &WaitForHydrationOutput{PodIP: pod.Status.PodIP, PodName: pod.Name}, nil
					}
					return nil, fmt.Errorf("hydration failed with exit code %d: %s",
						initStatus.State.Terminated.ExitCode,
						initStatus.State.Terminated.Message)
				}
			}
		}

		// Pod is running (past init) — hydration succeeded
		if pod.Status.Phase == corev1.PodRunning {
			slog.Info("hydration complete", "agentRun", input.AgentRunName, "podIP", pod.Status.PodIP, "pod", pod.Name)
			return &WaitForHydrationOutput{PodIP: pod.Status.PodIP, PodName: pod.Name}, nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			// Check if pod failed due to eviction
			if pod.Status.Reason == "Evicted" {
				return nil, temporalsdk.NewApplicationError(
					fmt.Sprintf("pod was evicted before hydration completed: %s", pod.Status.Message),
					"eviction",
				)
			}
			return nil, fmt.Errorf("pod failed before hydration completed: %s", pod.Status.Message)
		}

		// Log debug message every 10 iterations (~20 seconds)
		iteration++
		if iteration%10 == 0 {
			slog.Debug("WaitForHydration polling", "iteration", iteration, "agentRun", input.AgentRunName, "podPhase", pod.Status.Phase)
		}

		activity.RecordHeartbeat(ctx, fmt.Sprintf("waiting for hydration: pod %s", pod.Name))

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
	PodName      string
	Namespace    string
	PodIP        string
	Prompt       string
	RepoPath     string
	Model        string // LiteLLM model name override (passed as PI_MODEL env var)
	Stage        string // Pipeline stage: "plan", "execute", "verify", or "" for single
	ParentSpanID string // Links to parent stage span in the trace hierarchy
	TraceID      string // Shared trace identifier across all spans in a pipeline run
}

// StartAgent calls the sidecar StartAgent RPC, retrying until the sidecar is ready.
func (a *Activities) StartAgent(ctx context.Context, input StartAgentInput) error {
	slog.Debug("StartAgent started", "podName", input.PodName, "namespace", input.Namespace, "podIP", input.PodIP, "stage", input.Stage, "promptLength", len(input.Prompt))
	sc := a.sidecarClient(input.PodIP)

	// Retry until sidecar is ready (it may still be starting)
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("waiting for sidecar readiness: attempt %d", attempt+1))

		envVars := map[string]string{}
		if input.Model != "" {
			envVars["PI_MODEL"] = input.Model
		}

		resp, err := sc.StartAgent(ctx, connect.NewRequest(&agentv1.StartAgentRequest{
			Prompt:       input.Prompt,
			RepoPath:     input.RepoPath,
			Stage:        input.Stage,
			EnvVars:      envVars,
			ParentSpanId: input.ParentSpanID,
			TraceId:      input.TraceID,
		}))
		if err == nil {
			if !resp.Msg.Started {
				return fmt.Errorf("agent did not start: %s", resp.Msg.Error)
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
	slog.Debug("GetAgentStatus started", "podName", input.PodName, "namespace", input.Namespace, "podIP", input.PodIP)
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

// CheckPodStatusInput contains the parameters for checking pod status.
type CheckPodStatusInput struct {
	PodName   string
	Namespace string
}

// CheckPodStatusOutput contains the pod's current status.
type CheckPodStatusOutput struct {
	Phase  corev1.PodPhase
	Reason string
	Message string
}

// CheckPodStatus checks the current phase and reason of a pod.
func (a *Activities) CheckPodStatus(ctx context.Context, input CheckPodStatusInput) (*CheckPodStatusOutput, error) {
	pod, err := a.findPod(ctx, input.Namespace, "", input.PodName)
	if err != nil {
		return nil, fmt.Errorf("find pod %s: %w", input.PodName, err)
	}
	
	return &CheckPodStatusOutput{
		Phase:   pod.Status.Phase,
		Reason:  pod.Status.Reason,
		Message: pod.Status.Message,
	}, nil
}

// ForwardHumanInputInput contains the parameters for forwarding human input.
type ForwardHumanInputInput struct {
	AgentRunID string
	PodName    string
	Namespace  string
	PodIP      string
	Input      string
}

// ForwardHumanInput calls the sidecar SendInput RPC.
func (a *Activities) ForwardHumanInput(ctx context.Context, input ForwardHumanInputInput) error {
	sidecarClient := a.sidecarClient(input.PodIP)

	_, err := sidecarClient.SendInput(ctx, connect.NewRequest(&agentv1.SendInputRequest{
		AgentRunId: input.AgentRunID,
		Data:       []byte(input.Input),
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

// hostPathDirectoryOrCreate returns the HostPathType value for DirectoryOrCreate.
func hostPathDirectoryOrCreate() *corev1.HostPathType {
	t := corev1.HostPathDirectoryOrCreate
	return &t
}

// BuildAgentPod creates a pod spec for an agent run.
// Used by CreateAgentDeployment as the pod template source.
func BuildAgentPod(input CreateAgentDeploymentInput) *corev1.Pod {
	image := input.Image
	if image == "" {
		image = agentImage
	}

	reposJSON, _ := json.Marshal(input.Repos)
	envVars := []corev1.EnvVar{
		{Name: "AOT_AGENT_RUN_ID", Value: input.AgentRunName},
		{Name: "AOT_REPOS", Value: string(reposJSON)},
		{Name: "AOT_PROMPT", Value: input.Prompt},
	}
	if input.DevboxConfig != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "AOT_DEVBOX_CONFIG", Value: input.DevboxConfig})
	}
	if input.SpecContent != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "AOT_SPEC_CONTENT", Value: input.SpecContent})
	}
	// Collect and sort env var keys for deterministic iteration
	keys := make([]string, 0, len(input.EnvVars))
	for k := range input.EnvVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: input.EnvVars[k]})
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
	} else if input.LiteLLMBaseURL != "" {
		// Set a placeholder key for endpoints that don't require auth (e.g., Ollama)
		llmEnvVars = append(llmEnvVars, corev1.EnvVar{
			Name:  "OPENAI_API_KEY",
			Value: "not-required",
		})
	}
	agentEnvVars := append(envVars, llmEnvVars...) //nolint:gocritic // intentional new slice

	// Init container env: base env vars + optional GITHUB_TOKEN from Secret + optional CUDGEL_ENDPOINT
	initEnvVars := make([]corev1.EnvVar, len(envVars))
	copy(initEnvVars, envVars)
	if cudgelEndpoint != "" {
		initEnvVars = append(initEnvVars, corev1.EnvVar{Name: "CUDGEL_ENDPOINT", Value: cudgelEndpoint})
		// Also pass to agent containers (sidecar reads it for SemanticSearch RPC)
		agentEnvVars = append(agentEnvVars, corev1.EnvVar{Name: "CUDGEL_ENDPOINT", Value: cudgelEndpoint})
	}
	if input.GitHubTokenSecretName != "" {
		initEnvVars = append(initEnvVars, corev1.EnvVar{
			Name: "GITHUB_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: input.GitHubTokenSecretName,
					},
					Key: "token",
				},
			},
		})
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
					Name:            "hydration",
					Image:           initImage,
					ImagePullPolicy: imagePullPolicy(initImage),
					Env:             initEnvVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
						// Shared Nix store: devbox packages survive pod restarts and are
						// shared across all agent runs on the same node, making subsequent
						// devbox installs fast (cache hit) instead of re-downloading ~1 GB.
						{Name: "nix-store", MountPath: "/nix"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1000m"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            "agent",
					Image:           image,
					ImagePullPolicy: imagePullPolicy(image),
					Command:         []string{"sleep", "infinity"}, // Keep agent container alive while sidecar runs pi
					Env:             agentEnvVars,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
				{
					Name:            "rpc-gateway",
					Image:           sidecarImage,
					ImagePullPolicy: imagePullPolicy(sidecarImage),
					Ports: []corev1.ContainerPort{
						{Name: "grpc", ContainerPort: sidecarPort, Protocol: corev1.ProtocolTCP},
					},
					Env: append(append([]corev1.EnvVar{
						{Name: "AOT_AGENT_RUN_ID", Value: input.AgentRunName},
						{Name: "PI_MODEL", Value: input.ModelID},
					}, llmEnvVars...), corev1.EnvVar{
						Name: "PI_ACCEPT_TOS", Value: "1",
					}),
					VolumeMounts: []corev1.VolumeMount{
						{Name: "workspace", MountPath: "/workspace"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1000m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
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
				{
					Name: "nix-store",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/aot/nix",
							Type: hostPathDirectoryOrCreate(),
						},
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
// Used as a fallback when the proxy is unreachable.
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
// It dynamically queries available models from the proxy and grants
// access to all of them, falling back to hardcoded tier-based models
// if the proxy is unreachable.
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

	// Dynamically discover available models from LiteLLM proxy.
	// Fall back to hardcoded tier-based models if the proxy is unreachable.
	models, err := a.LiteLLMClient.ModelIDs(ctx)
	if err != nil {
		activity.GetLogger(ctx).Warn("Failed to list models from LiteLLM, using tier-based fallback", "error", err)
		models = modelsForTier(tier)
	}

	resp, err := a.LiteLLMClient.GenerateKey(ctx, litellm.GenerateKeyRequest{
		KeyAlias:  fmt.Sprintf("aot-%s-%s", input.Namespace, input.AgentRunName),
		MaxBudget: &budget,
		Models:    models,
		Metadata: map[string]string{
			"agent_run": input.AgentRunName,
			"namespace": input.Namespace,
			"tier":      tier,
		},
	})
	if err != nil {
		// Fall back to master key when virtual key generation fails
		// (e.g., no database configured in local dev)
		activity.GetLogger(ctx).Warn("Virtual key generation failed, falling back to master key", "error", err)
		return &ProvisionLLMKeyOutput{Key: a.LiteLLMClient.MasterKey()}, nil
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
	// Skip revocation if the key is the master key — it was used as a fallback
	// when no DB is configured and cannot be deleted via the key management API.
	if input.Key == a.LiteLLMClient.MasterKey() {
		return nil
	}

	_, err := a.LiteLLMClient.DeleteKey(ctx, []string{input.Key})
	if err != nil {
		return fmt.Errorf("revoke LLM key: %w", err)
	}
	return nil
}

// findPod discovers the pod for an agent run via label selector (preferred) or pod name.
func (a *Activities) findPod(ctx context.Context, namespace, agentRunName, podName string) (*corev1.Pod, error) {
	if agentRunName != "" {
		var podList corev1.PodList
		labelSelector := labels.SelectorFromSet(map[string]string{
			"aot.uncworks.io/agentrun": agentRunName,
		})
		if err := a.K8sClient.List(ctx, &podList,
			client.InNamespace(namespace),
			client.MatchingLabelsSelector{Selector: labelSelector},
		); err != nil {
			return nil, fmt.Errorf("list pods by label: %w", err)
		}
		// Find a running or pending pod (not terminated)
		for i := range podList.Items {
			p := &podList.Items[i]
			if p.DeletionTimestamp == nil && p.Status.Phase != corev1.PodSucceeded && p.Status.Phase != corev1.PodFailed {
				return p, nil
			}
		}
		if len(podList.Items) > 0 {
			return &podList.Items[0], nil
		}
		return nil, fmt.Errorf("no pod found with label aot.uncworks.io/agentrun=%s", agentRunName)
	}
	// Fallback: direct pod name lookup (deprecated bare-pod path)
	var pod corev1.Pod
	if err := a.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      podName,
	}, &pod); err != nil {
		return nil, fmt.Errorf("get pod %s: %w", podName, err)
	}
	return &pod, nil
}

// --- Deployment-based activities (persistent-workspace-architecture) ---

// CreateAgentDeploymentInput contains parameters for creating an agent Deployment + PVC.
type CreateAgentDeploymentInput struct {
	Name                  string
	Namespace             string
	AgentRunName          string
	Repos                 []Repository
	Prompt                string
	DevboxConfig          string
	Image                 string
	EnvVars               map[string]string
	LLMKey                string
	LiteLLMBaseURL        string
	ModelID               string
	SpecContent           string
	GitHubTokenSecretName string // k8s Secret name for GITHUB_TOKEN (init container only)
}

// CreateAgentDeploymentOutput contains the result of creating an agent Deployment + PVC.
type CreateAgentDeploymentOutput struct {
	DeploymentName string
	PVCName        string
}

// CreateAgentDeployment creates a PVC and Deployment for an agent run.
// The Deployment is structurally identical to BuildAgentPod but uses a PVC
// instead of emptyDir for the workspace volume.
func (a *Activities) CreateAgentDeployment(ctx context.Context, input CreateAgentDeploymentInput) (*CreateAgentDeploymentOutput, error) {
	pvcName := fmt.Sprintf("aot-ws-%s", input.AgentRunName)
	deployName := input.Name

	slog.Info("agent deployment created", "agentRunName", input.AgentRunName, "deploymentName", deployName, "namespace", input.Namespace)

	// Create PVC
	storageClass := envOrDefault("AOT_STORAGE_CLASS", "local-path")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: input.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "aot-agent",
				"app.kubernetes.io/managed-by": "aot-controller",
				"aot.uncworks.io/agentrun":     input.AgentRunName,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(envOrDefault("AOT_PVC_SIZE", "2Gi")),
				},
			},
		},
	}

	if err := a.K8sClient.Create(ctx, pvc); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("create PVC: %w", err)
		}
	}

	// Build the pod template
	podTemplate := BuildAgentPod(input)

	// Override: use PVC instead of emptyDir for workspace; preserve nix-store hostPath.
	podTemplate.Spec.Volumes = []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
		{
			Name: "nix-store",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/aot/nix",
					Type: hostPathDirectoryOrCreate(),
				},
			},
		},
	}

	// Override: Deployment-managed pods should restart on failure
	podTemplate.Spec.RestartPolicy = corev1.RestartPolicyAlways

	selectorLabels := map[string]string{
		"aot.uncworks.io/agentrun": input.AgentRunName,
	}

	var replicas int32 = 1
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: input.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "aot-agent",
				"app.kubernetes.io/managed-by": "aot-controller",
				"aot.uncworks.io/agentrun":     input.AgentRunName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podTemplate.Labels,
				},
				Spec: podTemplate.Spec,
			},
		},
	}

	if err := a.K8sClient.Create(ctx, deployment); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("create deployment: %w", err)
		}
	}

	return &CreateAgentDeploymentOutput{
		DeploymentName: deployName,
		PVCName:        pvcName,
	}, nil
}

// ScaleDownDeploymentInput contains the parameters for scaling a Deployment to 0.
type ScaleDownDeploymentInput struct {
	DeploymentName string
	Namespace      string
}

// ScaleDownDeployment patches a Deployment's replicas to 0.
// The Deployment and PVC are NOT deleted — workspace data persists.
func (a *Activities) ScaleDownDeployment(ctx context.Context, input ScaleDownDeploymentInput) error {
	if !strings.HasPrefix(input.DeploymentName, "agentrun-") {
		slog.Warn("ScaleDownDeployment: refusing to scale down non-agent deployment", "deployment", input.DeploymentName)
		return fmt.Errorf("safety check: deployment %q does not have agentrun- prefix", input.DeploymentName)
	}
	var deployment appsv1.Deployment
	if err := a.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.DeploymentName,
	}, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return nil // Already gone
		}
		return fmt.Errorf("get deployment: %w", err)
	}

	var replicas int32 = 0
	deployment.Spec.Replicas = &replicas
	if err := a.K8sClient.Update(ctx, &deployment); err != nil {
		slog.Warn("scale down deployment failed", "deployment", input.DeploymentName, "namespace", input.Namespace, "err", err)
		return fmt.Errorf("scale down deployment: %w", err)
	}
	slog.Info("deployment scaled down", "deployment", input.DeploymentName, "namespace", input.Namespace)
	return nil
}

// CollectAgentLogsInput contains parameters for collecting and persisting agent pod logs.
type CollectAgentLogsInput struct {
	AgentRunName string
	Namespace    string
	PodIP        string // sidecar address; required to read log from PVC via ExecCommand
}

// CollectAgentLogs reads the agent log file from the workspace PVC via the sidecar's
// ExecCommand RPC and persists the last 32 KB to AgentRun status.logOutput. Must be
// called before ScaleDownDeployment while the sidecar is still reachable.
func (a *Activities) CollectAgentLogs(ctx context.Context, input CollectAgentLogsInput) error {
	slog.Info("CollectAgentLogs started", "agentRun", input.AgentRunName, "namespace", input.Namespace, "podIP", input.PodIP)
	if input.PodIP == "" {
		slog.Warn("collect logs: no pod IP, skipping", "agentRun", input.AgentRunName)
		return nil
	}

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		sidecarURL,
	)

	// Collect both the full JSONL (for the UI structured-log endpoint) and the last
	// assistant text (for chain contextFrom). Output is a JSON object:
	//   {"logOutput":"...last assistant text...","agentJSONL":"...full jsonl..."}
	// Falls back to agent.log tail if node is unavailable.
	const extractScript = `
node -e "
const fs=require('fs');
const f='/workspace/.aot/logs/agent.jsonl';
const raw=fs.existsSync(f)?fs.readFileSync(f,'utf8'):'';
const lines=raw.trim().split('\n').filter(Boolean);
let lastText='';
for(const l of lines){
  try{
    const o=JSON.parse(l);
    if(o.type==='message_end'&&o.message&&o.message.role==='assistant'){
      const t=o.message.content.filter(c=>c.type==='text').map(c=>c.text).join('');
      if(t.trim())lastText=t;
    }
  }catch(e){}
}
const cappedJSONL=raw.length>524288?raw.slice(-524288):raw;
process.stdout.write(JSON.stringify({logOutput:lastText.slice(-32768),agentJSONL:cappedJSONL}));
" 2>/dev/null || (echo '{}' && true)`
	resp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        extractScript,
		WorkingDir:     "/workspace",
		TimeoutSeconds: 30,
	}))
	if err != nil {
		slog.Warn("collect logs: exec failed", "agentRun", input.AgentRunName, "err", err)
		return nil // non-fatal
	}

	stdout := strings.TrimRight(resp.Msg.Stdout, "\x00")

	var parsed struct {
		LogOutput  string `json:"logOutput"`
		AgentJSONL string `json:"agentJSONL"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		// Fallback: treat stdout as plain logOutput (old behavior)
		parsed.LogOutput = stdout
	}

	// Patch both fields in a single status update.
	patchBytes, _ := json.Marshal(map[string]interface{}{
		"status": map[string]string{
			"logOutput":  parsed.LogOutput,
			"agentJSONL": parsed.AgentJSONL,
		},
	})
	crd := &aotv1alpha1.AgentRun{}
	crd.Name = input.AgentRunName
	crd.Namespace = input.Namespace
	if err := a.K8sClient.Status().Patch(ctx, crd, client.RawPatch(types.MergePatchType, patchBytes)); err != nil {
		slog.Warn("collect logs: patch failed", "agentRun", input.AgentRunName, "err", err)
		return nil
	}

	slog.Info("CollectAgentLogs completed", "agentRun", input.AgentRunName,
		"logOutputBytes", len(parsed.LogOutput), "agentJSONLBytes", len(parsed.AgentJSONL))
	return nil
}

// ArchiveAndCleanupInput contains the parameters for archiving and cleaning up a run.
type ArchiveAndCleanupInput struct {
	DeploymentName string
	PVCName        string
	Namespace      string
}

// ArchiveAndCleanup deletes the Deployment and PVC for a completed run.
// This is called after the archive retention period expires.
func (a *Activities) ArchiveAndCleanup(ctx context.Context, input ArchiveAndCleanupInput) error {
	if !strings.HasPrefix(input.DeploymentName, "agentrun-") {
		slog.Warn("ArchiveAndCleanup: refusing to delete non-agent deployment", "deployment", input.DeploymentName)
		return fmt.Errorf("safety check: deployment %q does not have agentrun- prefix", input.DeploymentName)
	}
	if input.PVCName != "" && !strings.HasPrefix(input.PVCName, "aot-ws-") {
		slog.Warn("ArchiveAndCleanup: refusing to delete non-agent PVC", "pvc", input.PVCName)
		return fmt.Errorf("safety check: PVC %q does not have aot-ws- prefix", input.PVCName)
	}
	slog.Info("cleanup started", "agentRunName", input.DeploymentName, "deploymentName", input.DeploymentName)

	// Delete Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.DeploymentName,
			Namespace: input.Namespace,
		},
	}
	if err := a.K8sClient.Delete(ctx, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("delete deployment: %w", err)
		}
	}

	// Delete PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.PVCName,
			Namespace: input.Namespace,
		},
	}
	if err := a.K8sClient.Delete(ctx, pvc); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("delete PVC: %w", err)
		}
	}

	slog.Info("cleanup complete", "agentRunName", input.DeploymentName, "deploymentName", input.DeploymentName)
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
