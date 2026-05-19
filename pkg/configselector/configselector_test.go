package configselector

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/auth"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
)

func TestGetEnvVarName(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"google", "GEMINI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"custom-provider", "CUSTOM_PROVIDER_API_KEY"},
	}

	for _, tt := range tests {
		result := getEnvVarName(tt.provider)
		if result != tt.expected {
			t.Errorf("getEnvVarName(%q) = %q, want %q", tt.provider, result, tt.expected)
		}
	}
}

func TestConfigSelectorResult_Fields(t *testing.T) {
	result := &ConfigSelectorResult{
		Model:         ai.ModelInfo{ID: "test-model"},
		ThinkingLevel: ai.ThinkingMedium,
		Provider:      "openai",
	}
	if result.Model.ID != "test-model" {
		t.Error("Model mismatch")
	}
	if result.ThinkingLevel != ai.ThinkingMedium {
		t.Error("ThinkingLevel mismatch")
	}
	if result.Provider != "openai" {
		t.Error("Provider mismatch")
	}
}

func TestSelectConfig_NoModels(t *testing.T) {
	registry := modelresolver.NewModelRegistryWithDefaults()
	auth := auth.NewAuthStorage("")

	result, warnings := SelectConfig(registry, auth, "", "")
	// With no API keys, should return warnings or nil result
	_ = result
	_ = warnings
}
