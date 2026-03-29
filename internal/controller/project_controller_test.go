package controller

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

// mockRepoManager implements softserve.RepoManager for testing.
type mockRepoManager struct {
	createdRepos []string
	deletedRepos []string
	scaffolded   []softserve.ScaffoldProject
	existsMap    map[string]bool
	failCreate   bool
	failScaffold bool
	files        map[string]map[string]string // repo -> path -> content
}

func newMockRepoManager() *mockRepoManager {
	return &mockRepoManager{
		existsMap: make(map[string]bool),
		files:     make(map[string]map[string]string),
	}
}

func (m *mockRepoManager) CreateRepo(name string) error {
	if m.failCreate {
		return fmt.Errorf("mock: create failed")
	}
	m.createdRepos = append(m.createdRepos, name)
	m.existsMap[name] = true
	return nil
}

func (m *mockRepoManager) DeleteRepo(name string) error {
	m.deletedRepos = append(m.deletedRepos, name)
	delete(m.existsMap, name)
	return nil
}

func (m *mockRepoManager) RepoExists(name string) (bool, error) {
	return m.existsMap[name], nil
}

func (m *mockRepoManager) CloneURL(name string) string {
	return "ssh://soft-serve.test:23231/" + name + ".git"
}

func (m *mockRepoManager) ScaffoldAndPush(scaffold softserve.ScaffoldProject) error {
	if m.failScaffold {
		return fmt.Errorf("mock: scaffold failed")
	}
	m.scaffolded = append(m.scaffolded, scaffold)
	return nil
}

func (m *mockRepoManager) ReadFile(repoName, filePath string) (string, error) {
	if repo, ok := m.files[repoName]; ok {
		if content, ok := repo[filePath]; ok {
			return content, nil
		}
	}
	return "", fmt.Errorf("file not found: %s/%s", repoName, filePath)
}

func (m *mockRepoManager) WriteFile(repoName, filePath, content, commitMsg string) error {
	if m.files[repoName] == nil {
		m.files[repoName] = make(map[string]string)
	}
	m.files[repoName][filePath] = content
	return nil
}

func (m *mockRepoManager) ListFiles(repoName string) ([]string, error) {
	var files []string
	if repo, ok := m.files[repoName]; ok {
		for path := range repo {
			files = append(files, path)
		}
	}
	return files, nil
}

func TestProjectReconciler_CreateProject(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test-proj", Namespace: "aot"},
		Spec: aotv1alpha1.ProjectSpec{
			DisplayName: "Test Project",
			Devbox:      &aotv1alpha1.DevboxConfig{Packages: []string{"go@1.22"}},
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(project).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()
	mock := newMockRepoManager()

	r := &ProjectReconciler{Client: k8s, SoftServe: mock}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-proj", Namespace: "aot"}}
	// First reconcile: adds finalizer and returns early.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}
	// Second reconcile: creates repo and updates status.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (create): %v", err)
	}

	// Verify repo was created
	if len(mock.createdRepos) != 1 || mock.createdRepos[0] != "test-proj" {
		t.Errorf("expected repo 'test-proj' created, got: %v", mock.createdRepos)
	}

	// Verify scaffold was pushed with correct packages
	if len(mock.scaffolded) != 1 {
		t.Fatalf("expected 1 scaffold, got %d", len(mock.scaffolded))
	}
	if mock.scaffolded[0].Name != "test-proj" {
		t.Errorf("scaffold name = %q", mock.scaffolded[0].Name)
	}
	if len(mock.scaffolded[0].Packages) != 1 || mock.scaffolded[0].Packages[0] != "go@1.22" {
		t.Errorf("scaffold packages = %v", mock.scaffolded[0].Packages)
	}

	// Verify status was updated
	var updated aotv1alpha1.Project
	if err := k8s.Get(context.Background(), types.NamespacedName{Name: "test-proj", Namespace: "aot"}, &updated); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !updated.Status.ConfigRepoReady {
		t.Error("expected ConfigRepoReady=true")
	}
	if updated.Status.ConfigRepoURL == "" {
		t.Error("expected ConfigRepoURL to be set")
	}

	// Verify finalizer was added
	if !containsFinalizer(updated.Finalizers, projectFinalizerName) {
		t.Error("expected finalizer to be added")
	}
}

func TestProjectReconciler_SkipExistingRepo(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "existing-proj", Namespace: "aot"},
		Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Existing"},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(project).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()
	mock := newMockRepoManager()
	mock.existsMap["existing-proj"] = true // repo already exists

	r := &ProjectReconciler{Client: k8s, SoftServe: mock}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "existing-proj", Namespace: "aot"}}
	// First reconcile: adds finalizer and returns early.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}
	// Second reconcile: checks repo existence (already exists, so no create).
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (check): %v", err)
	}

	// Should NOT have created the repo again
	if len(mock.createdRepos) != 0 {
		t.Errorf("should not create existing repo, got: %v", mock.createdRepos)
	}
}

func TestProjectReconciler_AlreadyReady(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "ready-proj", Namespace: "aot",
			Finalizers: []string{projectFinalizerName}},
		Spec: aotv1alpha1.ProjectSpec{DisplayName: "Ready"},
		Status: aotv1alpha1.ProjectStatus{
			ConfigRepoReady: true,
			ConfigRepoURL:   "ssh://soft-serve.test/ready-proj.git",
		},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(project).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()
	mock := newMockRepoManager()

	r := &ProjectReconciler{Client: k8s, SoftServe: mock}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "ready-proj", Namespace: "aot"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Should NOT have done anything
	if len(mock.createdRepos) != 0 {
		t.Errorf("should not create repo for ready project")
	}
	if len(mock.scaffolded) != 0 {
		t.Errorf("should not scaffold ready project")
	}
}

func TestProjectReconciler_CreateFails(t *testing.T) {
	project := &aotv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fail-proj", Namespace: "aot"},
		Spec:       aotv1alpha1.ProjectSpec{DisplayName: "Fail"},
	}

	k8s := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(project).
		WithStatusSubresource(&aotv1alpha1.Project{}).
		Build()
	mock := newMockRepoManager()
	mock.failCreate = true

	r := &ProjectReconciler{Client: k8s, SoftServe: mock}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "fail-proj", Namespace: "aot"}}
	// First reconcile: adds finalizer and returns early (no error yet).
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}
	// Second reconcile: attempts to create repo — should fail.
	if _, err := r.Reconcile(context.Background(), req); err == nil {
		t.Fatal("expected error on create failure")
	}
}

func TestProjectReconciler_NotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	mock := newMockRepoManager()

	r := &ProjectReconciler{Client: k8s, SoftServe: mock}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "aot"},
	})
	if err != nil {
		t.Fatalf("expected no error for missing project, got: %v", err)
	}
}
