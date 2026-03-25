package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func makePushPayload(repo string, commits []commitInfo) []byte {
	p := pushPayload{
		Ref:   "refs/heads/main",
		After: "abc123",
		Repository: repoInfo{
			FullName: repo,
		},
		Commits: commits,
	}
	data, _ := json.Marshal(p)
	return data
}

// ---------- Signature Validation ----------

func TestValidateSignature_Valid(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "test-secret"
	sig := signPayload(body, secret)

	assert.True(t, validateSignature(body, sig, secret))
}

func TestValidateSignature_Invalid(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "test-secret"
	sig := signPayload(body, "wrong-secret")

	assert.False(t, validateSignature(body, sig, secret))
}

func TestValidateSignature_Missing(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	assert.False(t, validateSignature(body, "", "test-secret"))
}

func TestValidateSignature_MalformedHex(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	assert.False(t, validateSignature(body, "sha256=zzzz", "test-secret"))
}

func TestValidateSignature_NoPrefix(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	assert.False(t, validateSignature(body, "deadbeef", "test-secret"))
}

// ---------- Webhook Handler Signature Enforcement ----------

func TestWebhook_ValidSignature_Returns200(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "my-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	sig := signPayload(body, "my-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhook_InvalidSignature_Returns401(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "my-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	sig := signPayload(body, "wrong-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhook_MissingSignature_Returns401(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "my-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhook_EmptySecret_Returns401(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "", // not configured — fail-closed
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	// No signature header at all
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------- Payload Parsing ----------

func TestCollectSpecFiles_WithCSMDFiles(t *testing.T) {
	wh := &WebhookHandler{}
	commits := []commitInfo{
		{
			Added:    []string{"specs/feature.cs.md", "README.md"},
			Modified: []string{"specs/existing.cs.md"},
		},
		{
			Added:    []string{"other/file.go"},
			Modified: []string{"specs/feature.cs.md"}, // duplicate
		},
	}
	result := wh.collectSpecFiles(commits)
	assert.ElementsMatch(t, []string{"specs/feature.cs.md", "specs/existing.cs.md"}, result)
}

func TestCollectSpecFiles_NoCSMDFiles(t *testing.T) {
	wh := &WebhookHandler{}
	commits := []commitInfo{
		{
			Added:    []string{"main.go", "README.md"},
			Modified: []string{"pkg/server.go"},
		},
	}
	result := wh.collectSpecFiles(commits)
	assert.Empty(t, result)
}

func TestCollectSpecFiles_EmptyCommits(t *testing.T) {
	wh := &WebhookHandler{}
	result := wh.collectSpecFiles(nil)
	assert.Empty(t, result)
}

// ---------- Repo Allowlist ----------

func TestIsRepoAllowed_EmptyAllowlist(t *testing.T) {
	wh := &WebhookHandler{allowedRepos: nil}
	assert.True(t, wh.isRepoAllowed("any/repo"))
}

func TestIsRepoAllowed_RepoInList(t *testing.T) {
	wh := &WebhookHandler{allowedRepos: []string{"org/repo", "org/other"}}
	assert.True(t, wh.isRepoAllowed("org/repo"))
}

func TestIsRepoAllowed_RepoNotInList(t *testing.T) {
	wh := &WebhookHandler{allowedRepos: []string{"org/repo"}}
	assert.False(t, wh.isRepoAllowed("evil/repo"))
}

func TestIsRepoAllowed_CaseInsensitive(t *testing.T) {
	wh := &WebhookHandler{allowedRepos: []string{"Org/Repo"}}
	assert.True(t, wh.isRepoAllowed("org/repo"))
}

func TestWebhook_RepoNotAllowed_Returns200(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:       "test-secret",
		allowedRepos: []string{"org/allowed"},
		k8sClient:    k8s,
		namespace:    "default",
	}

	body := makePushPayload("org/notallowed", []commitInfo{
		{Added: []string{"spec.cs.md"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "repo not in allowlist")
}

// ---------- End-to-End: creates AgentRun for .cs.md files ----------

func TestWebhook_CreatesAgentRunForSpecFiles(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Stand up a fake GitHub API that returns file content.
	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "# My Spec\nDo the thing.")
	}))
	defer githubAPI.Close()

	wh := &WebhookHandler{
		secret:     "test-secret",
		k8sClient:  k8s,
		namespace:  "default",
		httpClient: githubAPI.Client(),
	}
	// Override fetchFileContent to use the test server.
	// We'll do this by setting the httpClient and reimplementing the URL.
	// Actually, it's simpler to test via createAgentRun directly and verify the CRD.

	// Instead, let's test the full handler with a real GitHub API mock.
	// We need to intercept the GitHub API URL. Let's use a custom RoundTripper.
	wh.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(rec, "# My Spec\nDo the thing.")
			return rec.Result(), nil
		}),
	}

	body := makePushPayload("org/repo", []commitInfo{
		{Added: []string{"specs/new-feature.cs.md", "README.md"}},
		{Modified: []string{"specs/updated.cs.md"}},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify CRDs were created.
	var list aotv1alpha1.AgentRunList
	err := k8s.List(context.Background(), &list)
	require.NoError(t, err)
	assert.Len(t, list.Items, 2)

	// Verify spec content and source.
	for _, item := range list.Items {
		assert.Equal(t, "# My Spec\nDo the thing.", item.Spec.SpecContent)
		assert.True(t, item.Spec.SpecSource == "webhook:github:org/repo/specs/new-feature.cs.md" ||
			item.Spec.SpecSource == "webhook:github:org/repo/specs/updated.cs.md",
			"unexpected spec source: %s", item.Spec.SpecSource)
		assert.Equal(t, aotv1alpha1.AgentRunPhasePending, item.Status.Phase)
		assert.Equal(t, "main", item.Spec.Repos[0].Branch)
	}
}

func TestWebhook_NoSpecFiles_Returns200WithZeroCreated(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "test-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", []commitInfo{
		{Added: []string{"main.go"}, Modified: []string{"go.sum"}},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"created":0`)
}

func TestWebhook_IgnoresNonPushEvent(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "test-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := []byte(`{"action":"opened"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ignored event type")
}

// roundTripFunc is an adapter to use a function as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
