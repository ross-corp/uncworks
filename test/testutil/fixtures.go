// test/testutil/fixtures.go — Shared test fixture data.
// Centralises the hardcoded values that appear verbatim across layer2,
// contract, regression, and bdd tests so a single change propagates everywhere.
package testutil

import (
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

const (
	// DefaultRepoURL is the canonical test repository URL used in AgentRunSpec
	// fixtures throughout the test suite.
	DefaultRepoURL = "https://github.com/example/repo.git"

	// DefaultNamespace is the Kubernetes namespace used by all in-process fake
	// k8s clients in the test suite.
	DefaultNamespace = "default"
)

// DefaultRepo returns the single-element Repository slice used in most
// AgentRunSpec fixtures.
func DefaultRepo() []*apiv1.Repository {
	return []*apiv1.Repository{{Url: DefaultRepoURL}}
}

// MinimalSpec returns an AgentRunSpec with the minimum required fields
// populated for a valid CreateAgentRun request.
func MinimalSpec(prompt string) *apiv1.AgentRunSpec {
	return &apiv1.AgentRunSpec{
		Backend: apiv1.Backend_BACKEND_POD,
		Repos:   DefaultRepo(),
		Prompt:  prompt,
	}
}
