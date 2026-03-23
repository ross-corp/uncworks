package contract

import (
	"os"
	"testing"
)

// TestBoundary_HelmCRD_MatchesDeployCRD verifies that the CRD files in
// deploy/helm/aot/crds/ are identical to deploy/crds/.
func TestBoundary_HelmCRD_MatchesDeployCRD(t *testing.T) {
	crds := []string{
		"agentrun-crd.yaml",
		"project-crd.yaml",
		"runtemplate-crd.yaml",
		"chain-crd.yaml",
		"chainrun-crd.yaml",
		"schedule-crd.yaml",
	}

	for _, crd := range crds {
		deploy, err := os.ReadFile("../../deploy/crds/" + crd)
		if err != nil {
			t.Fatalf("read deploy/crds/%s: %v", crd, err)
		}
		helm, err := os.ReadFile("../../deploy/helm/aot/crds/" + crd)
		if err != nil {
			t.Fatalf("read deploy/helm/aot/crds/%s: %v", crd, err)
		}
		if string(deploy) != string(helm) {
			t.Errorf("CRD %s differs between deploy/crds/ and deploy/helm/aot/crds/", crd)
		}
	}
}
