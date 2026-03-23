package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// ScheduleReconciler watches Schedule CRDs and creates runs/chain-runs on cron tick.
type ScheduleReconciler struct {
	client.Client
}

func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sched aotv1alpha1.Schedule
	if err := r.Get(ctx, req.NamespacedName, &sched); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if sched.Spec.Suspend {
		return ctrl.Result{}, nil
	}

	// Parse cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(sched.Spec.Cron)
	if err != nil {
		logger.Error(err, "invalid cron expression", "cron", sched.Spec.Cron)
		return ctrl.Result{}, nil
	}

	now := time.Now()
	nextTime := schedule.Next(now)

	// Update next schedule time in status
	nextMeta := metav1.NewTime(nextTime)
	sched.Status.NextScheduleTime = &nextMeta

	// Check if we should fire
	var lastScheduled time.Time
	if sched.Status.LastScheduledTime != nil {
		lastScheduled = sched.Status.LastScheduledTime.Time
	} else {
		lastScheduled = sched.CreationTimestamp.Time
	}

	// Find all missed schedules between last fire and now
	missedRun := schedule.Next(lastScheduled)
	if missedRun.After(now) {
		// No missed runs — just update next time and requeue
		if err := r.Status().Update(ctx, &sched); err != nil {
			return ctrl.Result{}, err
		}
		requeueAfter := time.Until(nextTime) + time.Second
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Concurrency policy check
	if sched.Spec.ConcurrencyPolicy == "Forbid" && len(sched.Status.Active) > 0 {
		logger.Info("Skipping schedule (Forbid policy, active run exists)", "schedule", sched.Name)
		requeueAfter := time.Until(nextTime) + time.Second
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Fire the schedule
	logger.Info("Firing schedule", "schedule", sched.Name, "cron", sched.Spec.Cron)

	var runID string
	if sched.Spec.ChainRef != "" {
		// Create a ChainRun
		cr := &aotv1alpha1.ChainRun{}
		cr.GenerateName = "cr-"
		cr.Namespace = sched.Namespace
		cr.Spec = aotv1alpha1.ChainRunSpec{
			ChainRef:    sched.Spec.ChainRef,
			TriggeredBy: fmt.Sprintf("schedule:%s", sched.Name),
		}
		if err := r.Create(ctx, cr); err != nil {
			return ctrl.Result{}, fmt.Errorf("create chain run: %w", err)
		}
		runID = cr.Name
		logger.Info("Created chain run from schedule", "chainRun", cr.Name, "chain", sched.Spec.ChainRef)
	} else if sched.Spec.TemplateRef != "" {
		// Look up the template and create an AgentRun
		tmpl := &aotv1alpha1.RunTemplate{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: sched.Namespace, Name: sched.Spec.TemplateRef}, tmpl); err != nil {
			return ctrl.Result{}, fmt.Errorf("get template %s: %w", sched.Spec.TemplateRef, err)
		}
		run := &aotv1alpha1.AgentRun{}
		run.GenerateName = "ar-"
		run.Namespace = sched.Namespace
		run.Spec = aotv1alpha1.AgentRunSpec{
			Backend:            aotv1alpha1.BackendPod,
			Repos:              tmpl.Spec.Repos,
			Prompt:             tmpl.Spec.Prompt,
			ModelTier:          tmpl.Spec.ModelTier,
			ManageModelTier:    tmpl.Spec.ManageModelTier,
			ImplementModelTier: tmpl.Spec.ImplementModelTier,
			OrchestrationMode:  tmpl.Spec.OrchestrationMode,
			TTLSeconds:         tmpl.Spec.TTLSeconds,
			AutoPush:           tmpl.Spec.AutoPush,
			AutoPR:             tmpl.Spec.AutoPR,
			PRBaseBranch:       tmpl.Spec.PRBaseBranch,
			ProjectRef:         tmpl.Spec.ProjectRef,
			SpecRef:            tmpl.Spec.SpecRef,
		}
		run.Labels = map[string]string{
			"aot.uncworks.io/template": tmpl.Name,
			"aot.uncworks.io/schedule": sched.Name,
		}
		if err := r.Create(ctx, run); err != nil {
			return ctrl.Result{}, fmt.Errorf("create agent run: %w", err)
		}
		runID = run.Name
		logger.Info("Created agent run from schedule", "run", run.Name, "template", sched.Spec.TemplateRef)
	}

	// Update status
	nowMeta := metav1.Now()
	sched.Status.LastScheduledTime = &nowMeta
	sched.Status.LastRunID = runID
	sched.Status.LastResult = "running"
	sched.Status.Active = append(sched.Status.Active, runID)
	if err := r.Status().Update(ctx, &sched); err != nil {
		return ctrl.Result{}, err
	}

	requeueAfter := time.Until(nextTime) + time.Second
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aotv1alpha1.Schedule{}).
		Complete(r)
}
