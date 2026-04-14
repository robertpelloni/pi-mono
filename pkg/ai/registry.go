package ai

import (
	"fmt"
	"sync"
)

// APIProvider represents a registered provider implementation for a specific API.
type APIProvider struct {
	API          Api
	Stream       StreamFunction
	StreamSimple StreamFunction // We use StreamFunction signature for both in Go, options will just be cast differently
}

type registeredAPIProvider struct {
	provider APIProvider
	sourceID string
}

var (
	registryMu          sync.RWMutex
	apiProviderRegistry = make(map[Api]registeredAPIProvider)
)

// wrapStream wraps a StreamFunction to validate the model API matches the registered API.
func wrapStream(api Api, stream StreamFunction) StreamFunction {
	return func(model ModelInfo, context Context, options any) AssistantMessageEventStream {
		if model.API != api {
			// In Go, since we can't easily throw an exception that fits into the stream protocol automatically,
			// we return a stream that immediately emits an error event.
			errStream := make(chan AssistantMessageEvent, 1)
			errMsg := fmt.Sprintf("Mismatched api: %s expected %s", model.API, api)
			reason := StopReasonError
			errStream <- AssistantMessageEvent{
				Type:   EventError,
				Reason: &reason,
				Error: &AssistantMessage{
					API:          model.API,
					Provider:     model.Provider,
					Model:        model.ID,
					StopReason:   reason,
					ErrorMessage: &errMsg,
				},
			}
			close(errStream)
			return errStream
		}
		return stream(model, context, options)
	}
}

// RegisterAPIProvider registers a new API provider implementation.
// sourceID is optional and used for clearing providers added by specific plugins/extensions.
func RegisterAPIProvider(provider APIProvider, sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	apiProviderRegistry[provider.API] = registeredAPIProvider{
		provider: APIProvider{
			API:          provider.API,
			Stream:       wrapStream(provider.API, provider.Stream),
			StreamSimple: wrapStream(provider.API, provider.StreamSimple),
		},
		sourceID: sourceID,
	}
}

// GetAPIProvider retrieves a registered provider implementation for the given API.
func GetAPIProvider(api Api) (*APIProvider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	entry, exists := apiProviderRegistry[api]
	if !exists {
		return nil, false
	}
	return &entry.provider, true
}

// GetAPIProviders returns all registered provider implementations.
func GetAPIProviders() []APIProvider {
	registryMu.RLock()
	defer registryMu.RUnlock()

	providers := make([]APIProvider, 0, len(apiProviderRegistry))
	for _, entry := range apiProviderRegistry {
		providers = append(providers, entry.provider)
	}
	return providers
}

// UnregisterAPIProviders removes all providers registered with the given sourceID.
func UnregisterAPIProviders(sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	for api, entry := range apiProviderRegistry {
		if entry.sourceID == sourceID {
			delete(apiProviderRegistry, api)
		}
	}
}

// ClearAPIProviders removes all registered providers.
func ClearAPIProviders() {
	registryMu.Lock()
	defer registryMu.Unlock()
	apiProviderRegistry = make(map[Api]registeredAPIProvider)
}

// RegisterBuiltInAPIProviders registers all the natively supported API providers.
func RegisterBuiltInAPIProviders() {
	RegisterAPIProvider(APIProvider{
		API:          ApiAnthropicMessages,
		Stream:       StreamAnthropic,
		StreamSimple: StreamAnthropic, // Using same stub for now
	}, "builtin")

	RegisterAPIProvider(APIProvider{
		API:          ApiOpenAIResponses,
		Stream:       StreamOpenAIResponses,
		StreamSimple: StreamOpenAIResponses, // Using same stub for now
	}, "builtin")

	RegisterAPIProvider(APIProvider{
		API:          ApiGoogleGenerativeAI,
		Stream:       StreamGoogle,
		StreamSimple: StreamGoogle, // Using same stub for now
	}, "builtin")

	RegisterAPIProvider(APIProvider{
		API:          ApiGoogleGeminiCLI,
		Stream:       StreamGoogleGeminiCli,
		StreamSimple: StreamGoogleGeminiCli, // Using same stub for now
	}, "builtin")

	RegisterAPIProvider(APIProvider{
		API:          ApiGoogleVertex,
		Stream:       StreamGoogleVertex,
		StreamSimple: StreamGoogleVertex, // Using same stub for now
	}, "builtin")

	// Other providers like Mistral, Azure, Bedrock, etc. will be added here
	// as their respective stubs/implementations are ported to Go.
}

// ResetAPIProviders clears and re-registers the built-in providers.
func ResetAPIProviders() {
	ClearAPIProviders()
	RegisterBuiltInAPIProviders()
}

func init() {
	// Automatically register built-ins on package initialization
	RegisterBuiltInAPIProviders()
}
