package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/softserve"
)

// ResolveProjectDefaults fills empty fields in the AgentRun spec from the
// referenced Project's defaults. Returns the project config repo URL (if any)
// so the hydration init container can clone it.
func ResolveProjectDefaults(
	ctx context.Context,
	k8s client.Client,
	ss softserve.RepoManager,
	agentRun *aotv1alpha1.AgentRun,
	namespace string,
) (configRepoURL string, err error) {
	if agentRun.Spec.ProjectRef == "" {
		return "", nil
	}

	var project aotv1alpha1.Project
	if err := k8s.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      agentRun.Spec.ProjectRef,
	}, &project); err != nil {
		return "", fmt.Errorf("get project %q: %w", agentRun.Spec.ProjectRef, err)
	}

	// Inherit repos if not specified on the run
	if len(agentRun.Spec.Repos) == 0 && len(project.Spec.Repos) > 0 {
		agentRun.Spec.Repos = project.Spec.Repos
	}

	// Inherit defaults
	if d := project.Spec.Defaults; d != nil {
		if agentRun.Spec.ModelTier == "" && d.ModelTier != "" {
			agentRun.Spec.ModelTier = d.ModelTier
		}
		if agentRun.Spec.ManageModelTier == "" && d.ManageModelTier != "" {
			agentRun.Spec.ManageModelTier = d.ManageModelTier
		}
		if agentRun.Spec.ImplementModelTier == "" && d.ImplementModelTier != "" {
			agentRun.Spec.ImplementModelTier = d.ImplementModelTier
		}
		if agentRun.Spec.TTLSeconds == 0 && d.TTLSeconds > 0 {
			agentRun.Spec.TTLSeconds = d.TTLSeconds
		}
		if agentRun.Spec.OrchestrationMode == "" && d.OrchestrationMode != "" {
			agentRun.Spec.OrchestrationMode = aotv1alpha1.OrchestrationMode(d.OrchestrationMode)
		}
		if !agentRun.Spec.AutoPush && d.AutoPush {
			agentRun.Spec.AutoPush = true
		}
		if !agentRun.Spec.AutoPR && d.AutoPR {
			agentRun.Spec.AutoPR = true
		}
		if agentRun.Spec.PRBaseBranch == "" && d.PRBaseBranch != "" {
			agentRun.Spec.PRBaseBranch = d.PRBaseBranch
		}
	}

	// Resolve specRef: fetch spec content from the project's config repo
	if agentRun.Spec.SpecRef != "" && agentRun.Spec.SpecContent == "" && ss != nil {
		specPath := fmt.Sprintf("openspec/specs/%s/spec.md", agentRun.Spec.SpecRef)
		content, readErr := ss.ReadFile(agentRun.Spec.ProjectRef, specPath)
		if readErr != nil {
			return "", fmt.Errorf("read spec %q from project %q: %w", agentRun.Spec.SpecRef, agentRun.Spec.ProjectRef, readErr)
		}
		agentRun.Spec.SpecContent = content
		agentRun.Spec.SpecSource = fmt.Sprintf("project:%s/spec:%s", agentRun.Spec.ProjectRef, agentRun.Spec.SpecRef)
	}

	// Set project label for filtering
	if agentRun.Spec.Project == "" {
		agentRun.Spec.Project = project.Spec.DisplayName
		if agentRun.Spec.Project == "" {
			agentRun.Spec.Project = project.Name
		}
	}

	return project.Status.ConfigRepoURL, nil
}
