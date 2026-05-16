package compaction

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// CompactionStrategy defines how messages are summarized.
type CompactionStrategy string

const (
	StrategySummarize CompactionStrategy = "summarize"
	StrategyTruncate  CompactionStrategy = "truncate"
	StrategyDrop      CompactionStrategy = "drop"
)

// CompactionConfig controls how context compaction works.
type CompactionConfig struct {
	// MaxTokens is the target maximum token count for the compacted context.
	MaxTokens int
	// Strategy selects the compaction algorithm.
	Strategy CompactionStrategy
	// KeepLastN messages are never compacted (always kept verbatim).
	KeepLastN int
	// SummarizeFn is called to generate a summary of old messages.
	// If nil, a simple concatenation+truncation strategy is used.
	SummarizeFn func(ctx context.Context, messages []ai.Message) (string, error)
}

// DefaultCompactionConfig returns sensible defaults.
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		MaxTokens: 100000,
		Strategy:  StrategySummarize,
		KeepLastN: 6, // Keep the last 3 exchanges (3 user + 3 assistant)
	}
}

// Compactor manages context window compaction.
type Compactor struct {
	mu     sync.RWMutex
	config CompactionConfig
	stats  CompactionStats
}

// CompactionStats tracks compaction history.
type CompactionStats struct {
	CompactionsCount    int `json:"compactionsCount"`
	MessagesBefore      int `json:"messagesBefore"`
	MessagesAfter       int `json:"messagesAfter"`
	TokensEstimatedBefore int `json:"tokensEstimatedBefore"`
	TokensEstimatedAfter  int `json:"tokensEstimatedAfter"`
}

// NewCompactor creates a new compactor with the given config.
func NewCompactor(config CompactionConfig) *Compactor {
	return &Compactor{
		config: config,
	}
}

// ShouldCompact checks if the messages exceed the token limit.
func (c *Compactor) ShouldCompact(messages []ai.Message) bool {
	estimated := estimateTokens(messages)
	return estimated > c.config.MaxTokens
}

// Compact performs context compaction on the message list.
// It keeps the system prompt and last N messages, and summarizes the rest.
func (c *Compactor) Compact(ctx context.Context, messages []ai.Message) ([]ai.Message, error) {
	if len(messages) <= c.config.KeepLastN {
		return messages, nil // Not enough messages to compact
	}

	c.mu.Lock()
	c.stats.CompactionsCount++
	c.stats.MessagesBefore = len(messages)
	c.stats.TokensEstimatedBefore = estimateTokens(messages)
	c.mu.Unlock()

	// Split messages: old (to summarize) + recent (to keep)
	splitIdx := len(messages) - c.config.KeepLastN
	oldMessages := messages[:splitIdx]
	recentMessages := messages[splitIdx:]

	var compacted []ai.Message

	switch c.config.Strategy {
	case StrategySummarize:
		summary, err := c.summarizeMessages(ctx, oldMessages)
		if err != nil {
			// Fall back to truncation
			compacted = c.truncateMessages(oldMessages)
		} else {
			// Insert the summary as a user message
			summaryMsg := ai.UserMessage{
				Content: []ai.Content{
					ai.TextContent{Text: fmt.Sprintf("[Context Summary]\n\n%s", summary)},
				},
				Timestamp: messages[0].GetTimestamp(),
			}
			compacted = append([]ai.Message{summaryMsg}, recentMessages...)
		}

	case StrategyTruncate:
		compacted = c.truncateMessages(oldMessages)
		compacted = append(compacted, recentMessages...)

	case StrategyDrop:
		compacted = recentMessages

	default:
		compacted = recentMessages
	}

	c.mu.Lock()
	c.stats.MessagesAfter = len(compacted)
	c.stats.TokensEstimatedAfter = estimateTokens(compacted)
	c.mu.Unlock()

	return compacted, nil
}

// Stats returns the current compaction statistics.
func (c *Compactor) Stats() CompactionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// --- Internal ---

func (c *Compactor) summarizeMessages(ctx context.Context, messages []ai.Message) (string, error) {
	if c.config.SummarizeFn != nil {
		return c.config.SummarizeFn(ctx, messages)
	}

	// Default: create a simple chronological summary
	return defaultSummarize(messages), nil
}

// defaultSummarize creates a basic summary by extracting key information from messages.
func defaultSummarize(messages []ai.Message) string {
	var sb strings.Builder
	sb.WriteString("Previous conversation summary:\n\n")

	toolCalls := 0
	userMessages := 0
	assistantMessages := 0
	var keyActions []string

	for _, msg := range messages {
		switch m := msg.(type) {
		case ai.UserMessage:
			userMessages++
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					if len(keyActions) < 10 && len(txt.Text) > 0 {
						preview := txt.Text
						if len(preview) > 100 {
							preview = preview[:97] + "..."
						}
						keyActions = append(keyActions, fmt.Sprintf("User asked: %s", preview))
					}
				}
			}
		case ai.AssistantMessage:
			assistantMessages++
			for _, c := range m.Content {
				if tc, ok := c.(ai.ToolCall); ok {
					toolCalls++
					if len(keyActions) < 10 {
						keyActions = append(keyActions, fmt.Sprintf("Used tool: %s", tc.Name))
					}
				}
			}
		case ai.ToolResultMessage:
			// Tool results are summarized via their parent tool call
		}
	}

	sb.WriteString(fmt.Sprintf("- %d user messages, %d assistant messages, %d tool calls\n", userMessages, assistantMessages, toolCalls))
	sb.WriteString("\nKey actions:\n")
	for _, action := range keyActions {
		sb.WriteString(fmt.Sprintf("  - %s\n", action))
	}

	return sb.String()
}

// truncateMessages keeps only the first and last few messages from old context.
func (c *Compactor) truncateMessages(messages []ai.Message) []ai.Message {
	if len(messages) <= 4 {
		return messages
	}

	// Keep first 2 and last 2 messages
	var result []ai.Message
	result = append(result, messages[:2]...)
	result = append(result, ai.UserMessage{
		Content: []ai.Content{
			ai.TextContent{Text: fmt.Sprintf("[... %d messages truncated ...]", len(messages)-4)},
		},
	})
	result = append(result, messages[len(messages)-2:]...)
	return result
}

// estimateTokens provides a rough token estimate for a message list.
// Uses the heuristic of ~4 characters per token.
func estimateTokens(messages []ai.Message) int {
	total := 0
	for _, msg := range messages {
		switch m := msg.(type) {
		case ai.UserMessage:
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					total += len(txt.Text) / 4
				}
			}
		case ai.AssistantMessage:
			for _, c := range m.Content {
				switch ct := c.(type) {
				case ai.TextContent:
					total += len(ct.Text) / 4
				case ai.ThinkingContent:
					total += len(ct.Thinking) / 4
				case ai.ToolCall:
					argBytes, _ := fmt.Sprintf("%v", ct.Arguments), 0
					total += len(ct.Name)/4 + len(argBytes)/4
				}
			}
		case ai.ToolResultMessage:
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					total += len(txt.Text) / 4
				}
			}
		}
	}
	return total
}

// GetTimestamp is a helper to get the timestamp from any message.
// This should be on the Message interface, but we add it here for compatibility.
func GetTimestamp(msg ai.Message) int64 {
	switch m := msg.(type) {
	case ai.UserMessage:
		return m.Timestamp
	case ai.AssistantMessage:
		return m.Timestamp
	case ai.ToolResultMessage:
		return m.Timestamp
	default:
		return 0
	}
}
