// Package controller implements the Kubernetes controller for AgentRun CRDs.
// The controller acts as a thin bridge between K8s CRDs and Temporal workflows:
// - New CRD → start Temporal workflow, annotate with workflow ID
// - Existing CRD → query Temporal workflow state, sync to CRD status
// - Deleted CRD → cancel Temporal workflow
package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	temporalclient "go.temporal.io/sdk/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/internal/eventbus"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

const (
	// annotationWorkflowID stores the Temporal workflow ID on the AgentRun CRD.
	annotationWorkflowID = "aot.uncworks.io/workflow-id"

	// finalizerName ensures the controller cancels the Temporal workflow before the CRD is deleted.
	finalizerName = "aot.uncworks.io/workflow-cleanup"

	// reconcileInterval is the requeue interval for syncing workflow state.
	reconcileInterval = 30 * time.Second

	// Labels and annotations for orchestration
	labelSpecRunID   = "aot.uncworks.io/spec-run-id"
	labelRunRole     = "aot.uncworks.io/run-role"
	annotationParent = "aot.uncworks.io/parent-run"
)

// AgentRunReconciler reconciles AgentRun objects by bridging to Temporal workflows.
type AgentRunReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	TemporalClient temporalclient.Client
	TaskQueue      string
	LiteLLMBaseURL string
	EventBus       eventbus.EventBus
	eventBusWarned bool
}

// +kubebuilder:rbac:groups=aot.uncworks.io,resources=agentruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aot.uncworks.io,resources=agentruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;delete

