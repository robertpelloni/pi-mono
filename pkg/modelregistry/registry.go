package modelregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
)

// ProviderInfo describes a provider and how to authenticate with it.
type ProviderInfo struct {
	Name        string   `json:"name"`
	EnvVars     []string `json:"envVars"`     // Environment variables that provide API keys
	Label       string   `json:"label"`       // Human-readable name
	DocsURL     string   `json:"docsUrl"`     // URL for setup instructions
	OAuthSupported bool  `json:"oauthSupported"`
}

// ModelRegistry manages available models and API key resolution.
type ModelRegistry struct {
	mu        sync.RWMutex
	providers map[string]ProviderInfo
	models    []ai.ModelInfo
}

// NewModelRegistry creates a registry with default providers and models.
func NewModelRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		providers: make(map[string]ProviderInfo),
	}

	// Use modelresolver's defaults as our model source
	mr := modelresolver.NewModelRegistryWithDefaults()
	registry.models = mr.AllModels()

	// Register known providers
	registry.RegisterProvider(ProviderInfo{
		Name:     "openai",
		EnvVars:  []string{"OPENAI_API_KEY"},
		Label:    "OpenAI",
		DocsURL:  "https://platform.openai.com/api-keys",
	})
	registry.RegisterProvider(ProviderInfo{
		Name:     "anthropic",
		EnvVars:  []string{"ANTHROPIC_API_KEY"},
		Label:    "Anthropic",
		DocsURL:  "https://console.anthropic.com/settings/keys",
	})
	registry.RegisterProvider(ProviderInfo{
		Name:     "google",
		EnvVars:  []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		Label:    "Google",
		DocsURL:  "https://aistudio.google.com/apikey",
	})

	return registry
}

// RegisterProvider adds a provider to the registry.
func (r *ModelRegistry) RegisterProvider(info ProviderInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[info.Name] = info
}

// GetProviderInfo returns information about a provider.
func (r *ModelRegistry) GetProviderInfo(name string) (ProviderInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.providers[name]
	return info, ok
}

// HasAPIKey checks if an API key is available for a provider.
func (r *ModelRegistry) HasAPIKey(providerName string) bool {
	info, ok := r.GetProviderInfo(providerName)
	if !ok {
		return false
	}
	for _, envVar := range info.EnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}

// GetAPIKey returns the API key for a provider.
func (r *ModelRegistry) GetAPIKey(providerName string) (string, error) {
	info, ok := r.GetProviderInfo(providerName)
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", providerName)
	}
	for _, envVar := range info.EnvVars {
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
	}
	return "", fmt.Errorf("no API key found for %s. Set one of: %s", info.Label, strings.Join(info.EnvVars, ", "))
}

// GetAllProviders returns all registered providers.
func (r *ModelRegistry) GetAllProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []ProviderInfo
	for _, info := range r.providers {
		result = append(result, info)
	}
	return result
}

// GetAvailableProviders returns providers that have API keys configured.
func (r *ModelRegistry) GetAvailableProviders() []ProviderInfo {
	all := r.GetAllProviders()
	var available []ProviderInfo
	for _, info := range all {
		if r.HasAPIKey(info.Name) {
			available = append(available, info)
		}
	}
	return available
}

// GetAllModels returns all known models.
func (r *ModelRegistry) GetAllModels() []ai.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ai.ModelInfo, len(r.models))
	copy(result, r.models)
	return result
}

// GetAvailableModels returns models for providers that have API keys.
func (r *ModelRegistry) GetAvailableModels() []ai.ModelInfo {
	all := r.GetAllModels()
	var available []ai.ModelInfo
	for _, model := range all {
		if r.HasAPIKey(string(model.Provider)) {
			available = append(available, model)
		}
	}
	return available
}

// GetModelsForProvider returns all models for a specific provider.
func (r *ModelRegistry) GetModelsForProvider(provider string) []ai.ModelInfo {
	all := r.GetAllModels()
	var result []ai.ModelInfo
	for _, model := range all {
		if string(model.Provider) == provider {
			result = append(result, model)
		}
	}
	return result
}

// SearchModels searches for models matching a pattern.
func (r *ModelRegistry) SearchModels(pattern string) []ai.ModelInfo {
	pattern = strings.ToLower(pattern)
	all := r.GetAllModels()
	var result []ai.ModelInfo
	for _, model := range all {
		if strings.Contains(strings.ToLower(model.ID), pattern) ||
			strings.Contains(strings.ToLower(model.Name), pattern) ||
			strings.Contains(strings.ToLower(string(model.Provider)), pattern) {
			result = append(result, model)
		}
	}
	return result
}

// FindModelByID finds a model by its ID.
func (r *ModelRegistry) FindModelByID(id string) (ai.ModelInfo, bool) {
	all := r.GetAllModels()
	for _, model := range all {
		if model.ID == id {
			return model, true
		}
	}
	return ai.ModelInfo{}, false
}

