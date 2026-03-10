package test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AgentRun CRD Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "deploy", "crds")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = aotv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("AgentRun CRD", func() {
	const namespace = "default"

	Context("when creating an AgentRun with Pod backend", func() {
		It("should create and retrieve the resource successfully", func() {
			agentRun := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-run",
					Namespace: namespace,
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: aotv1alpha1.BackendPod,
					RepoURL: "https://github.com/example/repo.git",
					Branch:  "main",
					Prompt:  "Fix the failing tests",
				},
			}

			Expect(k8sClient.Create(ctx, agentRun)).Should(Succeed())

			fetched := &aotv1alpha1.AgentRun{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-pod-run",
					Namespace: namespace,
				}, fetched)
			}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

			Expect(fetched.Spec.Backend).To(Equal(aotv1alpha1.BackendPod))
			Expect(fetched.Spec.RepoURL).To(Equal("https://github.com/example/repo.git"))
			Expect(fetched.Spec.Prompt).To(Equal("Fix the failing tests"))
		})
	})

	Context("when creating an AgentRun with KubeVirt backend", func() {
		It("should store the KubeVirt config in the CRD", func() {
			agentRun := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-kubevirt-run",
					Namespace: namespace,
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: aotv1alpha1.BackendKubeVirt,
					RepoURL: "https://github.com/example/repo.git",
					Prompt:  "Refactor the API",
					KubeVirtConfig: &aotv1alpha1.KubeVirtBackendConfig{
						CPUs:     4,
						MemoryMB: 8192,
						DiskGB:   40,
					},
				},
			}

			Expect(k8sClient.Create(ctx, agentRun)).Should(Succeed())

			fetched := &aotv1alpha1.AgentRun{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-kubevirt-run",
					Namespace: namespace,
				}, fetched)
			}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

			Expect(fetched.Spec.Backend).To(Equal(aotv1alpha1.BackendKubeVirt))
			Expect(fetched.Spec.KubeVirtConfig).NotTo(BeNil())
			Expect(fetched.Spec.KubeVirtConfig.CPUs).To(Equal(int32(4)))
			Expect(fetched.Spec.KubeVirtConfig.MemoryMB).To(Equal(int32(8192)))
		})
	})

	Context("when creating an AgentRun with External backend", func() {
		It("should store the External config in the CRD", func() {
			agentRun := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-external-run",
					Namespace: namespace,
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: aotv1alpha1.BackendExternal,
					RepoURL: "https://github.com/example/repo.git",
					Prompt:  "Deploy the service",
					ExternalConfig: &aotv1alpha1.ExternalBackendConfig{
						Host:         "192.168.1.100",
						Port:         2222,
						User:         "agent",
						SSHKeySecret: "ssh-key-secret",
					},
				},
			}

			Expect(k8sClient.Create(ctx, agentRun)).Should(Succeed())

			fetched := &aotv1alpha1.AgentRun{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-external-run",
					Namespace: namespace,
				}, fetched)
			}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

			Expect(fetched.Spec.Backend).To(Equal(aotv1alpha1.BackendExternal))
			Expect(fetched.Spec.ExternalConfig).NotTo(BeNil())
			Expect(fetched.Spec.ExternalConfig.Host).To(Equal("192.168.1.100"))
			Expect(fetched.Spec.ExternalConfig.Port).To(Equal(int32(2222)))
		})
	})

	Context("when updating AgentRun status", func() {
		It("should update the phase via status subresource", func() {
			agentRun := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-status-update",
					Namespace: namespace,
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: aotv1alpha1.BackendPod,
					RepoURL: "https://github.com/example/repo.git",
					Prompt:  "Run tests",
				},
			}

			Expect(k8sClient.Create(ctx, agentRun)).Should(Succeed())

			fetched := &aotv1alpha1.AgentRun{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-status-update",
					Namespace: namespace,
				}, fetched)
			}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

			now := metav1.Now()
			fetched.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
			fetched.Status.Message = "Agent pod started"
			fetched.Status.PodName = "test-status-update-pod"
			fetched.Status.StartedAt = &now
			Expect(k8sClient.Status().Update(ctx, fetched)).Should(Succeed())

			updated := &aotv1alpha1.AgentRun{}
			Eventually(func() aotv1alpha1.AgentRunPhase {
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-status-update",
					Namespace: namespace,
				}, updated)
				return updated.Status.Phase
			}, 10*time.Second, 250*time.Millisecond).Should(Equal(aotv1alpha1.AgentRunPhaseRunning))

			Expect(updated.Status.PodName).To(Equal("test-status-update-pod"))
		})
	})

	Context("when deleting an AgentRun", func() {
		It("should remove the resource", func() {
			agentRun := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delete-run",
					Namespace: namespace,
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend: aotv1alpha1.BackendPod,
					RepoURL: "https://github.com/example/repo.git",
					Prompt:  "Delete me",
				},
			}

			Expect(k8sClient.Create(ctx, agentRun)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, agentRun)).Should(Succeed())

			deleted := &aotv1alpha1.AgentRun{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-delete-run",
					Namespace: namespace,
				}, deleted)
				return err != nil
			}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())
		})
	})

	Context("when listing AgentRuns", func() {
		It("should return all created resources", func() {
			list := &aotv1alpha1.AgentRunList{}
			Expect(k8sClient.List(ctx, list, client.InNamespace(namespace))).Should(Succeed())
			// We created several above (some deleted), so at least 3 should remain
			Expect(len(list.Items)).To(BeNumerically(">=", 3))
		})
	})
})
