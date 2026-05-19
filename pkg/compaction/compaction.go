package compaction

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// ---------------------------------------------------------------------------
// File Operation Tracking
// ---------------------------------------------------------------------------

// FileOperations tracks which files have been read, written, or edited during a session.
type FileOperations struct {
	Read    map[string]bool
	Written map[string]bool
	Edited  map[string]bool
}

// NewFileOps creates an empty FileOperations set.
func NewFileOps() FileOperations {
	return FileOperations{
		Read:    make(map[string]bool),
		Written: make(map[string]bool),
		Edited:  make(map[string]bool),
	}
}

// ExtractFileOpsFromMessage extracts file operations from tool calls in an assistant message.
func ExtractFileOpsFromMessage(msg ai.Message, fileOps FileOperations) {
	am, ok := msg.(ai.AssistantMessage)
	if !ok {
		return
	}
	for _, block := range am.Content {
		tc, ok := block.(ai.ToolCall)
		if !ok {
			continue
		}
		path, _ := tc.Arguments["path"].(string)
		if path == "" {
			path, _ = tc.Arguments["file_path"].(string)
		}
		if path == "" {
			continue
		}
		switch tc.Name {
		case "read":
			fileOps.Read[path] = true
		case "write":
			fileOps.Written[path] = true
		case "edit":
			fileOps.Edited[path] = true
		}
	}
}

// FileLists contains deduplicated lists of read-only and modified files.
type FileLists struct {
	ReadFiles     []string
	ModifiedFiles []string
}

// ComputeFileLists produces sorted, deduplicated file lists from file operations.
func ComputeFileLists(fileOps FileOperations) FileLists {
	modified := make(map[string]bool)
	for f := range fileOps.Edited {
		modified[f] = true
	}
	for f := range fileOps.Written {
		modified[f] = true
	}

	var readOnly []string
	for f := range fileOps.Read {
		if !modified[f] {
			readOnly = append(readOnly, f)
		}
	}
	sort.Strings(readOnly)

	var modifiedFiles []string
	for f := range modified {
		modifiedFiles = append(modifiedFiles, f)
	}
	sort.Strings(modifiedFiles)

	return FileLists{ReadFiles: readOnly, ModifiedFiles: modifiedFiles}
}

// FormatFileOperations formats file operations as XML tags for a summary.
func FormatFileOperations(readFiles, modifiedFiles []string) string {
	var sections []string
	if len(readFiles) > 0 {
		sections = append(sections, "<read-files>\n"+strings.Join(readFiles, "\n")+"\n</read-files>")
	}
	if len(modifiedFiles) > 0 {
		sections = append(sections, "<modified-files>\n"+strings.Join(modifiedFiles, "\n")+"\n</modified-files>")
	}
	if len(sections) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(sections, "\n\n")
}

// ---------------------------------------------------------------------------
// Message Serialization
// ---------------------------------------------------------------------------

const toolResultMaxChars = 2000

// SerializeConversation serializes messages to text for summarization.
// This prevents the model from treating it as a conversation to continue.
func SerializeConversation(messages []ai.Message) string {
	var parts []string
	for _, msg := range messages {
		switch m := msg.(type) {
		case ai.UserMessage:
			var content string
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					content += txt.Text
				}
			}
			if content != "" {
				parts = append(parts, "[User]: "+content)
			}
		case ai.AssistantMessage:
			var textParts, thinkingParts, toolCalls []string
			for _, block := range m.Content {
				switch b := block.(type) {
				case ai.TextContent:
					textParts = append(textParts, b.Text)
				case ai.ThinkingContent:
					thinkingParts = append(thinkingParts, b.Thinking)
				case ai.ToolCall:
					var argParts []string
					for k, v := range b.Arguments {
						argParts = append(argParts, fmt.Sprintf("%s=%v", k, v))
					}
					toolCalls = append(toolCalls, fmt.Sprintf("%s(%s)", b.Name, strings.Join(argParts, ", ")))
				}
			}
			if len(thinkingParts) > 0 {
				parts = append(parts, "[Assistant thinking]: "+strings.Join(thinkingParts, "\n"))
			}
			if len(textParts) > 0 {
				parts = append(parts, "[Assistant]: "+strings.Join(textParts, "\n"))
			}
			if len(toolCalls) > 0 {
				parts = append(parts, "[Assistant tool calls]: "+strings.Join(toolCalls, "; "))
			}
		case ai.ToolResultMessage:
			var content string
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					content += txt.Text
				}
			}
			if content != "" {
				parts = append(parts, "[Tool result]: "+truncateForSummary(content, toolResultMaxChars))
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func truncateForSummary(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	truncatedChars := len(text) - maxChars
	return fmt.Sprintf("%s\n\n[... %d more characters truncated]", text[:maxChars], truncatedChars)
}

// ---------------------------------------------------------------------------
// Compaction Strategy
// ---------------------------------------------------------------------------

// CompactionStrategy defines how messages are summarized.
type CompactionStrategy string

const (
	StrategySummarize CompactionStrategy = "summarize"
	StrategyTruncate  CompactionStrategy = "truncate"
	StrategyDrop      CompactionStrategy = "drop"
)

// CompactionSettings controls how context compaction works.
type CompactionSettings struct {
	Enabled          bool `json:"enabled"`
	ReserveTokens    int  `json:"reserveTokens"`
	KeepRecentTokens int  `json:"keepRecentTokens"`
}

// DefaultCompactionSettings returns sensible defaults.
func DefaultCompactionSettings() CompactionSettings {
	return CompactionSettings{
		Enabled:          true,
		ReserveTokens:    16384,
		KeepRecentTokens: 20000,
	}
}

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
	// Settings for threshold-based compaction.
	Settings CompactionSettings
}

