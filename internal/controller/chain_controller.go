package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

const (
	chainReconcileInterval = 10 * time.Second
	chainFinalizerName     = "aot.uncworks.io/chainrun-cleanup"
)

// ChainRunReconciler watches ChainRun CRDs and manages the DAG execution.
type ChainRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ChainRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cr aotv1alpha1.ChainRun
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion: remove child AgentRuns that have not yet completed.
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, chainFinalizerName) {
			for _, s := range cr.Status.Steps {
				if s.RunID == "" {
					continue
				}
				var run aotv1alpha1.AgentRun
				if err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: s.RunID}, &run); err == nil {
					if err := r.Delete(ctx, &run); err != nil {
						logger.Error(err, "Failed to delete child AgentRun on ChainRun deletion", "run", s.RunID)
					}
				}
			}
			controllerutil.RemoveFinalizer(&cr, chainFinalizerName)
			if err := r.Update(ctx, &cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is set.
	if !controllerutil.ContainsFinalizer(&cr, chainFinalizerName) {
		controllerutil.AddFinalizer(&cr, chainFinalizerName)
		if err := r.Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Skip terminal states
	if cr.Status.Phase == aotv1alpha1.ChainRunPhaseSucceeded || cr.Status.Phase == aotv1alpha1.ChainRunPhaseFailed || cr.Status.Phase == aotv1alpha1.ChainRunPhaseCancelled {
		return ctrl.Result{}, nil
	}

	// Load the Chain definition
	var chain aotv1alpha1.Chain
	if err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Spec.ChainRef}, &chain); err != nil {
		cr.Status.Phase = aotv1alpha1.ChainRunPhaseFailed
		cr.Status.Message = fmt.Sprintf("chain %q not found: %v", cr.Spec.ChainRef, err)
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Validate chain DAG before executing (catches cycles, undefined deps, etc.)
	if err := aotv1alpha1.ValidateChainDAG(chain.Spec.Steps); err != nil {
		cr.Status.Phase = aotv1alpha1.ChainRunPhaseFailed
		cr.Status.Message = fmt.Sprintf("invalid chain DAG: %v", err)
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Initialize step statuses on first reconcile
	if len(cr.Status.Steps) == 0 {
		now := metav1.Now()
		cr.Status.Phase = aotv1alpha1.ChainRunPhaseRunning
		cr.Status.StartedAt = &now
		for _, step := range chain.Spec.Steps {
			cr.Status.Steps = append(cr.Status.Steps, aotv1alpha1.ChainRunStepStatus{
				Name:  step.Name,
				Phase: aotv1alpha1.ChainRunStepPhasePending,
			})
		}
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Build step status map
	stepStatus := make(map[string]*aotv1alpha1.ChainRunStepStatus)
	for i := range cr.Status.Steps {
		stepStatus[cr.Status.Steps[i].Name] = &cr.Status.Steps[i]
	}

	// Build step definition map
	stepDef := make(map[string]*aotv1alpha1.ChainStep)
	for i := range chain.Spec.Steps {
		stepDef[chain.Spec.Steps[i].Name] = &chain.Spec.Steps[i]
	}

	// Check running steps — sync their status from AgentRun CRDs
	updated := false
	for i := range cr.Status.Steps {
		s := &cr.Status.Steps[i]
		if s.Phase != aotv1alpha1.ChainRunStepPhaseRunning || s.RunID == "" {
			continue
		}
		var run aotv1alpha1.AgentRun
		if err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: s.RunID}, &run); err != nil {
			continue
		}
		switch run.Status.Phase {
		case aotv1alpha1.AgentRunPhaseSucceeded:
			s.Phase = aotv1alpha1.ChainRunStepPhaseSucceeded
			now := metav1.Now()
			s.CompletedAt = &now
			s.Message = "completed"
			updated = true
		case aotv1alpha1.AgentRunPhaseFailed:
			s.Phase = aotv1alpha1.ChainRunStepPhaseFailed
			now := metav1.Now()
			s.CompletedAt = &now
			s.Message = run.Status.Message
			updated = true
		case aotv1alpha1.AgentRunPhaseCancelled:
			s.Phase = aotv1alpha1.ChainRunStepPhaseFailed
			now := metav1.Now()
			s.CompletedAt = &now
			s.Message = "cancelled"
			updated = true
		}
	}

	// Find pending steps whose dependencies are all satisfied
	for i := range cr.Status.Steps {
		s := &cr.Status.Steps[i]
		if s.Phase != aotv1alpha1.ChainRunStepPhasePending {
			continue
		}
		def := stepDef[s.Name]
		if def == nil {
			continue
		}

		// Check all dependencies are succeeded
		allDepsOK := true
		for _, dep := range def.DependsOn {
			depStatus := stepStatus[dep]
			if depStatus == nil || depStatus.Phase != aotv1alpha1.ChainRunStepPhaseSucceeded {
				allDepsOK = false
				break
			}
		}
		if !allDepsOK {
			continue
		}

		// Look up the template
		tmpl := &aotv1alpha1.RunTemplate{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: def.TemplateRef}, tmpl); err != nil {
			s.Phase = aotv1alpha1.ChainRunStepPhaseFailed
			s.Message = fmt.Sprintf("template %q not found", def.TemplateRef)
			updated = true
			continue
		}

		// Create AgentRun for this step
		run := &aotv1alpha1.AgentRun{}
		run.GenerateName = "ar-"
		run.Namespace = cr.Namespace
		prompt := tmpl.Spec.Prompt

		// Inject context from dependency output
		if def.ContextFrom != "" {
			depStatus := stepStatus[def.ContextFrom]
			if depStatus != nil && depStatus.RunID != "" {
				var depRun aotv1alpha1.AgentRun
				if err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: depStatus.RunID}, &depRun); err == nil {
					prompt = fmt.Sprintf("Context from previous step '%s': %s\n\n%s",
						def.ContextFrom, depRun.Status.Message, prompt)
				}
			}
		}

		// Use branch from dependency if specified
		branch := ""
		if def.BranchFrom != "" {
			if branchDep := stepStatus[def.BranchFrom]; branchDep != nil {
				branch = fmt.Sprintf("aot/%s", branchDep.RunID)
			}
		}

		// Deep-copy the repos slice so we don't mutate the template object that
		// may be shared across multiple chain steps referencing the same template.
		repos := make([]aotv1alpha1.Repository, len(tmpl.Spec.Repos))
		copy(repos, tmpl.Spec.Repos)
		if branch != "" && len(repos) > 0 {
			repos[0].Branch = branch
		}

		run.Spec = aotv1alpha1.AgentRunSpec{
			Backend:            aotv1alpha1.BackendPod,
			Repos:              repos,
			Prompt:             prompt,
			ModelTier:          tmpl.Spec.ModelTier,
			ManageModelTier:    tmpl.Spec.ManageModelTier,
			ImplementModelTier: tmpl.Spec.ImplementModelTier,
			OrchestrationMode:  tmpl.Spec.OrchestrationMode,
			TTLSeconds:         tmpl.Spec.TTLSeconds,
			AutoPush:           tmpl.Spec.AutoPush,
			AutoPR:             tmpl.Spec.AutoPR,
			PRBaseBranch:       tmpl.Spec.PRBaseBranch,
			ProjectRef:         tmpl.Spec.ProjectRef,
		}
		run.Labels = map[string]string{
			"aot.uncworks.io/chain-run": cr.Name,
			"aot.uncworks.io/chain":     cr.Spec.ChainRef,
			"aot.uncworks.io/step":      s.Name,
		}

		// Set owner reference so the AgentRun is garbage collected when this ChainRun is deleted.
		if err := controllerutil.SetControllerReference(&cr, run, r.Scheme); err != nil {
			logger.Error(err, "Failed to set controller reference on chain step AgentRun", "step", s.Name)
		}
		if err := r.Create(ctx, run); err != nil {
			s.Phase = aotv1alpha1.ChainRunStepPhaseFailed
			s.Message = fmt.Sprintf("create run: %v", err)
			updated = true
			continue
		}

		now := metav1.Now()
		s.Phase = aotv1alpha1.ChainRunStepPhaseRunning
		s.RunID = run.Name
		s.StartedAt = &now
		s.Message = "started"
		updated = true
		logger.Info("Started chain step", "chainRun", cr.Name, "step", s.Name, "run", run.Name)
	}

	// Propagate failures: skip pending steps whose dependencies have failed or been skipped.
	// Iterate until stable (transitive closure).
	for changed := true; changed; {
		changed = false
		for i := range cr.Status.Steps {
			s := &cr.Status.Steps[i]
			if s.Phase != aotv1alpha1.ChainRunStepPhasePending {
				continue
			}
			def := stepDef[s.Name]
			if def == nil {
				continue
			}
			for _, dep := range def.DependsOn {
				depStatus := stepStatus[dep]
				if depStatus != nil && (depStatus.Phase == aotv1alpha1.ChainRunStepPhaseFailed || depStatus.Phase == aotv1alpha1.ChainRunStepPhaseSkipped) {
					s.Phase = aotv1alpha1.ChainRunStepPhaseSkipped
					s.Message = fmt.Sprintf("skipped because dependency %q %s", dep, depStatus.Phase)
					updated = true
					changed = true
					break
				}
			}
		}
	}

	// Check overall completion: done when no step is pending or running.
	allDone := true
	anyFailed := false
	for _, s := range cr.Status.Steps {
		if s.Phase == aotv1alpha1.ChainRunStepPhasePending || s.Phase == aotv1alpha1.ChainRunStepPhaseRunning {
			allDone = false
		}
		if s.Phase == aotv1alpha1.ChainRunStepPhaseFailed {
			anyFailed = true
		}
	}

	if allDone {
		now := metav1.Now()
		cr.Status.CompletedAt = &now
		if anyFailed {
			cr.Status.Phase = aotv1alpha1.ChainRunPhaseFailed
			cr.Status.Message = "one or more steps failed"
		} else {
			cr.Status.Phase = aotv1alpha1.ChainRunPhaseSucceeded
			cr.Status.Message = "all steps completed"
		}
		updated = true
	}

	if updated {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Requeue if still running
	if cr.Status.Phase == aotv1alpha1.ChainRunPhaseRunning {
		return ctrl.Result{RequeueAfter: chainReconcileInterval}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ChainRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aotv1alpha1.ChainRun{}).
		Complete(r)
}
