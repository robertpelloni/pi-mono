//go:build integration
// +build integration

package ai

import (
	"context"
	"os"
	"testing"
)

func TestStreamAnthropic_Integration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	model := ModelInfo{
		ID:       "claude-3-haiku-20240307",
		Provider: ProviderAnthropic,
	}

	aiCtx := Context{
		Messages: []Message{
			UserMessage{
				Content: []Content{
					TextContent{Text: "Say 'integration-test-success'"},
				},
			},
		},
	}

	stream := StreamAnthropic(context.Background(), model, aiCtx, StreamOptions{})

	found := false
	for event := range stream {
		if event.Type == EventTextDelta && event.Delta != nil {
			if *event.Delta != "" {
				found = true
			}
		}
		if event.Type == EventError {
			t.Errorf("received error event: %v", event.Error.ErrorMessage)
		}
	}

	if !found {
		t.Error("no text content received from Anthropic")
	}
}

func TestStreamOpenAI_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	model := ModelInfo{
		ID:       "gpt-4o-mini",
		Provider: ProviderOpenAI,
	}

	aiCtx := Context{
		Messages: []Message{
			UserMessage{
				Content: []Content{
					TextContent{Text: "Say 'integration-test-success'"},
				},
			},
		},
	}

	stream := StreamOpenAIResponses(context.Background(), model, aiCtx, StreamOptions{})

	found := false
	for event := range stream {
		if event.Type == EventTextDelta && event.Delta != nil {
			if *event.Delta != "" {
				found = true
			}
		}
		if event.Type == EventError {
			t.Errorf("received error event: %v", event.Error.ErrorMessage)
		}
	}

	if !found {
		t.Error("no text content received from OpenAI")
	}
}