// DefaultCompactionConfig returns sensible defaults.
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		MaxTokens:   100000,
		Strategy:    StrategySummarize,
		KeepLastN:   6,
		Settings:    DefaultCompactionSettings(),
	}
}

// CompactionStats tracks compaction history.
type CompactionStats struct {
	CompactionsCount      int `json:"compactionsCount"`
	MessagesBefore        int `json:"messagesBefore"`
	MessagesAfter         int `json:"messagesAfter"`
	TokensEstimatedBefore int `json:"tokensEstimatedBefore"`
	TokensEstimatedAfter  int `json:"tokensEstimatedAfter"`
}

// CompactionResult holds the result of a compaction operation.
type CompactionResult struct {
	Summary         string      `json:"summary"`
	FirstKeptEntryID string     `json:"firstKeptEntryId"`
	TokensBefore    int         `json:"tokensBefore"`
	Details         *CompactionDetails `json:"details,omitempty"`
}

// CompactionDetails stores file tracking info in a compaction entry.
type CompactionDetails struct {
	ReadFiles     []string `json:"readFiles"`
	ModifiedFiles []string `json:"modifiedFiles"`
}

// Compactor manages context window compaction.
type Compactor struct {
	mu     sync.RWMutex
	config CompactionConfig
	stats  CompactionStats
}

// NewCompactor creates a new compactor with the given config.
func NewCompactor(config CompactionConfig) *Compactor {
	return &Compactor{
		config: config,
	}
}

// ShouldCompact checks if the messages exceed the token limit.
func (c *Compactor) ShouldCompact(messages []ai.Message) bool {
	estimated := EstimateTokensFromMessages(messages)
	return estimated > c.config.MaxTokens
}

// ShouldCompactWithWindow checks if compaction should trigger based on context usage.
func (c *Compactor) ShouldCompactWithWindow(contextTokens, contextWindow int) bool {
	if !c.config.Settings.Enabled {
		return false
	}
	return contextTokens > contextWindow-c.config.Settings.ReserveTokens
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
	c.stats.TokensEstimatedBefore = EstimateTokensFromMessages(messages)
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
			compacted = append(compacted, recentMessages...)
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
	c.stats.TokensEstimatedAfter = EstimateTokensFromMessages(compacted)
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
		sb.WriteString(fmt.Sprintf(" - %s\n", action))
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

// ---------------------------------------------------------------------------
// Token Estimation
// ---------------------------------------------------------------------------

// EstimateTokensFromMessages provides a rough token estimate for a message list.
// Uses the heuristic of ~4 characters per token.
func EstimateTokensFromMessages(messages []ai.Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateTokens(msg)
	}
	return total
}

// EstimateTokens provides a rough token estimate for a single message.
// Uses the heuristic of ~4 characters per token.
func EstimateTokens(msg ai.Message) int {
	chars := 0
	switch m := msg.(type) {
	case ai.UserMessage:
		for _, c := range m.Content {
			if txt, ok := c.(ai.TextContent); ok {
				chars += len(txt.Text)
			}
		}
	case ai.AssistantMessage:
		for _, c := range m.Content {
			switch ct := c.(type) {
			case ai.TextContent:
				chars += len(ct.Text)
			case ai.ThinkingContent:
				chars += len(ct.Thinking)
			case ai.ToolCall:
				argBytes := fmt.Sprintf("%v", ct.Arguments)
				chars += len(ct.Name) + len(argBytes)
			}
		}
	case ai.ToolResultMessage:
		for _, c := range m.Content {
			if txt, ok := c.(ai.TextContent); ok {
				chars += len(txt.Text)
			}
		}
	}
	if chars == 0 {
		return 0
	}
	return (chars + 3) / 4 // ceil(chars/4)
}

// GetTimestamp is a helper to get the timestamp from any message.
func GetTimestamp(msg ai.Message) int64 {
	return msg.GetTimestamp()
}

