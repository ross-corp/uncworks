package bff

import "strings"

// displaySpanName remaps legacy span names for display.
func displaySpanName(name string) string {
	return strings.NewReplacer(
		"unc.", "manage.",
		"neph.", "implement.",
		"impl.", "implement.",
	).Replace(name)
}

// ModelPricing contains per-million-token pricing in USD.
type ModelPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

var modelPricing = map[string]ModelPricing{
	"deepseek-v3.1":  {0.15, 0.75},
	"deepseek-v3.2":  {0.26, 0.38},
	"qwen3-coder":    {0.22, 1.00},
	"qwen3:8b":       {0.00, 0.00},
	"mistral-medium": {0.40, 2.00},
	"default":        {0.15, 0.75},
}

// EstimateCost computes estimated USD cost from token counts and model name.
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	p, ok := modelPricing[model]
	if !ok {
		p = modelPricing["default"]
	}
	return (float64(inputTokens)*p.InputPerMillion + float64(outputTokens)*p.OutputPerMillion) / 1_000_000
}
