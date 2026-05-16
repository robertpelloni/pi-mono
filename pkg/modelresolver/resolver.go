package modelresolver

import (
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// ScopedModel represents a resolved model with an optional thinking level.
type ScopedModel struct {
	Model         ai.ModelInfo
	ThinkingLevel ai.ThinkingLevel
}

// Resolve resolves a model pattern string (possibly "provider/model" or "provider/model:thinking")
// into a ScopedModel using the model registry.
func Resolve(pattern string, registry *ModelRegistry) (*ScopedModel, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty model pattern")
	}

	provider, modelPattern, thinkingLevel := parseModelPattern(pattern)

	models := registry.FindByProvider(provider)
	if len(models) == 0 {
		// Try all providers
		models = registry.AllModels()
	}

	// Find the best match
	var bestMatch *ai.ModelInfo
	for i := range models {
		m := &models[i]
		if m.ID == modelPattern {
			bestMatch = m
			break
		}
		if bestMatch == nil && strings.Contains(strings.ToLower(m.ID), strings.ToLower(modelPattern)) {
			bestMatch = m
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no model found matching %q", pattern)
	}

	return &ScopedModel{
		Model:         *bestMatch,
		ThinkingLevel: thinkingLevel,
	}, nil
}

// ResolveWithProvider resolves a model using explicit provider and model pattern.
func ResolveWithProvider(providerName string, modelPattern string, registry *ModelRegistry) (*ScopedModel, error) {
	if modelPattern == "" {
		return nil, fmt.Errorf("no model specified")
	}

	thinkingLevel := ai.ThinkingLevel("")
	if strings.Contains(modelPattern, ":") {
		parts := strings.SplitN(modelPattern, ":", 2)
		modelPattern = parts[0]
		thinkingLevel = ai.ThinkingLevel(parts[1])
	}

	var provider ai.Provider
	switch strings.ToLower(providerName) {
	case "anthropic":
		provider = ai.ProviderAnthropic
	case "google", "gemini":
		provider = ai.ProviderGoogle
	case "openai":
		provider = ai.ProviderOpenAI
	default:
		provider = ai.Provider(providerName)
	}

	models := registry.FindByProvider(string(provider))

	var bestMatch *ai.ModelInfo
	for i := range models {
		m := &models[i]
		if m.ID == modelPattern {
			bestMatch = m
			break
		}
		if bestMatch == nil && strings.Contains(strings.ToLower(m.ID), strings.ToLower(modelPattern)) {
			bestMatch = m
		}
	}

	if bestMatch == nil {
		// Create a synthetic ModelInfo from what we know
		bestMatch = &ai.ModelInfo{
			ID:       modelPattern,
			Provider: provider,
			API:      providerToAPI(provider),
		}
	}

	return &ScopedModel{
		Model:         *bestMatch,
		ThinkingLevel: thinkingLevel,
	}, nil
}

// ResolveScope resolves a list of model patterns into ScopedModels.
func ResolveScope(patterns []string, registry *ModelRegistry) ([]ScopedModel, error) {
	var results []ScopedModel
	for _, p := range patterns {
		sm, err := Resolve(p, registry)
		if err != nil {
			continue // Skip unresolvable patterns
		}
		results = append(results, *sm)
	}
	return results, nil
}

// parseModelPattern parses a pattern like "anthropic/claude-sonnet-4:high"
// into provider="anthropic", model="claude-sonnet-4", thinking="high".
func parseModelPattern(pattern string) (provider, model string, thinking ai.ThinkingLevel) {
	pattern = strings.TrimSpace(pattern)

	// Split thinking level
	if colonIdx := strings.LastIndex(pattern, ":"); colonIdx > 0 {
		// Check if what follows is a valid thinking level
		after := pattern[colonIdx+1:]
		switch after {
		case "low", "medium", "high", "xhigh", "off":
			thinking = ai.ThinkingLevel(after)
			pattern = pattern[:colonIdx]
		}
	}

	// Split provider/model
	if slashIdx := strings.Index(pattern, "/"); slashIdx > 0 {
		provider = pattern[:slashIdx]
		model = pattern[slashIdx+1:]
	} else {
		model = pattern
	}

	return
}

// providerToAPI returns the default API type for a provider.
func providerToAPI(provider ai.Provider) ai.Api {
	switch provider {
	case ai.ProviderAnthropic:
		return ai.ApiAnthropicMessages
	case ai.ProviderGoogle:
		return ai.ApiGoogleGenerativeAI
	case ai.ProviderOpenAI:
		return ai.ApiOpenAIResponses
	case ai.ProviderAzureOpenAI:
		return ai.ApiAzureOpenAIResponses
	case ai.ProviderAmazonBedrock:
		return ai.ApiBedrockConverseStream
	case ai.ProviderMistral:
		return ai.ApiMistralConversations
	case ai.ProviderGeminiCLI:
		return ai.ApiGoogleGeminiCLI
	case ai.ProviderVertex:
		return ai.ApiGoogleVertex
	default:
		return ai.ApiOpenAICompletions
	}
}

// ModelRegistry is a simplified model registry for the Go port.
// In the TypeScript version, this is loaded from models.generated.ts.
// Here we use a lightweight in-memory registry that can be populated from
// the models.dev API or a static JSON file.
type ModelRegistry struct {
	models []ai.ModelInfo
}

// NewModelRegistry creates an empty model registry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{}
}

