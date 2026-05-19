package sdk

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		modelID  string
		provider string
	}{
		{"gpt-4o", "openai"},
		{"claude-3-5-sonnet", "anthropic"},
		{"gemini-2.0-flash", "google"},
		{"unknown-model", "openai"}, // defaults to openai
	}

	for _, tt := range tests {
		result := detectProvider(tt.modelID)
		if result != tt.provider {
			t.Errorf("detectProvider(%q) = %q, want %q", tt.modelID, result, tt.provider)
		}
	}
}

func TestProviderAPIKeyEnv(t *testing.T) {
	tests := []struct {
		provider ai.Provider
		envVar   string
	}{
		{ai.ProviderOpenAI, "OPENAI_API_KEY"},
		{ai.ProviderAnthropic, "ANTHROPIC_API_KEY"},
		{ai.ProviderGoogle, "GEMINI_API_KEY"},
	}

	for _, tt := range tests {
		result := providerAPIKeyEnv(tt.provider)
		if result != tt.envVar {
			t.Errorf("providerAPIKeyEnv(%v) = %q, want %q", tt.provider, result, tt.envVar)
		}
	}
}

func TestProviderToAPI(t *testing.T) {
	tests := []struct {
		provider ai.Provider
		api      ai.Api
	}{
		{ai.ProviderOpenAI, ai.ApiOpenAIResponses},
		{ai.ProviderAnthropic, ai.ApiAnthropicMessages},
		{ai.ProviderGoogle, ai.ApiGoogleGenerativeAI},
	}

	for _, tt := range tests {
		result := providerToAPI(tt.provider)
		if result != tt.api {
			t.Errorf("providerToAPI(%v) = %v, want %v", tt.provider, result, tt.api)
		}
	}
}

func TestInitAgentDir(t *testing.T) {
	// This will try to create .pi directory — just test it doesn't panic
	_, _ = InitAgentDir()
}
