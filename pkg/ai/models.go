package ai

import "strings"

// CalculateCost calculates the cost of an API call based on the model's pricing and token usage.
// The calculated cost is updated directly in the passed Usage struct's Cost field.
func CalculateCost(model ModelInfo, usage *Usage) UsageCost {
	usage.Cost.Input = (model.Cost.Input / 1000000.0) * float64(usage.Input)
	usage.Cost.Output = (model.Cost.Output / 1000000.0) * float64(usage.Output)
	usage.Cost.CacheRead = (model.Cost.CacheRead / 1000000.0) * float64(usage.CacheRead)
	usage.Cost.CacheWrite = (model.Cost.CacheWrite / 1000000.0) * float64(usage.CacheWrite)
	usage.Cost.Total = usage.Cost.Input + usage.Cost.Output + usage.Cost.CacheRead + usage.Cost.CacheWrite
	return usage.Cost
}

// SupportsXHigh checks if a model supports the "xhigh" reasoning level.
// Supported today:
// - GPT-5.2 / GPT-5.3 / GPT-5.4 model families
// - Opus 4.6 models (xhigh maps to adaptive effort "max" on Anthropic-compatible providers)
func SupportsXHigh(model ModelInfo) bool {
	id := strings.ToLower(model.ID)

	if strings.Contains(id, "gpt-5.2") || strings.Contains(id, "gpt-5.3") || strings.Contains(id, "gpt-5.4") {
		return true
	}

	if strings.Contains(id, "opus-4-6") || strings.Contains(id, "opus-4.6") {
		return true
	}

	return false
}

// ModelsAreEqual checks if two models are identical by comparing their ID and Provider.
func ModelsAreEqual(a *ModelInfo, b *ModelInfo) bool {
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID && a.Provider == b.Provider
}
