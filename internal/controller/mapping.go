package controller

import (
	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// BuildWorkflowInput maps a CRD AgentRun to a Temporal WorkflowInput.
// Extracted from startWorkflow for testability.
// The liteLLMBaseURL parameter corresponds to the reconciler's LiteLLMBaseURL field.
func BuildWorkflowInput(agentRun *aotv1alpha1.AgentRun, liteLLMBaseURL string) aottemporal.WorkflowInput {
	var repos []aottemporal.Repository
	for _, repo := range agentRun.Spec.Repos {
		repos = append(repos, aottemporal.Repository{
			URL:    repo.URL,
			Branch: repo.Branch,
			Path:   repo.Path,
		})
	}

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

	input := aottemporal.WorkflowInput{
		AgentRunName:      agentRun.Name,
		Namespace:         agentRun.Namespace,
		Repos:             repos,
		Prompt:            agentRun.Spec.Prompt,
		DevboxConfig:      agentRun.Spec.DevboxConfig,
		TTLSeconds:        agentRun.Spec.TTLSeconds,
		Image:             agentRun.Spec.Image,
		EnvVars:           agentRun.Spec.EnvVars,
		ModelTier:         agentRun.Spec.ModelTier,
		LiteLLMBaseURL:    liteLLMBaseURL,
		SpecContent:       agentRun.Spec.SpecContent,
		WorkspaceName:     agentRun.Spec.WorkspaceName,
		OrchestrationMode: aottemporal.OrchestrationMode(agentRun.Spec.OrchestrationMode),
		Orchestration:     orchTasks,
		ParentRunID:       agentRun.Spec.ParentRunID,
		SpecRunID:         agentRun.Spec.SpecRunID,
		MaxBudget:         agentRun.Spec.MaxBudget,
		AutoPush:          agentRun.Spec.AutoPush,
		AutoPR:            agentRun.Spec.AutoPR,
		PRBaseBranch:      agentRun.Spec.PRBaseBranch,
		Project:           agentRun.Spec.Project,
		Feature:           agentRun.Spec.Feature,
		Tags:              agentRun.Spec.Tags,
	}

	if agentRun.Spec.PipelineConfig != nil {
		input.PipelineConfig = &aottemporal.PipelineConfigInput{
			Plan: aottemporal.StageConfigInput{
				Model:          agentRun.Spec.PipelineConfig.Plan.Model,
				TimeoutSeconds: agentRun.Spec.PipelineConfig.Plan.TimeoutSeconds,
				MaxRetries:     agentRun.Spec.PipelineConfig.Plan.MaxRetries,
				OnFailure:      agentRun.Spec.PipelineConfig.Plan.OnFailure,
			},
			Execute: aottemporal.StageConfigInput{
				Model:          agentRun.Spec.PipelineConfig.Execute.Model,
				TimeoutSeconds: agentRun.Spec.PipelineConfig.Execute.TimeoutSeconds,
				MaxRetries:     agentRun.Spec.PipelineConfig.Execute.MaxRetries,
				OnFailure:      agentRun.Spec.PipelineConfig.Execute.OnFailure,
			},
			Verify: aottemporal.StageConfigInput{
				Model:          agentRun.Spec.PipelineConfig.Verify.Model,
				TimeoutSeconds: agentRun.Spec.PipelineConfig.Verify.TimeoutSeconds,
				MaxRetries:     agentRun.Spec.PipelineConfig.Verify.MaxRetries,
				OnFailure:      agentRun.Spec.PipelineConfig.Verify.OnFailure,
			},
		}
	}

	return input
}
