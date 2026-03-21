package temporal

// ModelPricing contains per-million-token pricing in USD.
type ModelPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
	ContextWindow    int
}

var modelPricing = map[string]ModelPricing{
	"deepseek-v3.1":  {0.15, 0.75, 32768},
	"deepseek-v3.2":  {0.26, 0.38, 163840},
	"qwen3-coder":    {0.22, 1.00, 262144},
	"qwen3:8b":       {0.00, 0.00, 32768}, // local, no cost
	"mistral-medium": {0.40, 2.00, 131072},
	"default":        {0.15, 0.75, 32768},
	"default-cloud":  {0.15, 0.75, 32768},
}

// EstimateCost returns the estimated cost in USD for the given token usage.
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	p, ok := modelPricing[model]
	if !ok {
		p = modelPricing["default"]
	}
	return (float64(inputTokens)*p.InputPerMillion + float64(outputTokens)*p.OutputPerMillion) / 1_000_000
}

// GetModelPricing returns the pricing for a model, falling back to the default.
func GetModelPricing(model string) ModelPricing {
	p, ok := modelPricing[model]
	if !ok {
		return modelPricing["default"]
	}
	return p
}
