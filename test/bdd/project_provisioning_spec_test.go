// test/bdd/project_provisioning_spec_test.go — BDD scenarios for project provisioning.
//
// Scenarios covered:
//   - Create → Get → List flow
//   - Config repo readiness transition (simulated controller)
//   - Duplicate project name returns 409
//   - Non-existent project returns 404
package bdd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
)

// projectTestEnv holds an HTTP test server wired to the REST project API.
type projectTestEnv struct {
	BaseURL   string
	K8s       client.Client
	cleanupFn func()
}

func newProjectTestEnv() *projectTestEnv {
	k8s := fake.NewClientBuilder().
		WithScheme(bddScheme).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()

	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)

	srv := httptest.NewServer(mux)
	return &projectTestEnv{
		BaseURL:   srv.URL,
		K8s:       k8s,
		cleanupFn: srv.Close,
	}
}

func (e *projectTestEnv) cleanup() { e.cleanupFn() }

// postProject sends a JSON POST to /api/v1/projects and returns the response.
func (e *projectTestEnv) postProject(body map[string]interface{}) *http.Response {
	b, err := json.Marshal(body)
	Expect(err).NotTo(HaveOccurred())
	resp, err := http.Post(e.BaseURL+"/api/v1/projects", "application/json", bytes.NewReader(b)) //nolint:noctx
	Expect(err).NotTo(HaveOccurred())
	return resp
}

// getProject sends a GET to /api/v1/projects/{name} and returns the response.
func (e *projectTestEnv) getProject(name string) *http.Response {
	resp, err := http.Get(e.BaseURL + "/api/v1/projects/" + name) //nolint:noctx
	Expect(err).NotTo(HaveOccurred())
	return resp
}

// listProjects sends a GET to /api/v1/projects and returns the decoded slice.
func (e *projectTestEnv) listProjects() []map[string]interface{} {
	resp, err := http.Get(e.BaseURL + "/api/v1/projects") //nolint:noctx
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	var projects []map[string]interface{}
	Expect(json.NewDecoder(resp.Body).Decode(&projects)).To(Succeed())
	return projects
}

var _ = Describe("Project Provisioning", func() {
	var (
		env *projectTestEnv
		ctx context.Context
	)

	BeforeEach(func() {
		env = newProjectTestEnv()
		ctx = context.Background()
	})

	AfterEach(func() {
		env.cleanup()
	})

	Describe("Create → Get → List flow", func() {
		Context("Given a project is created via the API", func() {
			BeforeEach(func() {
				resp := env.postProject(map[string]interface{}{
					"name":        "my-project",
					"displayName": "My Project",
				})
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			})

			It("returns the project on GET with configRepoReady=false initially", func() {
				resp := env.getProject("my-project")
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var got map[string]interface{}
				Expect(json.NewDecoder(resp.Body).Decode(&got)).To(Succeed())
				Expect(got["name"]).To(Equal("my-project"))
				Expect(got["configRepoReady"]).To(BeFalse())
			})

			It("appears in the projects list", func() {
				projects := env.listProjects()
				names := make([]string, 0, len(projects))
				for _, p := range projects {
					names = append(names, p["name"].(string))
				}
				Expect(names).To(ContainElement("my-project"))
			})
		})

		Context("Given two projects are created", func() {
			BeforeEach(func() {
				for _, name := range []string{"proj-alpha", "proj-beta"} {
					resp := env.postProject(map[string]interface{}{"name": name})
					resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				}
			})

			It("lists both projects", func() {
				projects := env.listProjects()
				names := make([]string, 0, len(projects))
				for _, p := range projects {
					names = append(names, p["name"].(string))
				}
				Expect(names).To(ConsistOf("proj-alpha", "proj-beta"))
			})
		})
	})

	Describe("Config repo readiness transition", func() {
		Context("Given a project is created and initially not ready", func() {
			BeforeEach(func() {
				resp := env.postProject(map[string]interface{}{"name": "provisioned-project"})
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			})

			When("the controller marks configRepoReady=true with a repo URL", func() {
				BeforeEach(func() {
					project := &aotv1alpha1.Project{}
					Expect(env.K8s.Get(ctx, client.ObjectKey{
						Namespace: "default",
						Name:      "provisioned-project",
					}, project)).To(Succeed())

					project.Status.ConfigRepoReady = true
					project.Status.ConfigRepoURL = "http://soft-serve.default.svc/provisioned-project"
					Expect(env.K8s.Status().Update(ctx, project)).To(Succeed())
				})

				It("reflects configRepoReady=true and the repo URL in GET", func() {
					resp := env.getProject("provisioned-project")
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					var got map[string]interface{}
					Expect(json.NewDecoder(resp.Body).Decode(&got)).To(Succeed())
					Expect(got["configRepoReady"]).To(BeTrue())
					Expect(got["configRepoURL"]).To(Equal("http://soft-serve.default.svc/provisioned-project"))
				})
			})
		})
	})

	Describe("Error paths", func() {
		Context("When fetching a project that does not exist", func() {
			It("returns 404", func() {
				resp := env.getProject("does-not-exist")
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("When creating a project with a duplicate name", func() {
			BeforeEach(func() {
				first := env.postProject(map[string]interface{}{"name": "duplicate-project"})
				first.Body.Close()
				Expect(first.StatusCode).To(Equal(http.StatusCreated))
			})

			It("returns 409 Conflict", func() {
				second := env.postProject(map[string]interface{}{"name": "duplicate-project"})
				defer second.Body.Close()
				Expect(second.StatusCode).To(Equal(http.StatusConflict))
			})
		})
	})
})
