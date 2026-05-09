// test/bdd/run_lifecycle_spec_test.go — BDD scenarios for the agent run lifecycle.
//
// Scenarios covered:
//   - Full happy-path lifecycle: PENDING → RUNNING → SUCCEEDED
//   - Create → list → cancel flow
//   - Cancellation from PENDING phase
//   - Fetching a non-existent run
package bdd_test

import (
	"context"

	"connectrpc.com/connect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

var _ = Describe("Agent Run Lifecycle", func() {
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

	// --- Happy-path lifecycle ---

	Describe("Full lifecycle: PENDING → RUNNING → SUCCEEDED", func() {
		var runID string

		Context("Given a new agent run is created", func() {
			BeforeEach(func() {
				resp, err := env.Client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
					Spec: &apiv1.AgentRunSpec{
						Backend: apiv1.Backend_BACKEND_POD,
						Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
						Prompt:  "Fix the auth layer",
					},
				}))
				Expect(err).NotTo(HaveOccurred())
				runID = resp.Msg.AgentRun.Id
				Expect(runID).NotTo(BeEmpty())
			})

			It("starts in PENDING phase", func() {
				resp, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Msg.Status.Phase).To(Equal(apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING))
			})

			Context("When the controller advances it to RUNNING", func() {
				BeforeEach(func() {
					crd := &aotv1alpha1.AgentRun{}
					Expect(env.K8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: runID}, crd)).To(Succeed())
					crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
					crd.Status.Message = "Agent pod is running"
					crd.Status.StartedAt = &metav1.Time{Time: metav1.Now().Time}
					Expect(env.K8s.Status().Update(ctx, crd)).To(Succeed())
				})

				It("reflects RUNNING phase via the API", func() {
					resp, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.Msg.Status.Phase).To(Equal(apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING))
				})

				Context("When the controller marks it as SUCCEEDED", func() {
					BeforeEach(func() {
						crd := &aotv1alpha1.AgentRun{}
						Expect(env.K8s.Get(ctx, client.ObjectKey{Namespace: "default", Name: runID}, crd)).To(Succeed())
						now := metav1.Now()
						crd.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
						crd.Status.Message = "Task completed successfully"
						crd.Status.CompletedAt = &metav1.Time{Time: now.Time}
						Expect(env.K8s.Status().Update(ctx, crd)).To(Succeed())
					})

					It("reflects SUCCEEDED phase and the completion message via the API", func() {
						resp, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.Msg.Status.Phase).To(Equal(apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED))
						Expect(resp.Msg.Status.Message).To(Equal("Task completed successfully"))
					})
				})
			})
		})
	})

	// --- Create → List → Cancel flow ---

	Describe("Create → List → Cancel flow", func() {
		Context("Given three runs are created", func() {
			BeforeEach(func() {
				for i := 0; i < 3; i++ {
					_, err := env.Client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
						Spec: &apiv1.AgentRunSpec{
							Backend: apiv1.Backend_BACKEND_POD,
							Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
							Prompt:  "Task",
						},
					}))
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("lists all three runs", func() {
				resp, err := env.Client.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Msg.AgentRuns).To(HaveLen(3))
			})

			Context("When one run is cancelled", func() {
				var cancelledID string

				BeforeEach(func() {
					listResp, err := env.Client.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
					Expect(err).NotTo(HaveOccurred())
					Expect(listResp.Msg.AgentRuns).NotTo(BeEmpty())
					cancelledID = listResp.Msg.AgentRuns[0].Id

					_, err = env.Client.CancelAgentRun(ctx, connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: cancelledID}))
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns the run on subsequent Get", func() {
					resp, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: cancelledID}))
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.Msg.Id).To(Equal(cancelledID))
				})
			})
		})
	})

	// --- Error paths ---

	Describe("Error paths", func() {
		Context("When fetching a run that does not exist", func() {
			It("returns a NotFound error", func() {
				_, err := env.Client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{
					Id: "ar-notfound",
				}))
				Expect(err).To(HaveOccurred())
				Expect(connect.CodeOf(err)).To(Equal(connect.CodeNotFound))
			})
		})

		Context("When cancelling a run that does not exist", func() {
			It("returns a NotFound error", func() {
				_, err := env.Client.CancelAgentRun(ctx, connect.NewRequest(&apiv1.CancelAgentRunRequest{
					Id: "ar-notfound",
				}))
				Expect(err).To(HaveOccurred())
				Expect(connect.CodeOf(err)).To(Equal(connect.CodeNotFound))
			})
		})

		Context("When creating a run with no spec", func() {
			It("returns an InvalidArgument error", func() {
				_, err := env.Client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{}))
				Expect(err).To(HaveOccurred())
				Expect(connect.CodeOf(err)).To(Equal(connect.CodeInvalidArgument))
			})
		})
	})
})
