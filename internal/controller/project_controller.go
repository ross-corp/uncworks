package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

// ProjectReconciler reconciles Project objects.
type ProjectReconciler struct {
	client.Client
	SoftServe softserve.RepoManager
}

// Reconcile handles changes to Project resources, ensuring a soft-serve config repo
// is created and ready for the project.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var project aotv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion: clean up soft-serve repo
	if !project.DeletionTimestamp.IsZero() {
		if containsFinalizer(project.Finalizers, projectFinalizerName) {
			logger.Info("Deleting soft-serve repo", "project", project.Name)
			if err := r.SoftServe.DeleteRepo(project.Name); err != nil {
				logger.Error(err, "Failed to delete soft-serve repo", "project", project.Name)
				// Don't block deletion on soft-serve errors
			}
			project.Finalizers = removeFinalizer(project.Finalizers, projectFinalizerName)
			if err := r.Update(ctx, &project); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer. Return immediately after persisting so the next reconcile
	// starts from a committed state and does not attempt repo creation before the
	// finalizer is durably recorded.
	if !containsFinalizer(project.Finalizers, projectFinalizerName) {
		project.Finalizers = append(project.Finalizers, projectFinalizerName)
		if err := r.Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create soft-serve repo if not ready (skip when soft-serve is not configured)
	if !project.Status.ConfigRepoReady && r.SoftServe != nil {
		logger.Info("Creating soft-serve repo", "project", project.Name)

		exists, err := r.SoftServe.RepoExists(project.Name)
		if err != nil {
			logger.Error(err, "Failed to check repo existence")
			meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
				Type:               "ConfigRepoReady",
				Status:             metav1.ConditionFalse,
				Reason:             "RepoExistenceCheckFailed",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
			if statusErr := r.Status().Update(ctx, &project); statusErr != nil {
				logger.Error(statusErr, "Failed to update status after repo existence check failure")
			}
			return ctrl.Result{}, err
		}

		if !exists {
			if err := r.SoftServe.CreateRepo(project.Name); err != nil {
				logger.Error(err, "Failed to create soft-serve repo")
				meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
					Type:               "ConfigRepoReady",
					Status:             metav1.ConditionFalse,
					Reason:             "CreateFailed",
					Message:            err.Error(),
					LastTransitionTime: metav1.Now(),
				})
				if statusErr := r.Status().Update(ctx, &project); statusErr != nil {
					logger.Error(statusErr, "Failed to update status")
				}
				return ctrl.Result{}, err
			}

			// Scaffold initial files
			var packages []string
			if project.Spec.Devbox != nil {
				packages = project.Spec.Devbox.Packages
			}
			if err := r.SoftServe.ScaffoldAndPush(softserve.ScaffoldProject{
				Name:     project.Name,
				Packages: packages,
			}); err != nil {
				logger.Error(err, "Failed to scaffold project repo")
				meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
					Type:               "ConfigRepoReady",
					Status:             metav1.ConditionFalse,
					Reason:             "ScaffoldFailed",
					Message:            err.Error(),
					LastTransitionTime: metav1.Now(),
				})
				if statusErr := r.Status().Update(ctx, &project); statusErr != nil {
					logger.Error(statusErr, "Failed to update status after scaffold failure")
				}
				return ctrl.Result{}, err
			}
		}

		// Update status
		project.Status.ConfigRepoReady = true
		project.Status.ConfigRepoURL = r.SoftServe.CloneURL(project.Name)
		meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
			Type:               "ConfigRepoReady",
			Status:             metav1.ConditionTrue,
			Reason:             "RepoCreated",
			Message:            fmt.Sprintf("Config repo created at %s", project.Status.ConfigRepoURL),
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Project config repo ready", "project", project.Name, "url", project.Status.ConfigRepoURL)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers ProjectReconciler with the controller manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aotv1alpha1.Project{}).
		Complete(r)
}

const projectFinalizerName = "aot.uncworks.io/project-finalizer"

func containsFinalizer(finalizers []string, name string) bool {
	for _, f := range finalizers {
		if f == name {
			return true
		}
	}
	return false
}

func removeFinalizer(finalizers []string, name string) []string {
	var result []string
	for _, f := range finalizers {
		if f != name {
			result = append(result, f)
		}
	}
	return result
}
