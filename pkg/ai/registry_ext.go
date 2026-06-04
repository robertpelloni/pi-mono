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
	modelsRegistryMu.RLock()
	defer modelsRegistryMu.RUnlock()
	// Simple stub for parity tools - in a real implementation this would be configurable.
	for _, providerModels := range modelsRegistry {
		for _, model := range providerModels {
			return &model
		}
	}
	return nil
}
