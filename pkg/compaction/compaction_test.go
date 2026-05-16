package compaction

import (
	"context"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func makeMessages(count int) []ai.Message {
	var msgs []ai.Message
	for i := 0; i < count; i++ {
		msgs = append(msgs, ai.UserMessage{
			Content: []ai.Content{ai.TextContent{Text: "This is a test message with enough content to estimate tokens properly."}},
		})
		msgs = append(msgs, ai.AssistantMessage{
			Content:   []ai.Content{ai.TextContent{Text: "This is a response message with enough content to estimate tokens properly."}},
			API:       ai.ApiOpenAIResponses,
			Provider:   ai.ProviderOpenAI,
			Model:     "gpt-4o",
			StopReason: ai.StopReasonStop,
		})
	}
	return msgs
}

func TestShouldCompact(t *testing.T) {
	config := DefaultCompactionConfig()
	config.MaxTokens = 50 // Very low threshold for testing
	c := NewCompactor(config)

	msgs := makeMessages(10)
	if !c.ShouldCompact(msgs) {
		t.Error("messages should exceed low token threshold")
	}
}

func TestShouldNotCompact(t *testing.T) {
	config := DefaultCompactionConfig()
	config.MaxTokens = 100000 // High threshold
	c := NewCompactor(config)

	msgs := makeMessages(2)
	if c.ShouldCompact(msgs) {
		t.Error("few messages should not exceed high token threshold")
	}
}

func TestCompactSummarize(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 50,
		Strategy:  StrategySummarize,
		KeepLastN: 2,
	}
	c := NewCompactor(config)

	msgs := makeMessages(10)
	result, err := c.Compact(context.Background(), msgs)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) >= len(msgs) {
		t.Errorf("compacted messages should be fewer: before=%d after=%d", len(msgs), len(result))
	}

	// Should contain summary marker
	hasSummary := false
	for _, m := range result {
		if um, ok := m.(ai.UserMessage); ok {
			for _, c := range um.Content {
				if txt, ok := c.(ai.TextContent); ok && len(txt.Text) > 0 && (txt.Text[:1] == "[" || txt.Text[:8] == "Previous") {
					hasSummary = true
				}
			}
		}
	}
	if !hasSummary {
		t.Error("compacted messages should contain a summary")
	}
}

func TestCompactDrop(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 50,
		Strategy:  StrategyDrop,
		KeepLastN: 4,
	}
	c := NewCompactor(config)

	msgs := makeMessages(10)
	result, err := c.Compact(context.Background(), msgs)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) > config.KeepLastN {
		t.Errorf("drop strategy should keep only last N: got %d, expected <= %d", len(result), config.KeepLastN)
	}
}

func TestCompactTruncate(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 50,
		Strategy:  StrategyTruncate,
		KeepLastN: 2,
	}
	c := NewCompactor(config)

	msgs := makeMessages(10)
	result, err := c.Compact(context.Background(), msgs)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) >= len(msgs) {
		t.Errorf("truncated messages should be fewer")
	}
}

func TestCompactNotEnough(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 100000,
		Strategy:  StrategySummarize,
		KeepLastN: 6,
	}
	c := NewCompactor(config)

	msgs := makeMessages(2)
	result, err := c.Compact(context.Background(), msgs)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) != len(msgs) {
		t.Error("not enough messages — should return as-is")
	}
}

func TestCompactStats(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 50,
		Strategy:  StrategyDrop,
		KeepLastN: 2,
	}
	c := NewCompactor(config)

	msgs := makeMessages(10)
	c.Compact(context.Background(), msgs)

	stats := c.Stats()
	if stats.CompactionsCount != 1 {
		t.Errorf("expected 1 compaction, got %d", stats.CompactionsCount)
	}
	if stats.MessagesBefore <= stats.MessagesAfter {
		t.Error("before should be greater than after")
	}
}

func TestEstimateTokens(t *testing.T) {
	msgs := []ai.Message{
		ai.UserMessage{
			Content: []ai.Content{ai.TextContent{Text: "Hello world this is a test"}},
		},
	}
	tokens := estimateTokens(msgs)
	if tokens <= 0 {
		t.Error("token estimate should be positive")
	}
}

func TestCustomSummarizeFn(t *testing.T) {
	config := CompactionConfig{
		MaxTokens: 50,
		Strategy:  StrategySummarize,
		KeepLastN: 2,
		SummarizeFn: func(ctx context.Context, messages []ai.Message) (string, error) {
			return "Custom summary of conversation", nil
		},
	}
	c := NewCompactor(config)

	msgs := makeMessages(10)
	result, err := c.Compact(context.Background(), msgs)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that custom summary appears
	for _, m := range result {
		if um, ok := m.(ai.UserMessage); ok {
			for _, c := range um.Content {
				if txt, ok := c.(ai.TextContent); ok && txt.Text == "[Context Summary]\n\nCustom summary of conversation" {
					return // Success
				}
			}
		}
	}
	t.Error("expected custom summary in compacted messages")
}