// ---------------------------------------------------------------------------
// Context Usage Estimation
// ---------------------------------------------------------------------------

// ContextUsageEstimate provides a detailed breakdown of context token usage.
type ContextUsageEstimate struct {
	Tokens          int  `json:"tokens"`
	UsageTokens     int  `json:"usageTokens"`
	TrailingTokens  int  `json:"trailingTokens"`
	LastUsageIndex  *int `json:"lastUsageIndex"`
}

// EstimateContextTokens estimates context tokens from messages.
// Uses the last assistant usage when available, falling back to heuristic estimation.
func EstimateContextTokens(messages []ai.Message) ContextUsageEstimate {
	// Find the last non-aborted, non-error assistant message with usage
	var usageIdx *int
	var lastUsage ai.Usage
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(ai.AssistantMessage); ok {
			if am.StopReason != ai.StopReasonAborted && am.StopReason != ai.StopReasonError && am.Usage.Input > 0 {
				idx := i
				usageIdx = &idx
				lastUsage = am.Usage
				break
			}
		}
	}

	if usageIdx == nil {
		estimated := EstimateTokensFromMessages(messages)
		return ContextUsageEstimate{
			Tokens:         estimated,
			UsageTokens:    0,
			TrailingTokens: estimated,
			LastUsageIndex: nil,
		}
	}

	usageTokens := lastUsage.Input + lastUsage.Output + lastUsage.CacheRead + lastUsage.CacheWrite
	var trailingTokens int
	for i := *usageIdx + 1; i < len(messages); i++ {
		trailingTokens += EstimateTokens(messages[i])
	}

	return ContextUsageEstimate{
		Tokens:         usageTokens + trailingTokens,
		UsageTokens:    usageTokens,
		TrailingTokens: trailingTokens,
		LastUsageIndex: usageIdx,
	}
}

// ShouldCompact checks if compaction should trigger based on context usage.
func ShouldCompact(contextTokens, contextWindow int, settings CompactionSettings) bool {
	if !settings.Enabled {
		return false
	}
	return contextTokens > contextWindow-settings.ReserveTokens
}

// ---------------------------------------------------------------------------
// LLM-Based Summarization
// ---------------------------------------------------------------------------

// SummarizationSystemPrompt is the system prompt used for summarization.
const SummarizationSystemPrompt = `You are a context summarization assistant. Your task is to read a conversation between a user and an AI coding assistant, then produce a structured summary following the exact format specified. Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`

// InitialSummarizationPrompt is the prompt for the first compaction summary.
const InitialSummarizationPrompt = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work. Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]
### In Progress
- [ ] [Current work]
### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

// UpdateSummarizationPrompt is the prompt for updating an existing summary.
const UpdateSummarizationPrompt = `The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags. Update the existing structured summary with new information.

RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]
### In Progress
- [ ] [Current work - update based on progress]
### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

// TurnPrefixSummarizationPrompt is used when splitting a turn during compaction.
const TurnPrefixSummarizationPrompt = `This is the PREFIX of a turn that was too large to keep. The SUFFIX (recent work) is retained. Summarize the prefix to provide context for the retained suffix:

## Original Request
[What did the user ask for in this turn?]

## Early Progress
- [Key decisions and work done in the prefix]

## Context for Suffix
- [Information needed to understand the retained recent work]

Be concise. Focus on what's needed to understand the kept suffix.`

// GenerateSummaryRequest contains the data needed for LLM-based summarization.
type GenerateSummaryRequest struct {
	Messages         []ai.Message
	ReserveTokens    int
	PreviousSummary  *string
	CustomInstructions *string
	Model            ai.ModelInfo
	APIKey           string
	Headers          map[string]string
}

// GenerateSummaryResult holds the result of summarization.
type GenerateSummaryResult struct {
	Summary   string
	ReadFiles []string
	ModifiedFiles []string
}

// PrepareSummarizationPrompt builds the user prompt for summarization.
func PrepareSummarizationPrompt(messages []ai.Message, previousSummary *string, customInstructions *string) string {
	conversationText := SerializeConversation(messages)

	var basePrompt string
	if previousSummary != nil && *previousSummary != "" {
		basePrompt = UpdateSummarizationPrompt
	} else {
		basePrompt = InitialSummarizationPrompt
	}

	if customInstructions != nil && *customInstructions != "" {
		basePrompt = basePrompt + "\n\nAdditional focus: " + *customInstructions
	}

	var sb strings.Builder
	sb.WriteString("<conversation>\n")
	sb.WriteString(conversationText)
	sb.WriteString("\n</conversation>\n\n")

	if previousSummary != nil && *previousSummary != "" {
		sb.WriteString("<previous-summary>\n")
		sb.WriteString(*previousSummary)
		sb.WriteString("\n</previous-summary>\n\n")
	}

	sb.WriteString(basePrompt)
	return sb.String()
}

