//go:build regression

// test/regression/project_provisioning_test.go — Tests that the project
// provisioning lifecycle is correctly reflected through the REST API. Uses a
// fake k8s client to simulate the configRepoReady status transition without a
// real cluster or soft-serve instance.
package regression

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/test/testutil"
)

// startProjectServer starts an in-process HTTP test server wired to a
// ProjectHandler backed by a fake k8s client. Returns the server URL, the
// underlying fake client (for direct status manipulation), and a cleanup func.
func startProjectServer(t *testing.T) (string, client.Client, func()) {
	t.Helper()

	k8s := fake.NewClientBuilder().
		WithScheme(testutil.NewScheme()).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()

	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	ph.RegisterProjectHandlers(mux)

	srv := httptest.NewServer(mux)
	return srv.URL, k8s, srv.Close
}

// TestProjectProvisioning_CreateReturns201 verifies that creating a project
// via POST /api/v1/projects returns 201 Created with the project name.
func TestProjectProvisioning_CreateReturns201(t *testing.T) {
	baseURL, _, cleanup := startProjectServer(t)
	defer cleanup()

	resp := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{
		"name":        "my-project",
		"displayName": "My Project",
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode,
		"creating a new project must return 201")

	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, "my-project", got["name"],
		"response must echo the project name")
	require.Equal(t, false, got["configRepoReady"],
		"freshly created project must have configRepoReady=false")
}

// TestProjectProvisioning_ConfigRepoReadyTransition verifies that when the
// controller marks configRepoReady=true on the k8s object, GET reflects the
// updated status and returns the config repo URL.
func TestProjectProvisioning_ConfigRepoReadyTransition(t *testing.T) {
	baseURL, k8s, cleanup := startProjectServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create the project via the API.
	resp := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{
		"name":        "provisioned-project",
		"displayName": "Provisioned Project",
	})
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Simulate the controller setting configRepoReady and the repo URL.
	project := &aotv1alpha1.Project{}
	require.NoError(t, k8s.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      "provisioned-project",
	}, project), "should be able to fetch the created Project from the fake client")

	project.Status.ConfigRepoReady = true
	project.Status.ConfigRepoURL = "http://soft-serve.default.svc.cluster.local/provisioned-project"
	require.NoError(t, k8s.Status().Update(ctx, project),
		"simulating controller status update should succeed")

	// Fetch via the API and verify the updated status is reflected.
	getResp, err := http.Get(baseURL + "/api/v1/projects/provisioned-project") //nolint:noctx
	require.NoError(t, err)
	defer getResp.Body.Close()

	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	require.Equal(t, true, got["configRepoReady"],
		"API must reflect configRepoReady=true set by the controller")
	require.Equal(t,
		"http://soft-serve.default.svc.cluster.local/provisioned-project",
		got["configRepoURL"],
		"API must reflect the configRepoURL set by the controller")
}

// TestProjectProvisioning_ConfigRepoMessageSurfaced verifies that when the
// controller sets a ConfigRepoReady=False condition with a message, the API
// surfaces that message in configRepoMessage so the UI can explain the failure.
func TestProjectProvisioning_ConfigRepoMessageSurfaced(t *testing.T) {
	baseURL, k8s, cleanup := startProjectServer(t)
	defer cleanup()

	ctx := context.Background()

	resp := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{
		"name": "stuck-project",
	})
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Simulate the controller recording a failure condition.
	project := &aotv1alpha1.Project{}
	require.NoError(t, k8s.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      "stuck-project",
	}, project))

	project.Status.ConfigRepoReady = false
	project.Status.Conditions = []metav1.Condition{
		{
			Type:               "ConfigRepoReady",
			Status:             metav1.ConditionFalse,
			Reason:             "SoftServeUnreachable",
			Message:            "Failed to reach soft-serve: connection refused",
			LastTransitionTime: metav1.Now(),
		},
	}
	require.NoError(t, k8s.Status().Update(ctx, project))

	getResp, err := http.Get(baseURL + "/api/v1/projects/stuck-project") //nolint:noctx
	require.NoError(t, err)
	defer getResp.Body.Close()

	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	require.Equal(t, false, got["configRepoReady"])
	require.Equal(t, "Failed to reach soft-serve: connection refused", got["configRepoMessage"],
		"configRepoMessage must surface the condition message when configRepoReady is false")
}

// TestProjectProvisioning_GetNotFound verifies that fetching a non-existent
// project returns 404.
func TestProjectProvisioning_GetNotFound(t *testing.T) {
	baseURL, _, cleanup := startProjectServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/v1/projects/does-not-exist") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"GET for a non-existent project must return 404")
}

// TestProjectProvisioning_DuplicateCreateReturns409 verifies that creating a
// project with a name that already exists returns 409 Conflict.
func TestProjectProvisioning_DuplicateCreateReturns409(t *testing.T) {
	baseURL, _, cleanup := startProjectServer(t)
	defer cleanup()

	first := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{
		"name": "duplicate-project",
	})
	first.Body.Close()
	require.Equal(t, http.StatusCreated, first.StatusCode)

	second := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{
		"name": "duplicate-project",
	})
	second.Body.Close()
	require.Equal(t, http.StatusConflict, second.StatusCode,
		"creating a project with a duplicate name must return 409")
}

// TestProjectProvisioning_ListReflectsCreatedProjects verifies that projects
// created via POST appear in GET /api/v1/projects.
func TestProjectProvisioning_ListReflectsCreatedProjects(t *testing.T) {
	baseURL, _, cleanup := startProjectServer(t)
	defer cleanup()

	for _, name := range []string{"proj-a", "proj-b"} {
		r := testutil.PostJSON(t, baseURL+"/api/v1/projects", map[string]interface{}{"name": name})
		r.Body.Close()
		require.Equal(t, http.StatusCreated, r.StatusCode)
	}

	listResp, err := http.Get(baseURL + "/api/v1/projects") //nolint:noctx
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	var projects []map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&projects))
	require.Len(t, projects, 2,
		"list must return all created projects")

	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p["name"].(string))
	}
	require.ElementsMatch(t, []string{"proj-a", "proj-b"}, names)
}