// Reconcile handles changes to AgentRun resources.
func (r *AgentRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agentRun aotv1alpha1.AgentRun
	if err := r.Get(ctx, req.NamespacedName, &agentRun); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion with finalizer
	if !agentRun.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&agentRun, finalizerName) {
			if err := r.cancelWorkflow(ctx, &agentRun); err != nil {
				logger.Error(err, "Failed to cancel workflow during deletion")
			}
			controllerutil.RemoveFinalizer(&agentRun, finalizerName)
			if err := r.Update(ctx, &agentRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is set
	if !controllerutil.ContainsFinalizer(&agentRun, finalizerName) {
		controllerutil.AddFinalizer(&agentRun, finalizerName)
		if err := r.Update(ctx, &agentRun); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check for workflow ID annotation
	workflowID := agentRun.Annotations[annotationWorkflowID]

	if workflowID == "" {
		// New CRD — start Temporal workflow
		return r.startWorkflow(ctx, &agentRun)
	}

	// Existing CRD — sync workflow state to CRD status
	logger.V(1).Info("Syncing workflow state", "workflowID", workflowID)
	return r.syncWorkflowState(ctx, &agentRun, workflowID)
}

// startWorkflow creates a new Temporal workflow for the AgentRun and annotates the CRD.
func (r *AgentRunReconciler) startWorkflow(ctx context.Context, agentRun *aotv1alpha1.AgentRun) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Handle unsupported backends
	switch agentRun.Spec.Backend {
	case aotv1alpha1.BackendKubeVirt:
		return r.handleNotImplemented(ctx, agentRun, "KubeVirt")
	case aotv1alpha1.BackendExternal:
		return r.handleNotImplemented(ctx, agentRun, "External")
	}

	var repos []aottemporal.Repository
	for _, repo := range agentRun.Spec.Repos {
		repos = append(repos, aottemporal.Repository{
			URL:    repo.URL,
			Branch: repo.Branch,
			Path:   repo.Path,
		})
	}
	// Map orchestration tasks from CRD to workflow input
	var orchTasks []aottemporal.OrchestrationTask
	if agentRun.Spec.Orchestration != nil {
		for _, t := range agentRun.Spec.Orchestration.Tasks {
			orchTasks = append(orchTasks, aottemporal.OrchestrationTask{
				Name:     t.Name,
				Prompt:   t.Prompt,
				RepoURLs: t.RepoURLs,
			})
		}
	}

	workflowInput := aottemporal.WorkflowInput{
		AgentRunName:      agentRun.Name,
		Namespace:         agentRun.Namespace,
		Repos:             repos,
		Prompt:            agentRun.Spec.Prompt,
		DevboxConfig:      agentRun.Spec.DevboxConfig,
		TTLSeconds:        agentRun.Spec.TTLSeconds,
		Image:             agentRun.Spec.Image,
		EnvVars:           agentRun.Spec.EnvVars,
		ModelTier:         agentRun.Spec.ModelTier,
		LiteLLMBaseURL:    r.LiteLLMBaseURL,
		SpecContent:       agentRun.Spec.SpecContent,
		WorkspaceName:     agentRun.Spec.WorkspaceName,
		OrchestrationMode: aottemporal.OrchestrationMode(agentRun.Spec.OrchestrationMode),
		Orchestration:     orchTasks,
		ParentRunID:       agentRun.Spec.ParentRunID,
		SpecRunID:         agentRun.Spec.SpecRunID,
	}

	// Set orchestration labels
	if agentRun.Labels == nil {
		agentRun.Labels = make(map[string]string)
	}
	if agentRun.Annotations == nil {
		agentRun.Annotations = make(map[string]string)
	}

	orchMode := agentRun.Spec.OrchestrationMode
	if orchMode == aotv1alpha1.OrchestrationModeAuto || orchMode == aotv1alpha1.OrchestrationModeManual {
		// Senior run
		agentRun.Labels[labelSpecRunID] = agentRun.Name
		agentRun.Labels[labelRunRole] = "senior"
	} else if agentRun.Spec.ParentRunID != "" {
		// Junior run
		specRunID := agentRun.Spec.SpecRunID
		if specRunID == "" {
			specRunID = agentRun.Spec.ParentRunID
		}
		agentRun.Labels[labelSpecRunID] = specRunID
		agentRun.Labels[labelRunRole] = "junior"
		agentRun.Annotations[annotationParent] = agentRun.Spec.ParentRunID
	}

	taskQueue := r.TaskQueue
	if taskQueue == "" {
		taskQueue = aottemporal.TaskQueue
	}

	workflowID := fmt.Sprintf("agentrun-%s", agentRun.Name)
	run, err := r.TemporalClient.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}, aottemporal.AgentRunWorkflow, workflowInput)
	if err != nil {
		logger.Error(err, "Failed to start Temporal workflow, will retry")
		agentRun.Status.Message = fmt.Sprintf("Retrying workflow start: %v", err)
		if updateErr := r.Status().Update(ctx, agentRun); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after workflow start failure")
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	logger.Info("Started Temporal workflow", "workflowID", run.GetID(), "runID", run.GetRunID())

	// Annotate CRD with workflow ID
	if agentRun.Annotations == nil {
		agentRun.Annotations = make(map[string]string)
	}
	agentRun.Annotations[annotationWorkflowID] = run.GetID()
	if err := r.Update(ctx, agentRun); err != nil {
		return ctrl.Result{}, err
	}

	// Re-fetch to get the latest resourceVersion after the annotation update
	if err := r.Get(ctx, client.ObjectKeyFromObject(agentRun), agentRun); err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	now := metav1.Now()
	agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	agentRun.Status.PodName = fmt.Sprintf("agentrun-%s", agentRun.Name)
	agentRun.Status.StartedAt = &now
	agentRun.Status.Message = "Temporal workflow started"
	if err := r.Status().Update(ctx, agentRun); err != nil {
		return ctrl.Result{}, err
	}
	r.emitPhaseEvent(agentRun, apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED)
	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// syncWorkflowState queries the Temporal workflow and syncs state to the CRD.
func (r *AgentRunReconciler) syncWorkflowState(ctx context.Context, agentRun *aotv1alpha1.AgentRun, workflowID string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Query workflow state
	resp, err := r.TemporalClient.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
	if err != nil {
		logger.V(1).Info("Failed to query workflow, may have completed", "error", err)
		// Workflow may have completed and been archived — check execution
		desc, descErr := r.TemporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
		if descErr != nil {
			agentRun.Status.Message = fmt.Sprintf("Temporal unreachable: %v", descErr)
			if updateErr := r.Status().Update(ctx, agentRun); updateErr != nil {
				logger.Error(updateErr, "Failed to update status with Temporal error")
			}
			return ctrl.Result{RequeueAfter: reconcileInterval}, nil
		}
		// Map Temporal execution status to CRD phase
		return r.syncFromDescription(ctx, agentRun, desc)
	}

	var state aottemporal.WorkflowState
	if err := resp.Get(&state); err != nil {
		logger.Error(err, "Failed to decode workflow state")
		return ctrl.Result{RequeueAfter: reconcileInterval}, nil
	}

	// Map workflow state to CRD status
	updated := false
	newPhase := mapPhase(state.Phase)
	if agentRun.Status.Phase != newPhase {
		agentRun.Status.Phase = newPhase
		updated = true
	}
	if agentRun.Status.Message != state.Message {
		agentRun.Status.Message = state.Message
		updated = true
	}
	if state.PodName != "" && agentRun.Status.PodName != state.PodName {
		agentRun.Status.PodName = state.PodName
		updated = true
	}
	if state.DeploymentName != "" && agentRun.Status.DeploymentName != state.DeploymentName {
		agentRun.Status.DeploymentName = state.DeploymentName
		updated = true
	}

	// Set CompletedAt for terminal states
	if isTerminal(newPhase) && agentRun.Status.CompletedAt == nil {
		now := metav1.Now()
		agentRun.Status.CompletedAt = &now
		updated = true
	}

	if updated {
		if err := r.Status().Update(ctx, agentRun); err != nil {
			return ctrl.Result{}, err
		}
		eventType := apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED
		if isTerminal(newPhase) {
			eventType = apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED
		}
		r.emitPhaseEvent(agentRun, eventType)
	}

	// Don't requeue for terminal states
	if isTerminal(newPhase) {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// syncFromDescription syncs CRD status from a Temporal workflow description
// when the workflow can no longer be queried (completed/terminated).
func (r *AgentRunReconciler) syncFromDescription(ctx context.Context, agentRun *aotv1alpha1.AgentRun, desc *workflowservice.DescribeWorkflowExecutionResponse) (ctrl.Result, error) {
	status := desc.GetWorkflowExecutionInfo().GetStatus()

	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
		agentRun.Status.Message = "Workflow completed"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
		agentRun.Status.Message = "Workflow failed"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseCancelled
		agentRun.Status.Message = "Workflow cancelled"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
		agentRun.Status.Message = "Workflow terminated"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
		agentRun.Status.Message = "Workflow timed out"
	default:
		return ctrl.Result{RequeueAfter: reconcileInterval}, nil
	}

	if agentRun.Status.CompletedAt == nil {
		// Use Temporal's CloseTime if available, otherwise fall back to now
		if closeTime := desc.GetWorkflowExecutionInfo().GetCloseTime(); closeTime != nil && closeTime.IsValid() {
			t := metav1.NewTime(closeTime.AsTime())
			agentRun.Status.CompletedAt = &t
		} else {
			now := metav1.Now()
			agentRun.Status.CompletedAt = &now
		}
	}

	if err := r.Status().Update(ctx, agentRun); err != nil {
		return ctrl.Result{}, err
	}
	r.emitPhaseEvent(agentRun, apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED)
	return ctrl.Result{}, nil
}

// cancelWorkflow cancels the Temporal workflow associated with the AgentRun.
func (r *AgentRunReconciler) cancelWorkflow(ctx context.Context, agentRun *aotv1alpha1.AgentRun) error {
	logger := log.FromContext(ctx)

	workflowID := agentRun.Annotations[annotationWorkflowID]
	if workflowID == "" {
		return nil
	}

	logger.Info("Cancelling Temporal workflow for deleted AgentRun", "workflowID", workflowID)
	if err := r.TemporalClient.CancelWorkflow(ctx, workflowID, ""); err != nil {
		logger.Error(err, "Failed to cancel workflow, may already be completed")
	}
	return nil
}

func (r *AgentRunReconciler) handleNotImplemented(ctx context.Context, agentRun *aotv1alpha1.AgentRun, backend string) (ctrl.Result, error) {
	agentRun.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
	agentRun.Status.Message = fmt.Sprintf("%s backend is not yet implemented", backend)
	if err := r.Status().Update(ctx, agentRun); err != nil {
		return ctrl.Result{}, err
	}
	r.emitPhaseEvent(agentRun, apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED)
	return ctrl.Result{}, nil
}

// mapPhase converts workflow state phase strings to CRD phase constants.
func mapPhase(workflowPhase string) aotv1alpha1.AgentRunPhase {
	switch workflowPhase {
	case "Pending", "Creating", "Hydrating":
		return aotv1alpha1.AgentRunPhasePending
	case "Running":
		return aotv1alpha1.AgentRunPhaseRunning
	case "WaitingForInput":
		return aotv1alpha1.AgentRunPhaseWaitingForInput
	case "Succeeded":
		return aotv1alpha1.AgentRunPhaseSucceeded
	case "Failed":
		return aotv1alpha1.AgentRunPhaseFailed
	case "Cancelled", "Cancelling":
		return aotv1alpha1.AgentRunPhaseCancelled
	default:
		return aotv1alpha1.AgentRunPhasePending
	}
}

func isTerminal(phase aotv1alpha1.AgentRunPhase) bool {
	return phase == aotv1alpha1.AgentRunPhaseSucceeded ||
		phase == aotv1alpha1.AgentRunPhaseFailed ||
		phase == aotv1alpha1.AgentRunPhaseCancelled
}

// emitPhaseEvent publishes a phase-change event to the event bus.
func (r *AgentRunReconciler) emitPhaseEvent(agentRun *aotv1alpha1.AgentRun, eventType apiv1.AgentRunEventType) {
	if r.EventBus == nil {
		if !r.eventBusWarned {
			r.eventBusWarned = true
			ctrl.Log.Info("WARNING: EventBus is nil, phase events will not be emitted to WatchAgentRun subscribers")
		}
		return
	}
	r.EventBus.Publish(agentRun.Name, &apiv1.AgentRunEvent{
		AgentRunId: agentRun.Name,
		Type:       eventType,
		Payload:    string(agentRun.Status.Phase),
	})
}

// TODO(persistent-workspace): Archive cleanup — delete Deployment + PVC for runs with
// completedAt older than 7 days. This should be implemented as either:
//   - A separate CronJob that lists AgentRuns with completedAt > 7d and calls ArchiveAndCleanup, or
//   - An additional reconciliation pass in this controller that checks retention expiry.
// The ArchiveAndCleanup Temporal activity already exists in internal/temporal/activities.go.
// For now, completed runs retain their Deployment (replicas=0) and PVC indefinitely.

// SetupWithManager sets up the controller with the Manager.
func (r *AgentRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aotv1alpha1.AgentRun{}).
		Complete(r)
}
