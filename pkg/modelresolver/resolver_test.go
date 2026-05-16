package modelresolver

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestNewModelRegistryWithDefaults(t *testing.T) {
	r := NewModelRegistryWithDefaults()
	models := r.AllModels()

	if len(models) == 0 {
		t.Error("default registry should have models")
	}
}

func TestFindByProvider(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	openaiModels := r.FindByProvider("openai")
	if len(openaiModels) == 0 {
		t.Error("should have OpenAI models")
	}

	anthropicModels := r.FindByProvider("anthropic")
	if len(anthropicModels) == 0 {
		t.Error("should have Anthropic models")
	}

	googleModels := r.FindByProvider("google")
	if len(googleModels) == 0 {
		t.Error("should have Google models")
	}
}

func TestFind(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	model := r.Find("openai", "gpt-4o")
	if model == nil {
		t.Fatal("should find gpt-4o")
	}
	if model.ID != "gpt-4o" {
		t.Errorf("expected ID 'gpt-4o', got '%s'", model.ID)
	}
	if model.Provider != ai.ProviderOpenAI {
		t.Errorf("expected Provider 'openai', got '%s'", model.Provider)
	}

	missing := r.Find("openai", "nonexistent-model")
	if missing != nil {
		t.Error("should not find nonexistent model")
	}
}

func TestSearch(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	results := r.Search("gpt")
	if len(results) == 0 {
		t.Error("search for 'gpt' should return results")
	}

	results = r.Search("claude")
	if len(results) == 0 {
		t.Error("search for 'claude' should return results")
	}

	results = r.Search("nonexistent")
	if len(results) != 0 {
		t.Error("search for nonexistent should return empty")
	}
}

func TestResolve(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	sm, err := Resolve("gpt-4o", r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm.Model.ID != "gpt-4o" {
		t.Errorf("expected 'gpt-4o', got '%s'", sm.Model.ID)
	}
}

func TestResolveWithProvider(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	sm, err := ResolveWithProvider("anthropic", "claude-sonnet-4-20250514", r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm.Model.Provider != ai.ProviderAnthropic {
		t.Errorf("expected anthropic provider, got '%s'", sm.Model.Provider)
	}
}

func TestResolveWithThinking(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	sm, err := Resolve("anthropic/claude-sonnet-4-20250514:high", r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm.ThinkingLevel != ai.ThinkingLevel("high") {
		t.Errorf("expected thinking level 'high', got '%s'", sm.ThinkingLevel)
	}
}

func TestResolveEmpty(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	_, err := Resolve("", r)
	if err == nil {
		t.Error("empty pattern should return error")
	}
}

func TestResolveNotFound(t *testing.T) {
	r := NewModelRegistryWithDefaults()

	_, err := Resolve("nonexistent-model-xyz", r)
	if err == nil {
		t.Error("nonexistent model should return error")
	}
}

func TestParseModelPattern(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		model    string
		thinking ai.ThinkingLevel
	}{
		{"gpt-4o", "", "gpt-4o", ""},
		{"anthropic/claude-sonnet-4:high", "anthropic", "claude-sonnet-4", "high"},
		{"google/gemini-2.5-pro:low", "google", "gemini-2.5-pro", "low"},
		{"o3:medium", "", "o3", "medium"},
		{"openai/gpt-4o-mini:off", "openai", "gpt-4o-mini", "off"},
	}

	for _, tc := range tests {
		provider, model, thinking := parseModelPattern(tc.input)
		if provider != tc.provider {
			t.Errorf("input=%q: expected provider=%q, got=%q", tc.input, tc.provider, provider)
		}
		if model != tc.model {
			t.Errorf("input=%q: expected model=%q, got=%q", tc.input, tc.model, model)
		}
		if thinking != tc.thinking {
			t.Errorf("input=%q: expected thinking=%q, got=%q", tc.input, tc.thinking, thinking)
		}
	}
}

func TestRegister(t *testing.T) {
	r := NewModelRegistry()

	r.Register(ai.ModelInfo{
		ID:       "custom-model",
		Provider: ai.Provider("custom"),
		API:      ai.ApiOpenAICompletions,
	})

	model := r.Find("custom", "custom-model")
	if model == nil {
		t.Error("should find registered model")
	}
}

func TestProviderToAPI(t *testing.T) {
	tests := []struct {
		provider ai.Provider
		expected ai.Api
	}{
		{ai.ProviderAnthropic, ai.ApiAnthropicMessages},
		{ai.ProviderGoogle, ai.ApiGoogleGenerativeAI},
		{ai.ProviderOpenAI, ai.ApiOpenAIResponses},
	}

	for _, tc := range tests {
		result := providerToAPI(tc.provider)
		if result != tc.expected {
			t.Errorf("provider=%s: expected %s, got %s", tc.provider, tc.expected, result)
		}
	}
}
