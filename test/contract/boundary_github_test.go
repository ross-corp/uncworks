package contract

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aotgithub "github.com/uncworks/aot/internal/github"
	"github.com/uncworks/aot/internal/server"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestBoundary_NoDirectGitHubTokenEnvAccess ensures no production code in
// internal/temporal/ or internal/server/ calls os.Getenv("GITHUB_TOKEN")
// directly. All consumers must go through the TokenProvider interface.
func TestBoundary_NoDirectGitHubTokenEnvAccess(t *testing.T) {
	root := findRepoRoot(t)
	dirs := []string{
		filepath.Join(root, "internal", "temporal"),
		filepath.Join(root, "internal", "server"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err, "reading directory %s", dir)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}

			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			require.NoError(t, err, "reading %s", path)

			content := string(data)
			assert.False(t,
				strings.Contains(content, `os.Getenv("GITHUB_TOKEN")`),
				"%s contains os.Getenv(\"GITHUB_TOKEN\") — use the TokenProvider interface instead", name)
		}
	}
}

// TestBoundary_ActivitiesHasGitHubProvider verifies the Activities struct
// exposes a GitHubProvider field of type TokenProvider, so all activity
// methods can obtain tokens through the provider abstraction.
func TestBoundary_ActivitiesHasGitHubProvider(t *testing.T) {
	a := &aottemporal.Activities{
		GitHubProvider: aotgithub.NewPATProvider("test-token"),
	}

	// The field exists and is usable
	token, err := a.GitHubProvider.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test-token", token)
}

// TestBoundary_GitHubClientRequiresProvider verifies NewGitHubClient accepts
// a TokenProvider, enforcing the dependency-injection boundary.
func TestBoundary_GitHubClientRequiresProvider(t *testing.T) {
	provider := aotgithub.NewPATProvider("test")
	client := server.NewGitHubClient(provider)
	require.NotNil(t, client)
}

// TestBoundary_WebhookHandlerRequiresProvider verifies NewWebhookHandler
// accepts a TokenProvider parameter for fetching file content.
func TestBoundary_WebhookHandlerRequiresProvider(t *testing.T) {
	provider := aotgithub.NewPATProvider("test")
	handler := server.NewWebhookHandler(context.Background(), nil, "default", provider)
	require.NotNil(t, handler)
}

// TestBoundary_AgentPodSecurityBoundary is a stronger version of the
// existing unit test: it verifies that when GitHubTokenSecretName is set
// the init container has GITHUB_TOKEN from the Secret, while both the
// agent container AND the sidecar container do NOT.
func TestBoundary_AgentPodSecurityBoundary(t *testing.T) {
	input := aottemporal.CreateAgentDeploymentInput{
		Name:                  "boundary-test",
		Namespace:             "default",
		AgentRunName:          "boundary-run",
		Image:                 "aot-agent:test",
		GitHubTokenSecretName: "github-token",
	}
	pod := aottemporal.BuildAgentPod(input)

	// Init container MUST have GITHUB_TOKEN from Secret
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
	require.True(t, initHasToken, "init container MUST have GITHUB_TOKEN env var from Secret")

	// Agent container MUST NOT have GITHUB_TOKEN
	agentContainer := pod.Spec.Containers[0]
	require.Equal(t, "agent", agentContainer.Name)
	for _, env := range agentContainer.Env {
		assert.NotEqual(t, "GITHUB_TOKEN", env.Name,
			"agent container MUST NOT have GITHUB_TOKEN")
	}

	// Sidecar container MUST NOT have GITHUB_TOKEN
	sidecarContainer := pod.Spec.Containers[1]
	require.Equal(t, "rpc-gateway", sidecarContainer.Name)
	for _, env := range sidecarContainer.Env {
		assert.NotEqual(t, "GITHUB_TOKEN", env.Name,
			"sidecar container MUST NOT have GITHUB_TOKEN")
	}
}