// GenerateDefaultSummary creates a summary without using an LLM.
// This is used as a fallback when no LLM is available for summarization.
func GenerateDefaultSummary(messages []ai.Message, previousSummary *string, customInstructions *string, fileOps FileOperations) GenerateSummaryResult {
	if len(messages) == 0 {
		if previousSummary != nil {
			return GenerateSummaryResult{Summary: *previousSummary}
		}
		return GenerateSummaryResult{Summary: "No prior history."}
	}

	summary := defaultSummarize(messages)

	if previousSummary != nil && *previousSummary != "" {
		summary = fmt.Sprintf("%s\n\n---\n\n**Updated context:**\n\n%s", *previousSummary, summary)
	}

	if customInstructions != nil && *customInstructions != "" {
		summary = summary + "\n\n**Additional focus:** " + *customInstructions
	}

	lists := ComputeFileLists(fileOps)
	summary += FormatFileOperations(lists.ReadFiles, lists.ModifiedFiles)

	return GenerateSummaryResult{
		Summary:       summary,
		ReadFiles:     lists.ReadFiles,
		ModifiedFiles: lists.ModifiedFiles,
	}
}

// ---------------------------------------------------------------------------
// LLM-Based Summarization Integration
// ---------------------------------------------------------------------------

// LLMSummarizationConfig provides the configuration for calling an LLM
// to generate compaction summaries.
type LLMSummarizationConfig struct {
	// StreamFunc is the AI provider streaming function.
	StreamFunc ai.StreamFunction
	// Model is the model info to use for summarization.
	Model ai.ModelInfo
	// APIKey for the provider.
	APIKey string
	// Headers for the API request.
	Headers map[string]string
	// MaxTokens to reserve for the response.
	MaxTokens int
}

// CreateLLMSummarizeFn creates a SummarizeFn that uses an LLM for summarization.
// If the LLM call fails, it falls back to GenerateDefaultSummary.
func CreateLLMSummarizeFn(config LLMSummarizationConfig) func(ctx context.Context, messages []ai.Message) (string, error) {
	return func(ctx context.Context, messages []ai.Message) (string, error) {
		// Try LLM-based summarization
		summary, err := callLLMForSummary(ctx, config, messages, nil)
		if err != nil {
			// Fall back to default summary
			result := GenerateDefaultSummary(messages, nil, nil, NewFileOps())
			return result.Summary, nil
		}
		return summary, nil
	}
}

// CreateLLMSummarizeFnWithPrevious creates a SummarizeFn that uses an LLM
// and includes a previous summary for updating.
func CreateLLMSummarizeFnWithPrevious(config LLMSummarizationConfig) func(ctx context.Context, messages []ai.Message, previousSummary *string) (string, error) {
	return func(ctx context.Context, messages []ai.Message, previousSummary *string) (string, error) {
		summary, err := callLLMForSummary(ctx, config, messages, previousSummary)
		if err != nil {
			result := GenerateDefaultSummary(messages, previousSummary, nil, NewFileOps())
			return result.Summary, nil
		}
		return summary, nil
	}
}

// callLLMForSummary sends a summarization request to the LLM.
func callLLMForSummary(ctx context.Context, config LLMSummarizationConfig, messages []ai.Message, previousSummary *string) (string, error) {
	if config.StreamFunc == nil {
		return "", fmt.Errorf("no stream function available for LLM summarization")
	}

	// Build the summarization prompt
	userPrompt := PrepareSummarizationPrompt(messages, previousSummary, nil)

	// Create the AI context with system + user messages
	systemPrompt := SummarizationSystemPrompt
	aiCtx := ai.Context{
		SystemPrompt: &systemPrompt,
		Messages: []ai.Message{
			ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: userPrompt}},
				Timestamp: time.Now().UnixMilli(),
			},
		},
		Tools: []ai.Tool{}, // No tools for summarization
	}

	// Set up streaming options
	options := ai.SimpleStreamOptions{
		StreamOptions: ai.StreamOptions{
			Headers:   config.Headers,
			MaxTokens: config.MaxTokens,
		},
	}

	// Call the LLM
	stream := config.StreamFunc(ctx, config.Model, aiCtx, options)

	// Collect the response
	var summaryBuilder strings.Builder
	for event := range stream {
		if event.Type == ai.EventTextDelta && event.Delta != nil {
			summaryBuilder.WriteString(*event.Delta)
		}
		if event.Error != nil {
			errMsg := "LLM summarization failed"
			if event.Error.ErrorMessage != nil {
				errMsg = *event.Error.ErrorMessage
			}
			return "", fmt.Errorf("%s", errMsg)
		}
	}

	summary := summaryBuilder.String()
	if summary == "" {
		return "", fmt.Errorf("LLM returned empty summary")
	}

	return summary, nil
}
