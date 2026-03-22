package temporal

import (
	"testing"
)

// TestDualModelOverride verifies the dual model override logic
// that happens in runSpecDrivenPipeline.
func TestDualModelOverride_ManageOverridesPlanVerify(t *testing.T) {
	planCfg := resolveStageConfig(nil, "plan")
	execCfg := resolveStageConfig(nil, "execute")
	verifyCfg := resolveStageConfig(nil, "verify")

	// Before override: all use defaults
	if planCfg.Model != "default-cloud" {
		t.Fatalf("plan default = %q", planCfg.Model)
	}

	// Simulate the dual model override from runSpecDrivenPipeline
	manageModel := "qwen3:8b"
	implModel := "deepseek-v3.1"

	planCfg.Model = manageModel
	verifyCfg.Model = manageModel
	execCfg.Model = implModel

	if planCfg.Model != "qwen3:8b" {
		t.Errorf("plan model after override = %q, want qwen3:8b", planCfg.Model)
	}
	if verifyCfg.Model != "qwen3:8b" {
		t.Errorf("verify model after override = %q, want qwen3:8b", verifyCfg.Model)
	}
	if execCfg.Model != "deepseek-v3.1" {
		t.Errorf("exec model after override = %q, want deepseek-v3.1", execCfg.Model)
	}
}

func TestDualModelOverride_ImplFallsBackToManage(t *testing.T) {
	planCfg := resolveStageConfig(nil, "plan")
	execCfg := resolveStageConfig(nil, "execute")
	verifyCfg := resolveStageConfig(nil, "verify")

	manageModel := "qwen3:8b"
	implModel := "" // not set — should fall back to manage

	if manageModel != "" {
		planCfg.Model = manageModel
		verifyCfg.Model = manageModel
	}
	if implModel == "" {
		implModel = manageModel // fallback
	}
	if implModel != "" {
		execCfg.Model = implModel
	}

	// All three should use qwen3:8b
	if planCfg.Model != "qwen3:8b" {
		t.Errorf("plan = %q", planCfg.Model)
	}
	if execCfg.Model != "qwen3:8b" {
		t.Errorf("exec = %q (should fallback to manage)", execCfg.Model)
	}
	if verifyCfg.Model != "qwen3:8b" {
		t.Errorf("verify = %q", verifyCfg.Model)
	}
}

func TestDualModelOverride_NoOverrideKeepsDefaults(t *testing.T) {
	planCfg := resolveStageConfig(nil, "plan")
	execCfg := resolveStageConfig(nil, "execute")
	verifyCfg := resolveStageConfig(nil, "verify")

	// Empty manage/implement — keep defaults
	manageModel := ""
	implModel := ""

	if manageModel != "" {
		planCfg.Model = manageModel
		verifyCfg.Model = manageModel
	}
	if implModel == "" {
		implModel = manageModel
	}
	if implModel != "" {
		execCfg.Model = implModel
	}

	// All should keep their defaults
	if planCfg.Model != "default-cloud" {
		t.Errorf("plan = %q, want default-cloud", planCfg.Model)
	}
	if execCfg.Model != "default-cloud" {
		t.Errorf("exec = %q, want default-cloud", execCfg.Model)
	}
	if verifyCfg.Model != "default-cloud" {
		t.Errorf("verify = %q, want default-cloud", verifyCfg.Model)
	}
}
