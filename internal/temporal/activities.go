// Package temporal implements Temporal workflows and activities for agent lifecycle orchestration.
package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
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

// imagePullPolicy returns Never for local images (no registry prefix or :local tag), Always otherwise.
func imagePullPolicy(image string) corev1.PullPolicy {
	if !strings.Contains(image, "/") {
		return corev1.PullNever
	}
	if strings.HasSuffix(image, ":local") {
		return corev1.PullNever
	}
	return corev1.PullAlways
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
	WorkspacePath string
}

// WaitForHydration polls the pod's init container status until hydration completes.
// Returns the pod IP so subsequent activities can reach the sidecar directly.
// When AgentRunName is set, discovers the pod via label selector (Deployment-managed).
func (a *Activities) WaitForHydration(ctx context.Context, input WaitForHydrationInput) (*WaitForHydrationOutput, error) {
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

	// Override: use PVC instead of emptyDir for workspace volume
	podTemplate.Spec.Volumes = []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
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
		return fmt.Errorf("scale down deployment: %w", err)
	}
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