// TestBoundary_HelmTemplateGitHubTokenConditional verifies that the worker
// and apiserver Helm templates only inject GITHUB_TOKEN inside a
// conditional block guarded by .Values.github.tokenSecretName.
func TestBoundary_HelmTemplateGitHubTokenConditional(t *testing.T) {
	root := findRepoRoot(t)
	templates := []struct {
		name string
		path string
	}{
		{"worker", filepath.Join(root, "deploy", "helm", "aot", "templates", "worker.yaml")},
		{"apiserver", filepath.Join(root, "deploy", "helm", "aot", "templates", "apiserver.yaml")},
	}

	for _, tmpl := range templates {
		t.Run(tmpl.name, func(t *testing.T) {
			data, err := os.ReadFile(tmpl.path)
			require.NoError(t, err, "reading template %s", tmpl.path)

			content := string(data)

			// Template must contain the conditional guard
			require.Contains(t, content, "{{- if .Values.github.tokenSecretName }}",
				"%s template must guard GITHUB_TOKEN with {{- if .Values.github.tokenSecretName }}", tmpl.name)

			// GITHUB_TOKEN must appear inside the template
			require.Contains(t, content, "GITHUB_TOKEN",
				"%s template must reference GITHUB_TOKEN", tmpl.name)

			// Verify GITHUB_TOKEN appears AFTER the if-guard and BEFORE the end block.
			// This ensures the token is only injected conditionally.
			ifIdx := strings.Index(content, "{{- if .Values.github.tokenSecretName }}")
			tokenIdx := strings.Index(content, "GITHUB_TOKEN")
			endIdx := strings.Index(content[ifIdx:], "{{- end }}")
			require.Greater(t, endIdx, 0, "must have {{- end }} after the if-guard")

			assert.Greater(t, tokenIdx, ifIdx,
				"%s: GITHUB_TOKEN must appear after the if-guard", tmpl.name)
			assert.Less(t, tokenIdx, ifIdx+endIdx,
				"%s: GITHUB_TOKEN must appear before the matching end block", tmpl.name)
		})
	}
}

// TestBoundary_PushChangesUsesProvider scans the PushChanges source to verify
// it uses GitHubProvider / Token(ctx) and does NOT call os.Getenv("GITHUB_TOKEN").
func TestBoundary_PushChangesUsesProvider(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Join(root, "internal", "temporal", "activities_git.go")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)

	assert.False(t,
		strings.Contains(content, `os.Getenv("GITHUB_TOKEN")`),
		"activities_git.go must NOT call os.Getenv(\"GITHUB_TOKEN\")")

	assert.True(t,
		strings.Contains(content, "GitHubProvider") || strings.Contains(content, "Token(ctx)"),
		"activities_git.go must reference GitHubProvider or Token(ctx)")
}

// TestBoundary_CreatePRUsesProvider scans the CreatePR source to verify
// it uses GitHubProvider / Token(ctx) and does NOT call os.Getenv("GITHUB_TOKEN").
func TestBoundary_CreatePRUsesProvider(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Join(root, "internal", "temporal", "activities_git.go")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)

	// Locate the CreatePR function body
	createPRIdx := strings.Index(content, "func (a *Activities) CreatePR(")
	require.Greater(t, createPRIdx, 0, "CreatePR function must exist in activities_git.go")

	createPRBody := content[createPRIdx:]

	assert.False(t,
		strings.Contains(createPRBody, `os.Getenv("GITHUB_TOKEN")`),
		"CreatePR must NOT call os.Getenv(\"GITHUB_TOKEN\")")

	assert.True(t,
		strings.Contains(createPRBody, "GitHubProvider") || strings.Contains(createPRBody, "Token(ctx)"),
		"CreatePR must reference GitHubProvider or Token(ctx)")
}

// findRepoRoot walks up from the current working directory to find the
// repository root (identified by go.mod).
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}