// NewModelRegistryWithDefaults creates a registry pre-populated with common models.
func NewModelRegistryWithDefaults() *ModelRegistry {
	r := &ModelRegistry{}
	r.RegisterDefaults()
	return r
}

// Register adds a model to the registry.
func (r *ModelRegistry) Register(model ai.ModelInfo) {
	r.models = append(r.models, model)
}

// Find looks up a model by provider and ID.
func (r *ModelRegistry) Find(provider, modelID string) *ai.ModelInfo {
	for i := range r.models {
		if r.models[i].Provider == ai.Provider(provider) && r.models[i].ID == modelID {
			return &r.models[i]
		}
	}
	return nil
}

// FindByProvider returns all models for a given provider.
func (r *ModelRegistry) FindByProvider(provider string) []ai.ModelInfo {
	var results []ai.ModelInfo
	for _, m := range r.models {
		if string(m.Provider) == provider {
			results = append(results, m)
		}
	}
	return results
}

// AllModels returns all registered models.
func (r *ModelRegistry) AllModels() []ai.ModelInfo {
	return r.models
}

// Search returns models matching a search pattern.
func (r *ModelRegistry) Search(pattern string) []ai.ModelInfo {
	pattern = strings.ToLower(pattern)
	var results []ai.ModelInfo
	for _, m := range r.models {
		if strings.Contains(strings.ToLower(m.ID), pattern) ||
			strings.Contains(strings.ToLower(m.Name), pattern) ||
			strings.Contains(strings.ToLower(string(m.Provider)), pattern) {
			results = append(results, m)
		}
	}
	return results
}

