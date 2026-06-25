package ai

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages model and provider registration.
type Registry struct {
	mu             sync.RWMutex
	models         map[Provider]map[string]ModelInfo
	apiProviders   map[Api]registeredAPIProvider
	defaultModelID string
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		models:       make(map[Provider]map[string]ModelInfo),
		apiProviders: make(map[Api]registeredAPIProvider),
	}
}

// RegisterModel adds a model to this registry instance.
func (r *Registry) RegisterModel(model ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.models == nil {
		r.models = make(map[Provider]map[string]ModelInfo)
	}

	providerModels, exists := r.models[model.Provider]
	if !exists {
		providerModels = make(map[string]ModelInfo)
		r.models[model.Provider] = providerModels
	}
	providerModels[model.ID] = model
}

// Stream executes a streaming request to the model.
func (m *ModelInfo) Stream(ctx context.Context, aiCtx Context, options any) (AssistantMessageEventStream, error) {
	provider, exists := GetAPIProvider(m.API)
	if !exists {
		return nil, fmt.Errorf("no provider registered for api %s", m.API)
	}
	return provider.Stream(ctx, *m, aiCtx, options), nil
}

// GetDefaultModel returns the default model from the registry.
func (r *Registry) GetDefaultModel() *ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Simple stub for parity tools - in a real implementation this would be configurable.
	for _, providerModels := range r.models {
		for _, model := range providerModels {
			return &model
		}
	}
	return nil
}
