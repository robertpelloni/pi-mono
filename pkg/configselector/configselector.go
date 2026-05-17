package configselector

import (
	"fmt"
	"os"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/auth"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
)

// ConfigSelectorResult holds the result of a config selection.
type ConfigSelectorResult struct {
	Model         ai.ModelInfo
	ThinkingLevel ai.ThinkingLevel
	Provider      string
}

// SelectConfig selects the appropriate model and configuration.
// This implements the CLI-level config resolution that the TypeScript version
// performs in interactive-mode.ts.
func SelectConfig(
	registry *modelresolver.ModelRegistry,
	authStorage *auth.AuthStorage,
	provider string,
	modelPattern string,
) (*ConfigSelectorResult, []string) {
	var warnings []string

	// If explicit provider + model provided
	if provider != "" && modelPattern != "" {
		sm, err := modelresolver.ResolveWithProvider(provider, modelPattern, registry)
		if err != nil {
			return nil, []string{fmt.Sprintf("Model not found: %s/%s", provider, modelPattern)}
		}
		return &ConfigSelectorResult{
			Model:         sm.Model,
			ThinkingLevel: sm.ThinkingLevel,
			Provider:      provider,
		}, warnings
	}

	// If only model pattern provided (might include provider prefix)
	if modelPattern != "" {
		sm, err := modelresolver.Resolve(modelPattern, registry)
		if err != nil {
			// Try detecting provider from model ID
			detectedProvider := modelresolver.DetectProvider(modelPattern)
			sm, err = modelresolver.ResolveWithProvider(detectedProvider, modelPattern, registry)
			if err != nil {
				return nil, []string{fmt.Sprintf("Model not found: %s", modelPattern)}
			}
		}
		return &ConfigSelectorResult{
			Model:         sm.Model,
			ThinkingLevel: sm.ThinkingLevel,
			Provider:      string(sm.Model.Provider),
		}, warnings
	}

	// Auto-detect: find first available model
	available := registry.GetAvailable(authStorage)
	if len(available) == 0 {
		return nil, []string{"No models available. Set API keys in environment variables."}
	}

	// Prefer known default models
	for providerName, defaultID := range modelresolver.DefaultModelPerProvider {
		for _, m := range available {
			if string(m.Provider) == providerName && strings.HasPrefix(m.ID, defaultID[:strings.LastIndex(defaultID, "-")+1]) {
				return &ConfigSelectorResult{
					Model:         m,
					ThinkingLevel: ai.ThinkingLevel("medium"),
					Provider:      providerName,
				}, warnings
			}
		}
	}

	// Fall back to first available
	model := available[0]
	return &ConfigSelectorResult{
		Model:         model,
		ThinkingLevel: ai.ThinkingLevel("medium"),
		Provider:      string(model.Provider),
	}, warnings
}

// GetAPIKey resolves the API key for a given model.
func GetAPIKey(authStorage *auth.AuthStorage, model ai.ModelInfo) (string, error) {
	key := authStorage.GetAPIKey(string(model.Provider))
	if key == "" {
		return "", fmt.Errorf("no API key found for %s. Set the %s environment variable or use /login",
			model.Provider, getEnvVarName(string(model.Provider)))
	}
	return key, nil
}

func getEnvVarName(provider string) string {
	switch provider {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "google", "gemini":
		return "GEMINI_API_KEY"
	default:
		return strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_API_KEY"
	}
}

// Ensure os is used
var _ = os.Getenv
