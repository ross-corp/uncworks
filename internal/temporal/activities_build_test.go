package temporal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAgentPod_Structure(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:           "agentrun-test",
		Namespace:      "default",
		AgentRunName:   "test-run",
		Repos:          []Repository{{URL: "https://github.com/org/repo.git", Branch: "main"}},
		LLMKey:         "sk-test",
		LiteLLMBaseURL: "http://litellm:4000",
		ModelID:        "default",
		Image:          "aot-agent:test",
	}
	pod := BuildAgentPod(input)

	// Verify init containers
	require.Len(t, pod.Spec.InitContainers, 1, "should have hydration init container")
	assert.Equal(t, "hydration", pod.Spec.InitContainers[0].Name)

	// Verify containers
	require.Len(t, pod.Spec.Containers, 2, "should have agent + sidecar containers")
	assert.Equal(t, "agent", pod.Spec.Containers[0].Name)
	assert.Equal(t, "rpc-gateway", pod.Spec.Containers[1].Name)

	// Verify agent image
	assert.Equal(t, "aot-agent:test", pod.Spec.Containers[0].Image)

	// Verify OPENAI_API_KEY env var on agent container
	var foundLLMKey bool
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "OPENAI_API_KEY" {
			foundLLMKey = true
			assert.Equal(t, "sk-test", env.Value)
		}
	}
	require.True(t, foundLLMKey, "agent container should have OPENAI_API_KEY")

	// Verify OPENAI_BASE_URL env var on agent container
	var foundBaseURL bool
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "OPENAI_BASE_URL" {
			foundBaseURL = true
			assert.Equal(t, "http://litellm:4000/v1", env.Value)
		}
	}
	require.True(t, foundBaseURL, "agent container should have OPENAI_BASE_URL")

	// Verify labels
	assert.Equal(t, "test-run", pod.Labels["aot.uncworks.io/agentrun"])
	assert.Equal(t, "aot-agent", pod.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "aot-controller", pod.Labels["app.kubernetes.io/managed-by"])

	// Verify metadata
	assert.Equal(t, "agentrun-test", pod.Name)
	assert.Equal(t, "default", pod.Namespace)

	// Verify volume
	require.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, "workspace", pod.Spec.Volumes[0].Name)
	assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir, "workspace volume should be emptyDir")
}

func TestBuildAgentPod_DefaultImage(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		Image:        "", // empty — should use default
	}
	pod := BuildAgentPod(input)

	// Should use the default agent image (from env or hardcoded fallback)
	assert.NotEmpty(t, pod.Spec.Containers[0].Image, "agent image should not be empty")
}

func TestBuildAgentPod_ReposEnvVar(t *testing.T) {
	repos := []Repository{
		{URL: "https://github.com/org/repo1.git", Branch: "main"},
		{URL: "https://github.com/org/repo2.git", Branch: "develop"},
	}
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		Repos:        repos,
		Image:        "aot-agent:test",
	}
	pod := BuildAgentPod(input)

	// Find AOT_REPOS env var on agent container
	var reposEnv string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "AOT_REPOS" {
			reposEnv = env.Value
		}
	}
	require.NotEmpty(t, reposEnv, "should have AOT_REPOS env var")

	// Parse and verify
	var parsed []Repository
	require.NoError(t, json.Unmarshal([]byte(reposEnv), &parsed))
	require.Len(t, parsed, 2)
	assert.Equal(t, "https://github.com/org/repo1.git", parsed[0].URL)
	assert.Equal(t, "develop", parsed[1].Branch)
}

func TestBuildAgentPod_PlaceholderKeyWhenNoLLMKey(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:           "agentrun-test",
		Namespace:      "default",
		AgentRunName:   "test-run",
		LLMKey:         "",                    // no key
		LiteLLMBaseURL: "http://litellm:4000", // but base URL is set
		Image:          "aot-agent:test",
	}
	pod := BuildAgentPod(input)

	// Should set a placeholder key
	var keyValue string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "OPENAI_API_KEY" {
			keyValue = env.Value
		}
	}
	assert.Equal(t, "not-required", keyValue, "should use placeholder key when no LLMKey but LiteLLMBaseURL is set")
}

func TestBuildAgentPod_SidecarHasModelID(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		ModelID:      "gpt-4o",
		Image:        "aot-agent:test",
	}
	pod := BuildAgentPod(input)

	// Verify PI_MODEL env var on sidecar container
	sidecar := pod.Spec.Containers[1]
	var modelValue string
	for _, env := range sidecar.Env {
		if env.Name == "PI_MODEL" {
			modelValue = env.Value
		}
	}
	assert.Equal(t, "gpt-4o", modelValue, "sidecar should have PI_MODEL env var")
}