// RegisterDefaults populates the registry with commonly-used models.
func (r *ModelRegistry) RegisterDefaults() {
	// OpenAI
	for _, m := range []struct {
		id    string
		name  string
		api   ai.Api
		cost  ai.ModelCost
		ctx   int
		max   int
		reason bool
	}{
		{"gpt-4o", "GPT-4o", ai.ApiOpenAIResponses, ai.ModelCost{Input: 2.5, Output: 10}, 128000, 16384, false},
		{"gpt-4o-mini", "GPT-4o Mini", ai.ApiOpenAIResponses, ai.ModelCost{Input: 0.15, Output: 0.6}, 128000, 16384, false},
		{"gpt-4.1", "GPT-4.1", ai.ApiOpenAIResponses, ai.ModelCost{Input: 2, Output: 8}, 1047576, 32768, false},
		{"gpt-4.1-mini", "GPT-4.1 Mini", ai.ApiOpenAIResponses, ai.ModelCost{Input: 0.4, Output: 1.6}, 1047576, 32768, false},
		{"gpt-4.1-nano", "GPT-4.1 Nano", ai.ApiOpenAIResponses, ai.ModelCost{Input: 0.1, Output: 0.4}, 1047576, 32768, false},
		{"o3", "o3", ai.ApiOpenAIResponses, ai.ModelCost{Input: 2, Output: 8}, 200000, 100000, true},
		{"o3-mini", "o3 Mini", ai.ApiOpenAIResponses, ai.ModelCost{Input: 1.1, Output: 4.4}, 200000, 100000, true},
		{"o4-mini", "o4 Mini", ai.ApiOpenAIResponses, ai.ModelCost{Input: 1.1, Output: 4.4}, 200000, 100000, true},
		{"codex-mini-latest", "Codex Mini", ai.ApiOpenAICodexResponses, ai.ModelCost{Input: 1.5, Output: 6}, 192000, 100000, true},
	} {
		r.Register(ai.ModelInfo{
			ID:       m.id,
			Name:     m.name,
			API:      m.api,
			Provider: ai.ProviderOpenAI,
			BaseURL:  "https://api.openai.com/v1",
			Reasoning: m.reason,
			Input:    []string{"text", "image"},
			Cost:     m.cost,
			ContextWindow: m.ctx,
			MaxTokens: m.max,
		})
	}

	// Anthropic
	for _, m := range []struct {
		id    string
		name  string
		api   ai.Api
		cost  ai.ModelCost
		ctx   int
		max   int
		reason bool
	}{
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", ai.ApiAnthropicMessages, ai.ModelCost{Input: 3, Output: 15}, 200000, 16384, true},
		{"claude-opus-4-20250514", "Claude Opus 4", ai.ApiAnthropicMessages, ai.ModelCost{Input: 15, Output: 75}, 200000, 16384, true},
		{"claude-haiku-3-5-20241022", "Claude 3.5 Haiku", ai.ApiAnthropicMessages, ai.ModelCost{Input: 0.8, Output: 4}, 200000, 8192, false},
	} {
		r.Register(ai.ModelInfo{
			ID:       m.id,
			Name:     m.name,
			API:      m.api,
			Provider: ai.ProviderAnthropic,
			BaseURL:  "https://api.anthropic.com/v1",
			Reasoning: m.reason,
			Input:    []string{"text", "image"},
			Cost:     m.cost,
			ContextWindow: m.ctx,
			MaxTokens: m.max,
		})
	}

	// Google
	for _, m := range []struct {
		id    string
		name  string
		api   ai.Api
		cost  ai.ModelCost
		ctx   int
		max   int
		reason bool
	}{
		{"gemini-2.5-pro", "Gemini 2.5 Pro", ai.ApiGoogleGenerativeAI, ai.ModelCost{Input: 1.25, Output: 10}, 1048576, 65536, true},
		{"gemini-2.5-flash", "Gemini 2.5 Flash", ai.ApiGoogleGenerativeAI, ai.ModelCost{Input: 0.15, Output: 0.6}, 1048576, 65536, true},
		{"gemini-2.0-flash", "Gemini 2.0 Flash", ai.ApiGoogleGenerativeAI, ai.ModelCost{Input: 0.1, Output: 0.4}, 1048576, 8192, false},
	} {
		r.Register(ai.ModelInfo{
			ID:       m.id,
			Name:     m.name,
			API:      m.api,
			Provider: ai.ProviderGoogle,
			BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
			Reasoning: m.reason,
			Input:    []string{"text", "image"},
			Cost:     m.cost,
			ContextWindow: m.ctx,
			MaxTokens: m.max,
		})
	}
}
