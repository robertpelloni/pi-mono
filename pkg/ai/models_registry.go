package ai

import "sync"

var (
	modelsRegistryMu sync.RWMutex
	modelsRegistry   = make(map[Provider]map[string]ModelInfo)
)

// RegisterModel adds a new model to the global registry under its provider.
// If the provider doesn't exist in the registry, it is created.
// This function replaces manual iteration over MODELS objects in the TypeScript initialization.
func RegisterModel(model ModelInfo) {
	modelsRegistryMu.Lock()
	defer modelsRegistryMu.Unlock()

	providerModels, exists := modelsRegistry[model.Provider]
	if !exists {
		providerModels = make(map[string]ModelInfo)
		modelsRegistry[model.Provider] = providerModels
	}
	providerModels[model.ID] = model
}

// GetModel retrieves a specific model from the global registry by provider and model ID.
// Returns a pointer to the ModelInfo and a boolean indicating if it was found.
func GetModel(provider Provider, modelID string) (*ModelInfo, bool) {
	modelsRegistryMu.RLock()
	defer modelsRegistryMu.RUnlock()

	providerModels, providerExists := modelsRegistry[provider]
	if !providerExists {
		return nil, false
	}

	model, modelExists := providerModels[modelID]
	if !modelExists {
		return nil, false
	}

	return &model, true
}

// GetProviders returns a slice of all providers currently holding registered models.
func GetProviders() []Provider {
	modelsRegistryMu.RLock()
	defer modelsRegistryMu.RUnlock()

	providers := make([]Provider, 0, len(modelsRegistry))
	for provider := range modelsRegistry {
		providers = append(providers, provider)
	}

	return providers
}

// GetModels returns a slice of all registered models for a given provider.
func GetModels(provider Provider) []ModelInfo {
	modelsRegistryMu.RLock()
	defer modelsRegistryMu.RUnlock()

	providerModels, exists := modelsRegistry[provider]
	if !exists {
		return nil
	}

	models := make([]ModelInfo, 0, len(providerModels))
	for _, model := range providerModels {
		models = append(models, model)
	}

	return models
}

// ClearModelsRegistry removes all models from the global registry.
func ClearModelsRegistry() {
	modelsRegistryMu.Lock()
	defer modelsRegistryMu.Unlock()

	modelsRegistry = make(map[Provider]map[string]ModelInfo)
}
