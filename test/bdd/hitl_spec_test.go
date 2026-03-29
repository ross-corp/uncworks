// test/bdd/hitl_spec_test.go — BDD scenarios for the Human-in-the-Loop (HITL) flow.
//
// Scenarios covered:
//   - SendHumanInput is rejected when run is not waiting (FailedPrecondition)
//   - SendHumanInput is rejected when run does not exist (NotFound)
//   - SendHumanInput succeeds when run is in WaitingForInput phase
package bdd_test

import (
	"context"

	"connectrpc.com/connect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

var _ = Describe("Human-in-the-Loop (HITL) flow", func() {
	var (
		env *testEnv
		ctx context.Context
	)

	BeforeEach(func() {
		env = newTestEnv()
		ctx = context.Background()
	})

	AfterEach(func() {
		env.cleanup()
	})

	Describe("Sending human input to a run that is not waiting", func() {
		Context("Given a freshly created run in PENDING phase", func() {
			var runID string

			BeforeEach(func() {
				resp, err := env.Client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
					Spec: &apiv1.AgentRunSpec{
						Backend: apiv1.Backend_BACKEND_POD,
						Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
						Prompt:  "Do something that needs human input",
					},
				}))
				Expect(err).NotTo(HaveOccurred())
				runID = resp.Msg.AgentRun.Id
			})

			When("human input is sent before the agent requests it", func() {
				It("returns FailedPrecondition because the run is not waiting for input", func() {
					_, err := env.Client.SendHumanInput(ctx, connect.NewRequest(&apiv1.SendHumanInputRequest{
						AgentRunId: runID,
						Input:      "here is my answer",
					}))
					Expect(err).To(HaveOccurred())
					Expect(connect.CodeOf(err)).To(Equal(connect.CodeFailedPrecondition))
				})
			})
		})
	})

	Describe("Sending human input when the run does not exist", func() {
		When("a non-existent run ID is provided", func() {
			It("returns NotFound", func() {
				_, err := env.Client.SendHumanInput(ctx, connect.NewRequest(&apiv1.SendHumanInputRequest{
					AgentRunId: "ar-does-not-exist",
					Input:      "hello",
				}))
				Expect(err).To(HaveOccurred())
				Expect(connect.CodeOf(err)).To(Equal(connect.CodeNotFound))
			})
		})
	})

	Describe("HITL precondition check: run in WaitingForInput phase", func() {
		// SendHumanInput passes the phase-gate and then calls Temporal to signal the workflow.
		// In the in-process test environment there is no Temporal client, so the server returns
		// CodeUnavailable after passing all precondition checks. This scenario verifies that:
		//   1. The run is visible in WaitingForInput phase.
		//   2. SendHumanInput does NOT return FailedPrecondition (the phase check passes).
		//   3. The only remaining error is CodeUnavailable (Temporal not wired), confirming the
		//      gate reached the Temporal dispatch step.
		Context("Given a run exists and the controller advances it to WaitingForInput", func() {
			var runID string

			BeforeEach(func() {
				// Create the run.
				resp, err := env.Client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
					Spec: &apiv1.AgentRunSpec{
						Backend: apiv1.Backend_BACKEND_POD,
						Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
						Prompt:  "Refactor the auth module, ask if you are unsure",
					},
				}))
				Expect(err).NotTo(HaveOccurred())
				runID = resp.Msg.AgentRun.Id

				// Simulate the controller pausing the run and requesting human input.
				crd := &aotv1alpha1.AgentRun{}
				Expect(env.K8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: runID}, crd)).To(Succeed())
				crd.Status.Phase = aotv1alpha1.AgentRunPhaseWaitingForInput
				crd.Status.Message = "Should I delete the legacy OAuth handler?"
				Expect(env.K8s.Status().Update(ctx, crd)).To(Succeed())
			})

			It("reflects WaitingForInput phase via GetAgentRun", func() {
				resp, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Msg.Status.Phase).To(Equal(apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT))
			})

			When("the user sends their answer", func() {
				It("passes the precondition check and reaches Temporal dispatch (CodeUnavailable without a live Temporal client)", func() {
					// Without Temporal wired, the server returns CodeUnavailable — meaning it passed
					// NotFound and FailedPrecondition checks and only failed at Temporal signal dispatch.
					_, err := env.Client.SendHumanInput(ctx, connect.NewRequest(&apiv1.SendHumanInputRequest{
						AgentRunId: runID,
						Input:      "Yes, delete it — it has no active callers.",
					}))
					Expect(err).To(HaveOccurred())
					Expect(connect.CodeOf(err)).To(Equal(connect.CodeUnavailable),
						"CodeUnavailable means preconditions passed; only Temporal dispatch is missing")
				})
			})
		})
	})
})
