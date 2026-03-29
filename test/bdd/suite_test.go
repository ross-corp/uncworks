// test/bdd/suite_test.go — Ginkgo BDD suite bootstrap for AOT service scenarios.
// Run with: go test ./test/bdd/... -v
// or:       task test:bdd
package bdd_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/server"
)

func TestBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AOT BDD Suite")
}

// bddScheme is shared across all specs in the suite.
var bddScheme *runtime.Scheme

func init() {
	bddScheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(bddScheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(bddScheme))
}

// testEnv holds the in-process server and client for a single spec.
// Call newTestEnv() in BeforeEach; call cleanup() in AfterEach.
type testEnv struct {
	Client    apiv1connect.AOTServiceClient
	K8s       client.Client
	cleanupFn func()
}

func newTestEnv() *testEnv {
	k8sClient := fake.NewClientBuilder().
		WithScheme(bddScheme).
		WithStatusSubresource(&aotv1alpha1.AgentRun{}).
		Build()

	svc := server.NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	return &testEnv{
		Client:    apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL),
		K8s:       k8sClient,
		cleanupFn: srv.Close,
	}
}

func (e *testEnv) cleanup() {
	e.cleanupFn()
}