// ResolveProvider determines the provider from a model ID.
func (r *ModelRegistry) ResolveProvider(modelID string) (string, error) {
	// Check if model ID contains a provider prefix
	if idx := strings.Index(modelID, "/"); idx > 0 {
		return modelID[:idx], nil
	}

	// Look up model in registry
	if model, ok := r.FindModelByID(modelID); ok {
		return string(model.Provider), nil
	}

	// Try to infer from model name patterns
	lower := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") || strings.HasPrefix(lower, "codex"):
		return "openai", nil
	case strings.HasPrefix(lower, "claude-"):
		return "anthropic", nil
	case strings.HasPrefix(lower, "gemini-"):
		return "google", nil
	default:
		return "", fmt.Errorf("cannot determine provider for model: %s", modelID)
	}
}

// FormatProviderStatus returns a string showing API key status for all providers.
func (r *ModelRegistry) FormatProviderStatus() string {
	all := r.GetAllProviders()
	var lines []string
	for _, info := range all {
		status := "not configured"
		if r.HasAPIKey(info.Name) {
			status = "configured"
		}
		lines = append(lines, fmt.Sprintf("  %-12s %s (%s)", info.Label, status, strings.Join(info.EnvVars, ", ")))
	}
	return strings.Join(lines, "\n")
}

// LoadFromProviderCache loads models from a provider-cache.json file.
// This is the format used by the TypeScript version to store dynamically
// discovered models (e.g., from Ollama Cloud extension).
func (r *ModelRegistry) LoadFromProviderCache(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading provider cache: %w", err)
	}

	var cache struct {
		Providers map[string]struct {
			Provider string `json:"provider"`
			Models   []struct {
				ID            string            `json:"id"`
				Name          string            `json:"name"`
				Reasoning     bool              `json:"reasoning"`
				Input         []string          `json:"input"`
				ContextWindow int               `json:"contextWindow"`
				MaxTokens     int               `json:"maxTokens"`
				ThinkingMap   map[string]string `json:"thinkingLevelMap"`
				Compat        struct {
					SupportsDeveloperRole     bool `json:"supportsDeveloperRole"`
					SupportsReasoningEffort   bool `json:"supportsReasoningEffort"`
				} `json:"compat"`
			} `json:"models"`
		} `json:"providers"`
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("parsing provider cache: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for provName, provData := range cache.Providers {
		providerName := provData.Provider
		if providerName == "" {
			providerName = provName
		}

		// Register provider if not already known
		if _, exists := r.providers[providerName]; !exists {
			envVars := []string{}
			switch providerName {
			case "ollama-cloud":
				envVars = []string{"OLLAMA_API_KEY"}
			default:
				envVars = []string{strings.ToUpper(strings.ReplaceAll(providerName, "-", "_")) + "_API_KEY"}
			}
			r.providers[providerName] = ProviderInfo{
				Name:   providerName,
				EnvVars: envVars,
				Label:  providerName,
			}
		}

		// Add models
		for _, m := range provData.Models {
			// Check if model already exists
			exists := false
			for _, existing := range r.models {
				if existing.ID == m.ID && string(existing.Provider) == providerName {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			model := ai.ModelInfo{
				ID:            m.ID,
				Name:          m.Name,
				Provider:      ai.Provider(providerName),
				ContextWindow: m.ContextWindow,
				MaxTokens:     m.MaxTokens,
				Reasoning:     m.Reasoning,
				Input:         m.Input,
			}

			// Set API type based on provider
			switch ai.Provider(providerName) {
			case ai.ProviderOllama:
				model.API = ai.ApiOpenAICompletions
				model.BaseURL = "https://ollama.com/v1"
			default:
				model.API = ai.ApiOpenAICompletions
			}

			r.models = append(r.models, model)
		}
	}

	return nil
}


// LoadProviderCacheModels reads a provider-cache.json file and returns
// the models as a flat slice of ai.ModelInfo. This is used by cmd/pi
// to merge extension-discovered models into the main model registry.
func LoadProviderCacheModels(path string) ([]ai.ModelInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading provider cache: %w", err)
	}

	var cache struct {
		Providers map[string]struct {
			Provider string `json:"provider"`
			Models   []struct {
				ID            string   `json:"id"`
				Name          string   `json:"name"`
				Reasoning     bool     `json:"reasoning"`
				Input         []string `json:"input"`
				ContextWindow int      `json:"contextWindow"`
				MaxTokens     int      `json:"maxTokens"`
			} `json:"models"`
		} `json:"providers"`
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing provider cache: %w", err)
	}

	var models []ai.ModelInfo
	for provName, provData := range cache.Providers {
		providerName := provData.Provider
		if providerName == "" {
			providerName = provName
		}
		for _, m := range provData.Models {
			model := ai.ModelInfo{
				ID:            m.ID,
				Name:          m.Name,
				Provider:      ai.Provider(providerName),
				ContextWindow: m.ContextWindow,
				MaxTokens:     m.MaxTokens,
				Reasoning:     m.Reasoning,
				Input:         m.Input,
			}
			switch ai.Provider(providerName) {
			case ai.ProviderOllama:
				model.API = ai.ApiOpenAICompletions
				model.BaseURL = "https://ollama.com/v1"
			default:
				model.API = ai.ApiOpenAICompletions
			}
			models = append(models, model)
		}
	}
	return models, nil
}
