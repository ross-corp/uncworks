// test/bdd/suite_test.go — Ginkgo BDD suite bootstrap for AOT service scenarios.
// Run with: go test ./test/bdd/... -v
// or:       task test:bdd
package bdd_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/test/testutil"
)

// bddScheme is the shared k8s runtime.Scheme for BDD tests that construct
// a fake k8s client directly (e.g. auth/rate-limit specs that bypass newTestEnv).
var bddScheme = testutil.NewScheme()

func TestBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AOT BDD Suite")
}

// testEnv holds the in-process server and client for a single spec.
// Call newTestEnv() in BeforeEach; call cleanup() in AfterEach.
type testEnv struct {
	Client    apiv1connect.AOTServiceClient
	K8s       client.Client
	cleanupFn func()
}

// newTestEnv creates a fresh in-process AOT server for a Ginkgo spec.
// Uses NewAOTServerNoT because Ginkgo BeforeEach blocks do not have a *testing.T.
func newTestEnv() *testEnv {
	c, k8sClient, closeFn := testutil.NewAOTServerNoT()
	return &testEnv{
		Client:    c,
		K8s:       k8sClient,
		cleanupFn: closeFn,
	}
}

func (e *testEnv) cleanup() {
	e.cleanupFn()
}
