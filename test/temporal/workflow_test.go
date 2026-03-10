// Package temporal contains unit tests for Temporal workflow logic.
//
// These tests use go.temporal.io/sdk/testsuite to test workflow logic
// with mocked activities and fast-forwarded timers.
//
// BLOCKED: These tests require the workflow implementation from the
// temporal-workflow-engine change. The test structure and interfaces
// are defined here so they can be filled in once workflows exist.
package temporal

// The following test stubs document the expected workflow behavior.
// Each will be implemented when the AgentRunWorkflow is created.
//
// Test plan:
// - TestWorkflow_HappyPath: create pod → hydrate → start → complete → cleanup
// - TestWorkflow_TTLExpiry: fast-forward timer → stop agent → cleanup
// - TestWorkflow_HITLSignal: send "human-input" signal → ForwardHumanInput called
// - TestWorkflow_CancelSignal: send "cancel" signal → StopAgent → CleanupPod
// - TestWorkflow_SpawnJunior: start child workflow → child completes → parent continues
// - TestWorkflow_CompensationOnFailure: CreatePod ok → StartAgent fail → CleanupPod called
// - TestWorkflow_GetStateQuery: verify phase at each lifecycle stage
// - TestWorkflow_Integration: real workflow execution with temporal-cli dev server
