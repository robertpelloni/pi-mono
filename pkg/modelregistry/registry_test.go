package modelregistry

import (
	"os"
	"testing"
)

func TestNewModelRegistry(t *testing.T) {
	registry := NewModelRegistry()
	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}

	providers := registry.GetAllProviders()
	if len(providers) < 3 {
		t.Errorf("Expected at least 3 providers, got %d", len(providers))
	}
}

func TestResolveProvider(t *testing.T) {
	registry := NewModelRegistry()

	tests := []struct {
		modelID     string
		expected    string
		shouldError bool
	}{
		{"gpt-4o", "openai", false},
		{"claude-sonnet-4-20250514", "anthropic", false},
		{"gemini-2.0-flash", "google", false},
		{"o3-mini", "openai", false},
		{"openai/gpt-4o", "openai", false},
		{"unknown-model", "", true},
	}

	for _, tt := range tests {
		provider, err := registry.ResolveProvider(tt.modelID)
		if tt.shouldError {
			if err == nil {
				t.Errorf("Expected error for model %s, got provider %s", tt.modelID, provider)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for model %s: %v", tt.modelID, err)
			}
			if provider != tt.expected {
				t.Errorf("Expected provider %s for model %s, got %s", tt.expected, tt.modelID, provider)
			}
		}
	}
}

func TestHasAPIKey(t *testing.T) {
	registry := NewModelRegistry()

	// Without env vars set
	if registry.HasAPIKey("openai") {
		// May be set in the test environment
		if os.Getenv("OPENAI_API_KEY") == "" {
			t.Error("Expected HasAPIKey to return false when no env var is set")
		}
	}

	// With env var set
	os.Setenv("TEST_API_KEY", "test-key")
	defer os.Unsetenv("TEST_API_KEY")
	registry.RegisterProvider(ProviderInfo{
		Name:    "test-provider",
		EnvVars: []string{"TEST_API_KEY"},
		Label:   "Test",
	})
	if !registry.HasAPIKey("test-provider") {
		t.Error("Expected HasAPIKey to return true when env var is set")
	}
}

func TestSearchModels(t *testing.T) {
	registry := NewModelRegistry()

	results := registry.SearchModels("gpt")
	if len(results) == 0 {
		t.Error("Expected to find GPT models")
	}

	results2 := registry.SearchModels("claude")
	if len(results2) == 0 {
		t.Error("Expected to find Claude models")
	}

	results3 := registry.SearchModels("xyznonexistent")
	if len(results3) != 0 {
		t.Error("Expected no results for nonexistent pattern")
	}
}

func TestFormatProviderStatus(t *testing.T) {
	registry := NewModelRegistry()
	status := registry.FormatProviderStatus()
	if len(status) == 0 {
		t.Error("Expected non-empty status")
	}
}
