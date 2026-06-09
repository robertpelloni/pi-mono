package modelresolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/auth"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		modelID  string
		expected string
	}{
		{"claude-sonnet-4-20250514", "anthropic"},
		{"gpt-4o", "openai"},
		{"gemini-2.5-pro", "google"},
		{"grok-4", "xai"},
		{"devstral-medium", "mistral"},
		{"unknown-model", "openai"},
	}

	for _, tt := range tests {
		got := DetectProvider(tt.modelID)
		if got != tt.expected {
			t.Errorf("DetectProvider(%q) = %q, want %q", tt.modelID, got, tt.expected)
		}
	}
}

func TestLoadModelsJSON_NoFile(t *testing.T) {
	models, overrides, err := LoadModelsJSON("/nonexistent/models.json", nil)
	if err != nil {
		t.Fatalf("Expected no error for missing file, got: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("Expected no models, got %d", len(models))
	}
	if len(overrides) != 0 {
		t.Errorf("Expected no overrides, got %d", len(overrides))
	}
}

func TestLoadModelsJSON_ValidFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsJSON := `{
		"providers": {
			"my-provider": {
				"baseUrl": "https://api.my-provider.com/v1",
				"apiKey": "my-key",
				"api": "openai-completions",
				"models": [
					{
						"id": "my-model-1",
						"name": "My Model 1",
						"reasoning": true,
						"contextWindow": 128000,
						"maxTokens": 8192
					}
				]
			}
		}
	}`

	path := filepath.Join(tmpDir, "models.json")
	os.WriteFile(path, []byte(modelsJSON), 0644)

	models, overrides, err := LoadModelsJSON(path, nil)
	if err != nil {
		t.Fatalf("LoadModelsJSON failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}

	if models[0].ID != "my-model-1" {
		t.Errorf("Expected model ID my-model-1, got %s", models[0].ID)
	}

	if models[0].Provider != ai.Provider("my-provider") {
		t.Errorf("Expected provider my-provider, got %s", models[0].Provider)
	}

	if overrides["my-provider"].BaseURL != "https://api.my-provider.com/v1" {
		t.Errorf("Expected baseUrl override, got %s", overrides["my-provider"].BaseURL)
	}
}

func TestModelRegistry_GetAvailable(t *testing.T) {
	registry := NewModelRegistryWithDefaults()

	// Set an API key so some models are available
	os.Setenv("OPENAI_API_KEY", "test")
	defer os.Unsetenv("OPENAI_API_KEY")

	authStorage := auth.InMemoryAuthStorage(nil)
	available := registry.GetAvailable(authStorage)
	if len(available) == 0 {
		t.Error("Expected available models with OpenAI API key")
	}
}

func TestModelRegistry_LoadFromModelsJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "models_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelsJSON := `{
		"providers": {
			"my-custom": {
				"baseUrl": "https://api.custom.com/v1",
				"apiKey": "custom-key",
				"api": "openai-completions",
				"models": [
					{
						"id": "custom-1",
						"name": "Custom 1",
						"contextWindow": 64000,
						"maxTokens": 4096
					}
				]
			}
		}
	}`

	path := filepath.Join(tmpDir, "models.json")
	os.WriteFile(path, []byte(modelsJSON), 0644)

	registry := NewModelRegistryWithDefaults()
	authStorage := auth.InMemoryAuthStorage(nil)

	err = registry.LoadFromModelsJSON(path, authStorage)
	if err != nil {
		t.Fatalf("LoadFromModelsJSON failed: %v", err)
	}

	model := registry.Find("my-custom", "custom-1")
	if model == nil {
		t.Fatal("Expected to find custom-1 model")
	}
	if model.ContextWindow != 64000 {
		t.Errorf("Expected contextWindow 64000, got %d", model.ContextWindow)
	}
}

func TestFindInitialModel(t *testing.T) {
	registry := NewModelRegistryWithDefaults()
	authStorage := auth.InMemoryAuthStorage(nil)

	// Set env var so a model is available
	os.Setenv("OPENAI_API_KEY", "sk-test")
	defer os.Unsetenv("OPENAI_API_KEY")

	options := FindInitialModelOptions{
		CLIProvider:   "",
		CLIModel:      "",
		IsContinuing:  false,
		ModelRegistry: registry,
		AuthStorage:   authStorage,
	}

	sm, fallbackMsg := FindInitialModel(options)
	if sm == nil {
		t.Fatal("Expected a model to be found")
	}
	if fallbackMsg != "" {
		t.Errorf("Expected no fallback message, got: %s", fallbackMsg)
	}
}

func TestGetModelsJSONPath(t *testing.T) {
	path := GetModelsJSONPath("/home/user/.pi")
	expected := filepath.Join("/home/user/.pi", "models.json")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestDefaultModelPerProvider(t *testing.T) {
	if DefaultModelPerProvider["anthropic"] == "" {
		t.Error("Expected default model for anthropic")
	}
	if DefaultModelPerProvider["openai"] == "" {
		t.Error("Expected default model for openai")
	}
}
