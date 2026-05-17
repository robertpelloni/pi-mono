package listmodels

import (
	"bytes"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{500, "500"},
		{1000, "1K"},
		{1500, "1.5K"},
		{2000, "2K"},
		{100000, "100K"},
		{128000, "128K"},
		{1000000, "1M"},
		{1500000, "1.5M"},
		{2000000, "2M"},
	}

	for _, tt := range tests {
		got := FormatTokenCount(tt.count)
		if got != tt.expected {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.count, got, tt.expected)
		}
	}
}

func TestListModels_Empty(t *testing.T) {
	var buf bytes.Buffer
	ListModels(nil, "", &buf)
	if buf.String() != "No models available. Set API keys in environment variables.\n" {
		t.Errorf("Unexpected output: %q", buf.String())
	}
}

func TestListModels_WithModels(t *testing.T) {
	models := []ai.ModelInfo{
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Provider:      ai.ProviderOpenAI,
			ContextWindow: 128000,
			MaxTokens:     16384,
			Reasoning:     false,
			Input:         []string{"text", "image"},
		},
		{
			ID:            "claude-sonnet-4-20250514",
			Name:          "Claude Sonnet 4",
			Provider:      ai.ProviderAnthropic,
			ContextWindow: 200000,
			MaxTokens:     16384,
			Reasoning:     true,
			Input:         []string{"text", "image"},
		},
	}

	var buf bytes.Buffer
	ListModels(models, "", &buf)

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("provider")) {
		t.Error("Expected header row with 'provider'")
	}
	if !bytes.Contains(buf.Bytes(), []byte("gpt-4o")) {
		t.Error("Expected model gpt-4o in output")
	}
}

func TestListModels_SearchFilter(t *testing.T) {
	models := []ai.ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", Provider: ai.ProviderOpenAI, ContextWindow: 128000, MaxTokens: 16384},
		{ID: "claude-sonnet-4", Name: "Claude Sonnet 4", Provider: ai.ProviderAnthropic, ContextWindow: 200000, MaxTokens: 16384},
	}

	var buf bytes.Buffer
	ListModels(models, "claude", &buf)

	if bytes.Contains(buf.Bytes(), []byte("gpt-4o")) {
		t.Error("gpt-4o should be filtered out when searching for 'claude'")
	}
	if !bytes.Contains(buf.Bytes(), []byte("claude-sonnet-4")) {
		t.Error("Expected claude-sonnet-4 in filtered output")
	}
}

func TestListModels_NoMatch(t *testing.T) {
	models := []ai.ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", Provider: ai.ProviderOpenAI, ContextWindow: 128000, MaxTokens: 16384},
	}

	var buf bytes.Buffer
	ListModels(models, "nonexistent", &buf)

	if !bytes.Contains(buf.Bytes(), []byte("No models matching")) {
		t.Error("Expected 'No models matching' message")
	}
}
