//go:build regression

// test/regression/webhook_delivery_test.go — End-to-end tests for webhook
// delivery at the HTTP handler level. Verifies that GitHub push events are
// validated correctly and produce the right HTTP responses and k8s side effects.
//
// Note: WebhookHandler.httpClient is unexported so we cannot inject a fake
// transport from outside the server package.  Tests that require fetchFileContent
// to succeed (i.e., push events with .cs.md files) are covered in
// internal/server/webhook_test.go (same-package access).  This regression suite
// covers all other observable boundaries: HMAC auth, method routing, event
// filtering, and zero-spec-file pushes.
package regression

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/test/testutil"
)

// makeWebhookHandler creates a WebhookHandler via the public constructor,
// setting the GITHUB_WEBHOOK_SECRET env var to the given secret.
func makeWebhookHandler(t *testing.T, secret string) *server.WebhookHandler {
	t.Helper()
	t.Setenv("GITHUB_WEBHOOK_SECRET", secret)
	k8s := fake.NewClientBuilder().WithScheme(testutil.NewScheme()).Build()
	return server.NewWebhookHandler(context.Background(), k8s, testutil.DefaultNamespace, nil)
}

// makeWebhookHandlerWithClient returns both the handler and the underlying
// fake k8s client so callers can inspect CRDs after the request.
func makeWebhookHandlerWithClient(t *testing.T, secret string) (*server.WebhookHandler, *fake.ClientBuilder) {
	t.Helper()
	t.Setenv("GITHUB_WEBHOOK_SECRET", secret)
	builder := fake.NewClientBuilder().WithScheme(testutil.NewScheme())
	k8s := builder.Build()
	wh := server.NewWebhookHandler(context.Background(), k8s, testutil.DefaultNamespace, nil)
	return wh, builder
}

// TestWebhookDelivery_MissingSignature_Returns401 verifies that push events
// without an HMAC signature are rejected with 401 Unauthorized.
func TestWebhookDelivery_MissingSignature_Returns401(t *testing.T) {
	const secret = "webhook-regression-secret"
	wh := makeWebhookHandler(t, secret)

	body := testutil.BuildWebhookPushPayload("org/repo", []string{"README.md"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	// No X-Hub-Signature-256 header.
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"webhook without HMAC signature must be rejected with 401")
}

// TestWebhookDelivery_WrongSignature_Returns401 verifies that a request signed
// with the wrong secret is rejected.
func TestWebhookDelivery_WrongSignature_Returns401(t *testing.T) {
	const secret = "webhook-regression-secret"
	wh := makeWebhookHandler(t, secret)

	body := testutil.BuildWebhookPushPayload("org/repo", []string{"README.md"})
	sig := testutil.SignWebhookPayload(body, "wrong-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"webhook signed with wrong secret must return 401")
}

// TestWebhookDelivery_CorrectSignature_Returns200 verifies that a correctly
// signed push event with no .cs.md files returns 200 and reports 0 AgentRuns
// created (avoiding any network calls to the GitHub API).
func TestWebhookDelivery_CorrectSignature_Returns200(t *testing.T) {
	const secret = "webhook-regression-secret"
	wh := makeWebhookHandler(t, secret)

	// No .cs.md files: handler returns 200 with {"ok":true,"created":0}
	// without calling fetchFileContent.
	body := testutil.BuildWebhookPushPayload("org/repo", []string{"main.go", "go.mod"})
	sig := testutil.SignWebhookPayload(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"correctly signed push with no spec files should return 200")
	require.Contains(t, rec.Body.String(), `"created":0`,
		"response should report 0 AgentRuns created")
}

// TestWebhookDelivery_NonPostMethod_Returns405 verifies that non-POST methods
// are rejected with 405 Method Not Allowed.
func TestWebhookDelivery_NonPostMethod_Returns405(t *testing.T) {
	const secret = "webhook-regression-secret"
	wh := makeWebhookHandler(t, secret)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/webhooks/github", nil)
			rec := httptest.NewRecorder()
			wh.ServeHTTP(rec, req)
			require.Equal(t, http.StatusMethodNotAllowed, rec.Code,
				"%s on webhook endpoint should return 405", method)
		})
	}
}

// TestWebhookDelivery_NonPushEvent_IsIgnored verifies that non-push GitHub
// event types (e.g. pull_request) are accepted with 200 but silently ignored.
func TestWebhookDelivery_NonPushEvent_IsIgnored(t *testing.T) {
	const secret = "webhook-regression-secret"
	wh := makeWebhookHandler(t, secret)

	body := []byte(`{"action":"opened"}`)
	sig := testutil.SignWebhookPayload(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", sig)
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "ignored event type",
		"non-push event should be reported as ignored")
}

// TestWebhookDelivery_NoSecretConfigured_Returns401 verifies that when
// GITHUB_WEBHOOK_SECRET is not set the endpoint is fail-closed (returns 401).
func TestWebhookDelivery_NoSecretConfigured_Returns401(t *testing.T) {
	// Override the secret to empty — fail-closed behavior.
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")
	k8s := fake.NewClientBuilder().WithScheme(testutil.NewScheme()).Build()
	wh := server.NewWebhookHandler(context.Background(), k8s, testutil.DefaultNamespace, nil)

	body := testutil.BuildWebhookPushPayload("org/repo", []string{"README.md"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"unconfigured webhook secret must fail-closed with 401")
}

// TestWebhookDelivery_RepoAllowlist_BlockedRepo verifies that pushes from
// repos not in the GITHUB_WEBHOOK_REPOS allowlist are accepted with 200 but
// do not create AgentRuns.
func TestWebhookDelivery_RepoAllowlist_BlockedRepo(t *testing.T) {
	const secret = "webhook-regression-secret"
	t.Setenv("GITHUB_WEBHOOK_SECRET", secret)
	t.Setenv("GITHUB_WEBHOOK_REPOS", "org/allowed-repo")

	k8s := fake.NewClientBuilder().WithScheme(testutil.NewScheme()).Build()
	wh := server.NewWebhookHandler(context.Background(), k8s, testutil.DefaultNamespace, nil)

	// Push from a repo that is NOT in the allowlist.
	body := testutil.BuildWebhookPushPayload("org/blocked-repo", []string{"spec.cs.md"})
	sig := testutil.SignWebhookPayload(body, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "repo not in allowlist",
		"blocked repo should be reported as not in allowlist")

	// Verify no AgentRuns were created.
	var list aotv1alpha1.AgentRunList
	require.NoError(t, k8s.List(context.Background(), &list))
	require.Empty(t, list.Items, "blocked repo push must not create any AgentRuns")
}
