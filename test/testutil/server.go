// test/testutil/server.go — Shared helpers for spinning up an in-process
// ConnectRPC AOT service server backed by a fake k8s client.
// All layer2, contract, regression, and bdd packages use the same setup
// pattern; this file is the single source of truth for that pattern.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/server"
)

// NewAOTServer starts an httptest TLS server backed by a fake k8s client and
// a NoOpEventBus.  It returns:
//   - the ConnectRPC client for the AOT service
//   - the underlying fake k8s client (for direct CRD status mutations)
//   - a cleanup function that must be deferred by the caller
//
// The server uses NewScheme() so AgentRun and the standard k8s types are
// registered.
func NewAOTServer(t *testing.T) (apiv1connect.AOTServiceClient, client.Client, func()) {
	t.Helper()

	k8sClient := fake.NewClientBuilder().
		WithScheme(NewScheme()).
		WithStatusSubresource(&aotv1alpha1.AgentRun{}).
		Build()

	svc := server.NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	c := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return c, k8sClient, srv.Close
}

// NewFakeK8sClient returns a fake k8s client registered with NewScheme() and
// AgentRun as a status subresource.  Useful when tests need the k8s client
// without the full HTTP server stack.
func NewFakeK8sClient(t *testing.T, statusSubresources ...client.Object) client.Client {
	t.Helper()

	b := fake.NewClientBuilder().WithScheme(NewScheme())
	if len(statusSubresources) > 0 {
		b = b.WithStatusSubresource(statusSubresources...)
	}
	return b.Build()
}