func TestBuildAgentPod_CustomEnvVars(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		Image:        "aot-agent:test",
		EnvVars: map[string]string{
			"CUSTOM_VAR": "custom-value",
		},
	}
	pod := BuildAgentPod(input)

	var found bool
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "CUSTOM_VAR" && env.Value == "custom-value" {
			found = true
		}
	}
	assert.True(t, found, "agent container should have custom env vars")
}

func TestBuildAgentPod_DevboxConfig(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		Image:        "aot-agent:test",
		DevboxConfig: `{"packages":["go@1.22"]}`,
	}
	pod := BuildAgentPod(input)

	var found bool
	for _, env := range pod.Spec.InitContainers[0].Env {
		if env.Name == "AOT_DEVBOX_CONFIG" {
			found = true
			assert.Equal(t, `{"packages":["go@1.22"]}`, env.Value)
		}
	}
	assert.True(t, found, "init container should have AOT_DEVBOX_CONFIG when set")
}

func TestBuildAgentPod_SpecContent(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:         "agentrun-test",
		Namespace:    "default",
		AgentRunName: "test-run",
		Image:        "aot-agent:test",
		SpecContent:  "## Requirements\nThe system SHALL do X.",
	}
	pod := BuildAgentPod(input)

	var found bool
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "AOT_SPEC_CONTENT" {
			found = true
			assert.Contains(t, env.Value, "SHALL")
		}
	}
	assert.True(t, found, "agent container should have AOT_SPEC_CONTENT when set")
}

func TestImagePullPolicy(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"aot-agent:test", "Never"},                     // local image, no registry prefix
		{"ghcr.io/uncworks/aot-agent:latest", "Always"}, // registry image
		{"docker.io/library/nginx:latest", "Always"},    // registry image
		{"my-local-image:dev", "Never"},                 // local image
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got := string(imagePullPolicy(tt.image))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildAgentPod_GitHubTokenSecretOnInitOnly(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:                  "agentrun-test",
		Namespace:             "default",
		AgentRunName:          "test-run",
		Image:                 "aot-agent:test",
		GitHubTokenSecretName: "github-token",
	}
	pod := BuildAgentPod(input)

	// Init container SHOULD have GITHUB_TOKEN from Secret
	initContainer := pod.Spec.InitContainers[0]
	var initHasToken bool
	for _, env := range initContainer.Env {
		if env.Name == "GITHUB_TOKEN" {
			initHasToken = true
			require.NotNil(t, env.ValueFrom, "GITHUB_TOKEN should come from Secret ref")
			require.NotNil(t, env.ValueFrom.SecretKeyRef)
			assert.Equal(t, "github-token", env.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, "token", env.ValueFrom.SecretKeyRef.Key)
		}
	}
	assert.True(t, initHasToken, "init container should have GITHUB_TOKEN env var")

	// Agent and sidecar containers should NOT have GITHUB_TOKEN
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			assert.NotEqual(t, "GITHUB_TOKEN", env.Name,
				"container %s should NOT have GITHUB_TOKEN", container.Name)
		}
	}
}

func TestBuildAgentPod_NoGitHubTokenSecretWhenEmpty(t *testing.T) {
	input := CreateAgentDeploymentInput{
		Name:                  "agentrun-test",
		Namespace:             "default",
		AgentRunName:          "test-run",
		Image:                 "aot-agent:test",
		GitHubTokenSecretName: "", // not configured
	}
	pod := BuildAgentPod(input)

	// Init container should NOT have GITHUB_TOKEN when secret name is empty
	initContainer := pod.Spec.InitContainers[0]
	for _, env := range initContainer.Env {
		assert.NotEqual(t, "GITHUB_TOKEN", env.Name,
			"init container should NOT have GITHUB_TOKEN when secret name is empty")
	}
}

func TestModelsForTier(t *testing.T) {
	tests := []struct {
		tier     string
		expected []string
	}{
		{"premium", []string{"default", "default-cloud", "premium"}},
		{"default-cloud", []string{"default", "default-cloud"}},
		{"default", []string{"default", "default-cloud"}},
		{"", []string{"default", "default-cloud"}},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := modelsForTier(tt.tier)
			assert.Equal(t, tt.expected, got)
		})
	}
}
