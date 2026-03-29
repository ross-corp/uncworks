//go:build regression

// test/regression/auth_boundary_test.go — Tests that unauthenticated requests
// to protected endpoints return 401. The auth middleware in cmd/apiserver/main.go
// is not exported, so this file replicates the exact pattern under test to
// verify the design contract holds: Bearer-token auth gates the REST API.
package regression

import (
	"crypto/subtle"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
)

// withTestAuth replicates the withAuth function from cmd/apiserver/main.go.
// When apiKey is empty, all requests are passed through (no auth).
// When apiKey is set, requests missing a valid Bearer token receive 401.
func withTestAuth(h http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and webhooks (which use HMAC auth).
		p := r.URL.Path
		if p == "/healthz" || p == "/readyz" || p == "/api/v1/webhooks/github" {
			h.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		token := ""
		if auth != "" {
			token = strings.TrimPrefix(auth, "Bearer ")
			if token == auth {
				token = "" // not a Bearer token
			}
		}
		// Fallback: query param for WebSocket clients.
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func newRegressionScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = aotv1alpha1.AddToScheme(s)
	return s
}

// TestAuthBoundary_ProjectsEndpoint_NoToken verifies that GET /api/v1/projects
// returns 401 when no auth token is provided and the auth middleware is active.
func TestAuthBoundary_ProjectsEndpoint_NoToken(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newRegressionScheme()).Build()
	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)

	protected := withTestAuth(mux, "secret-api-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"unauthenticated GET /api/v1/projects should return 401")
}

// TestAuthBoundary_ProjectsEndpoint_WrongToken verifies that a wrong token
// also receives 401.
func TestAuthBoundary_ProjectsEndpoint_WrongToken(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newRegressionScheme()).Build()
	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)

	protected := withTestAuth(mux, "correct-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"wrong Bearer token for GET /api/v1/projects should return 401")
}

// TestAuthBoundary_ProjectsEndpoint_ValidToken verifies that a correct token
// passes through to the underlying handler.
func TestAuthBoundary_ProjectsEndpoint_ValidToken(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newRegressionScheme()).Build()
	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)

	protected := withTestAuth(mux, "correct-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer correct-key")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"valid Bearer token for GET /api/v1/projects should return 200")
}

// TestAuthBoundary_CountsEndpoint_NoToken verifies that GET /api/v1/counts
// returns 401 when no token is provided.
func TestAuthBoundary_CountsEndpoint_NoToken(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newRegressionScheme()).Build()
	mux := http.NewServeMux()
	ch := &server.CountsHandler{K8sClient: k8s, Namespace: "default"}
	ch.RegisterCountsHandlers(mux)

	protected := withTestAuth(mux, "secret-api-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"unauthenticated GET /api/v1/counts should return 401")
}

// TestAuthBoundary_HealthEndpoints_AreExempt verifies that /healthz and
// /readyz are not gated by the auth middleware.
func TestAuthBoundary_HealthEndpoints_AreExempt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := withTestAuth(mux, "secret-api-key")

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// Deliberately send no token.
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code,
			"health endpoint %s should bypass auth and return 200", path)
	}
}

// TestAuthBoundary_WebhookEndpoint_IsExempt verifies that the webhook endpoint
// is exempt from the Bearer auth layer (it uses its own HMAC auth).
func TestAuthBoundary_WebhookEndpoint_IsExempt(t *testing.T) {
	mux := http.NewServeMux()
	// Register a dummy webhook handler to confirm routing reaches it.
	mux.HandleFunc("/api/v1/webhooks/github", func(w http.ResponseWriter, _ *http.Request) {
		// Real handler returns 401 for missing HMAC secret — that's fine.
		// The point is the auth middleware must not return 401 first.
		w.WriteHeader(http.StatusUnauthorized) // HMAC layer rejects it, not Bearer layer
	})

	protected := withTestAuth(mux, "secret-api-key")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", nil)
	// No Bearer token — the auth middleware should not intercept this path.
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	// The dummy handler returns 401 (HMAC rejection), not the auth middleware 401.
	// Either way the body should not contain "invalid or missing API key".
	require.NotContains(t, rec.Body.String(), "invalid or missing API key",
		"webhook endpoint should not be rejected by the Bearer auth layer")
}

// TestAuthBoundary_QueryParamToken verifies that the ?token= query parameter
// is accepted for WebSocket-style clients that cannot set headers.
func TestAuthBoundary_QueryParamToken(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newRegressionScheme()).Build()
	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)

	protected := withTestAuth(mux, "correct-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects?token=correct-key", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"query-param token should be accepted as a valid auth method")
}
