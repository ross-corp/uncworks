package temporal

import (
	"math"
	"testing"
)

func TestEstimateCost_KnownModel(t *testing.T) {
	// deepseek-v3.1: input=$0.15/M, output=$0.75/M
	cost := EstimateCost("deepseek-v3.1", 1_000_000, 1_000_000)
	expected := 0.15 + 0.75 // $0.90 for 1M input + 1M output
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("EstimateCost(deepseek-v3.1, 1M, 1M) = %f, want %f", cost, expected)
	}
}

func TestEstimateCost_KnownModel_Qwen(t *testing.T) {
	// qwen3-coder: input=$0.22/M, output=$1.00/M
	cost := EstimateCost("qwen3-coder", 500_000, 200_000)
	expected := (500_000*0.22 + 200_000*1.00) / 1_000_000
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("EstimateCost(qwen3-coder, 500k, 200k) = %f, want %f", cost, expected)
	}
}

func TestEstimateCost_UnknownModel_FallsBackToDefault(t *testing.T) {
	// Unknown model should use "default" pricing: input=$0.15/M, output=$0.75/M
	costUnknown := EstimateCost("totally-unknown-model-v99", 1_000_000, 1_000_000)
	costDefault := EstimateCost("default", 1_000_000, 1_000_000)
	if costUnknown != costDefault {
		t.Errorf("unknown model cost = %f, default cost = %f — should be equal", costUnknown, costDefault)
	}
}

func TestEstimateCost_ZeroTokens(t *testing.T) {
	cost := EstimateCost("deepseek-v3.1", 0, 0)
	if cost != 0 {
		t.Errorf("EstimateCost with zero tokens = %f, want 0", cost)
	}
}

func TestEstimateCost_LocalModelFree(t *testing.T) {
	// qwen3:8b is local with zero pricing
	cost := EstimateCost("qwen3:8b", 10_000, 5_000)
	if cost != 0 {
		t.Errorf("EstimateCost(qwen3:8b) = %f, want 0 (local model)", cost)
	}
}

func TestGetModelPricing_KnownModel(t *testing.T) {
	p := GetModelPricing("mistral-medium")
	if p.InputPerMillion != 0.40 {
		t.Errorf("InputPerMillion = %f, want 0.40", p.InputPerMillion)
	}
	if p.OutputPerMillion != 2.00 {
		t.Errorf("OutputPerMillion = %f, want 2.00", p.OutputPerMillion)
	}
	if p.ContextWindow != 131072 {
		t.Errorf("ContextWindow = %d, want 131072", p.ContextWindow)
	}
}

func TestGetModelPricing_UnknownModel(t *testing.T) {
	p := GetModelPricing("nonexistent-model")
	d := GetModelPricing("default")
	if p != d {
		t.Errorf("unknown model pricing %+v != default pricing %+v", p, d)
	}
}
