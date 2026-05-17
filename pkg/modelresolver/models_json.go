package modelresolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/auth"
)

// DefaultModelPerProvider maps known providers to their default model IDs.
var DefaultModelPerProvider = map[string]string{
	"amazon-bedrock":         "us.anthropic.claude-opus-4-6-v1",
	"anthropic":             "claude-opus-4-6",
	"openai":                "gpt-5.4",
	"azure-openai-responses": "gpt-5.2",
	"openai-codex":          "gpt-5.4",
	"google":                "gemini-2.5-pro",
	"google-gemini-cli":     "gemini-2.5-pro",
	"google-vertex":         "gemini-3-pro-preview",
	"github-copilot":        "gpt-4o",
	"openrouter":            "openai/gpt-5.1-codex",
	"vercel-ai-gateway":     "anthropic/claude-opus-4-6",
	"xai":                   "grok-4-fast-non-reasoning",
	"groq":                  "openai/gpt-oss-120b",
	"zai":                   "glm-5",
	"mistral":               "devstral-medium-latest",
	"huggingface":           "moonshotai/Kimi-K2.5",
}

// modelsJSONConfig represents the structure of models.json.
type modelsJSONConfig struct {
	Providers map[string]providerConfig `json:"providers"`
}

type providerConfig struct {
	BaseURL       string                       `json:"baseUrl,omitempty"`
	APIKey        string                       `json:"apiKey,omitempty"`
	API           string                       `json:"api,omitempty"`
	Headers       map[string]string            `json:"headers,omitempty"`
	AuthHeader    bool                         `json:"authHeader,omitempty"`
	Models        []modelDefinition            `json:"models,omitempty"`
	ModelOverrides map[string]modelOverride    `json:"modelOverrides,omitempty"`
	Compat        *ai.OpenAICompletionsCompat  `json:"compat,omitempty"`
}

type modelDefinition struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name,omitempty"`
	API           string                       `json:"api,omitempty"`
	BaseURL       string                       `json:"baseUrl,omitempty"`
	Reasoning     bool                         `json:"reasoning,omitempty"`
	Input         []string                     `json:"input,omitempty"`
	Cost          *modelCostDef                `json:"cost,omitempty"`
	ContextWindow int                          `json:"contextWindow,omitempty"`
	MaxTokens     int                          `json:"maxTokens,omitempty"`
	Headers       map[string]string            `json:"headers,omitempty"`
	Compat        *ai.OpenAICompletionsCompat  `json:"compat,omitempty"`
}

type modelOverride struct {
	Name          string                       `json:"name,omitempty"`
	Reasoning     *bool                        `json:"reasoning,omitempty"`
	Input         []string                     `json:"input,omitempty"`
	Cost          *modelCostDef                `json:"cost,omitempty"`
	ContextWindow *int                         `json:"contextWindow,omitempty"`
	MaxTokens     *int                         `json:"maxTokens,omitempty"`
	Headers       map[string]string            `json:"headers,omitempty"`
	Compat        *ai.OpenAICompletionsCompat  `json:"compat,omitempty"`
}

type modelCostDef struct {
	Input     float64 `json:"input,omitempty"`
	Output    float64 `json:"output,omitempty"`
	CacheRead float64 `json:"cacheRead,omitempty"`
	CacheWrite float64 `json:"cacheWrite,omitempty"`
}

// ProviderOverride stores provider-level overrides from models.json.
type ProviderOverride struct {
	BaseURL string
	Compat  *ai.OpenAICompletionsCompat
}

// ResolvedRequestAuth is the result of API key resolution for a model.
type ResolvedRequestAuth struct {
	OK      bool
	APIKey  string
	Headers map[string]string
	Error   string
}

// LoadModelsJSON loads custom models and overrides from a models.json file.
// Returns custom models, provider overrides, and any error.
func LoadModelsJSON(modelsJSONPath string, authStorage *auth.AuthStorage) (
	customModels []ai.ModelInfo,
	overrides map[string]ProviderOverride,
	loadError error,
) {
	overrides = make(map[string]ProviderOverride)

	if modelsJSONPath == "" {
		return nil, overrides, nil
	}

	content, err := os.ReadFile(modelsJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, overrides, nil
		}
		return nil, overrides, fmt.Errorf("failed to read models.json: %w", err)
	}

	var config modelsJSONConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, overrides, fmt.Errorf("failed to parse models.json: %w", err)
	}

	// Process each provider
	for providerName, providerCfg := range config.Providers {
		// Store provider override
		if providerCfg.BaseURL != "" || providerCfg.Compat != nil {
			overrides[providerName] = ProviderOverride{
				BaseURL: providerCfg.BaseURL,
				Compat:  providerCfg.Compat,
			}
		}

		// Parse custom models
		for _, modelDef := range providerCfg.Models {
			api := modelDef.API
			if api == "" {
				api = providerCfg.API
			}
			if api == "" {
				continue
			}

			input := modelDef.Input
			if len(input) == 0 {
				input = []string{"text"}
			}

			cost := ai.ModelCost{}
			if modelDef.Cost != nil {
				cost = ai.ModelCost{
					Input:     modelDef.Cost.Input,
					Output:    modelDef.Cost.Output,
					CacheRead: modelDef.Cost.CacheRead,
					CacheWrite: modelDef.Cost.CacheWrite,
				}
			}

			baseURL := modelDef.BaseURL
			if baseURL == "" {
				baseURL = providerCfg.BaseURL
			}

			name := modelDef.Name
			if name == "" {
				name = modelDef.ID
			}

			model := ai.ModelInfo{
				ID:            modelDef.ID,
				Name:          name,
				API:           ai.Api(api),
				Provider:      ai.Provider(providerName),
				BaseURL:       baseURL,
				Reasoning:     modelDef.Reasoning,
				Input:         input,
				Cost:          cost,
				ContextWindow: modelDef.ContextWindow,
				MaxTokens:     modelDef.MaxTokens,
				Headers:       modelDef.Headers,
				Compat:        modelDef.Compat,
			}

			// Apply defaults
			if model.ContextWindow == 0 {
				model.ContextWindow = 128000
			}
			if model.MaxTokens == 0 {
				model.MaxTokens = 16384
			}

			customModels = append(customModels, model)
		}
	}

	return customModels, overrides, nil
}

// FindInitialModel finds the initial model based on priority:
// 1. CLI args (provider + model)
// 2. Scoped models first
// 3. Saved default from settings
// 4. First available model with valid API key
func FindInitialModel(options FindInitialModelOptions) (*ScopedModel, string) {
	// 1. Try CLI provider + model
	if options.CLIProvider != "" && options.CLIModel != "" {
		sm, err := ResolveWithProvider(options.CLIProvider, options.CLIModel, options.ModelRegistry)
		if err == nil && sm != nil {
			return sm, ""
		}
	}

	// 2. Use first from scoped models (skip if continuing)
	if len(options.ScopedModels) > 0 && !options.IsContinuing {
		return &options.ScopedModels[0], ""
	}

	// 3. Try saved default
	if options.DefaultProvider != "" && options.DefaultModelID != "" {
		found := options.ModelRegistry.Find(options.DefaultProvider, options.DefaultModelID)
		if found != nil {
			tl := ai.ThinkingLevel("medium")
			if options.DefaultThinkingLevel != "" {
				tl = options.DefaultThinkingLevel
			}
			return &ScopedModel{Model: *found, ThinkingLevel: tl}, ""
		}
	}

	// 4. First available with auth
	available := options.ModelRegistry.GetAvailable(options.AuthStorage)
	if len(available) > 0 {
		// Try default per provider first
		for provider, defaultID := range DefaultModelPerProvider {
			for _, m := range available {
				if string(m.Provider) == provider && m.ID == defaultID {
					return &ScopedModel{Model: m, ThinkingLevel: ai.ThinkingLevel("medium")}, ""
				}
			}
		}
		// Fall back to first available
		return &ScopedModel{Model: available[0], ThinkingLevel: ai.ThinkingLevel("medium")}, ""
	}

	// 5. No model found
	return nil, "No models available. Set API keys in environment variables."
}

// FindInitialModelOptions configures model finding.
type FindInitialModelOptions struct {
	CLIProvider         string
	CLIModel            string
	ScopedModels        []ScopedModel
	IsContinuing        bool
	DefaultProvider     string
	DefaultModelID      string
	DefaultThinkingLevel ai.ThinkingLevel
	ModelRegistry       *ModelRegistry
	AuthStorage         *auth.AuthStorage
}

// GetAvailable returns models that have auth configured.
func (r *ModelRegistry) GetAvailable(authStorage *auth.AuthStorage) []ai.ModelInfo {
	var available []ai.ModelInfo
	for _, m := range r.models {
		if authStorage != nil && authStorage.HasAuth(string(m.Provider)) {
			available = append(available, m)
		} else if authStorage == nil {
			// No auth storage - include all
			available = append(available, m)
		}
	}
	return available
}

// LoadFromModelsJSON loads custom models from models.json and merges them.
func (r *ModelRegistry) LoadFromModelsJSON(modelsJSONPath string, authStorage *auth.AuthStorage) error {
	customModels, overrides, err := LoadModelsJSON(modelsJSONPath, authStorage)
	if err != nil {
		return err
	}

	// Apply provider overrides to built-in models
	for i, m := range r.models {
		if override, ok := overrides[string(m.Provider)]; ok {
			if override.BaseURL != "" {
				r.models[i].BaseURL = override.BaseURL
			}
			if override.Compat != nil {
				r.models[i].Compat = override.Compat
			}
		}
	}

	// Merge custom models (custom wins on conflicts)
	for _, custom := range customModels {
		existingIdx := -1
		for i, m := range r.models {
			if m.Provider == custom.Provider && m.ID == custom.ID {
				existingIdx = i
				break
			}
		}
		if existingIdx >= 0 {
			r.models[existingIdx] = custom
		} else {
			r.models = append(r.models, custom)
		}
	}

	return nil
}

// GetModelsJSONPath returns the default path for models.json.
func GetModelsJSONPath(agentDir string) string {
	if agentDir == "" {
		home, _ := os.UserHomeDir()
		agentDir = filepath.Join(home, ".pi")
	}
	return filepath.Join(agentDir, "models.json")
}

// DetectProvider detects the provider from a model ID.
func DetectProvider(modelID string) string {
	m := strings.ToLower(modelID)
	if strings.HasPrefix(m, "claude") {
		return "anthropic"
	}
	if strings.HasPrefix(m, "gemini") || strings.HasPrefix(m, "gemma-") {
		return "google"
	}
	if strings.HasPrefix(m, "gpt") || strings.HasPrefix(m, "o1-") || strings.HasPrefix(m, "o3-") || strings.HasPrefix(m, "o4-") || strings.HasPrefix(m, "codex") {
		return "openai"
	}
	if strings.HasPrefix(m, "grok") {
		return "xai"
	}
	if strings.HasPrefix(m, "devstral") || strings.HasPrefix(m, "mistral") {
		return "mistral"
	}
	return "openai"
}

// Ensure imports are used
var _ = filepath.Join
var _ = fmt.Sprintf
