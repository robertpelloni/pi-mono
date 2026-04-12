package ai

import (
	"strings"
	"time"
)

// NormalizeToolCallIDFunc represents a function to normalize tool call IDs for cross-provider compatibility.
type NormalizeToolCallIDFunc func(id string, model ModelInfo, source AssistantMessage) string

// TransformMessages normalizes messages for cross-provider compatibility.
// It handles thinking block drops for cross-model handoffs, normalizes tool call IDs,
// and synthetic tool result insertion for orphaned tool calls.
func TransformMessages(
	messages []Message,
	model ModelInfo,
	normalizeToolCallID NormalizeToolCallIDFunc,
) []Message {
	toolCallIDMap := make(map[string]string)

	// First pass: transform messages
	var transformed []Message
	for _, msg := range messages {
		switch m := msg.(type) {
		case UserMessage:
			transformed = append(transformed, m)
		case ToolResultMessage:
			normalizedID, exists := toolCallIDMap[m.ToolCallID]
			if exists && normalizedID != m.ToolCallID {
				m.ToolCallID = normalizedID
			}
			transformed = append(transformed, m)
		case AssistantMessage:
			isSameModel := m.Provider == model.Provider &&
				m.API == model.API &&
				m.Model == model.ID

			var transformedContent []Content
			for _, block := range m.Content {
				switch b := block.(type) {
				case ThinkingContent:
					// Redacted thinking is opaque encrypted content, only valid for the same model.
					if b.Redacted != nil && *b.Redacted {
						if isSameModel {
							transformedContent = append(transformedContent, b)
						}
						continue
					}
					// For same model: keep thinking blocks with signatures (needed for replay)
					if isSameModel && b.ThinkingSignature != nil {
						transformedContent = append(transformedContent, b)
						continue
					}
					// Skip empty thinking blocks, convert others to plain text
					if strings.TrimSpace(b.Thinking) == "" {
						continue
					}
					if isSameModel {
						transformedContent = append(transformedContent, b)
					} else {
						transformedContent = append(transformedContent, TextContent{Text: b.Thinking})
					}
				case TextContent:
					if isSameModel {
						transformedContent = append(transformedContent, b)
					} else {
						// Drop signature if cross-model
						transformedContent = append(transformedContent, TextContent{Text: b.Text})
					}
				case ToolCall:
					normalizedToolCall := b
					if !isSameModel && b.ThoughtSignature != nil {
						normalizedToolCall.ThoughtSignature = nil
					}
					if !isSameModel && normalizeToolCallID != nil {
						normalizedID := normalizeToolCallID(b.ID, model, m)
						if normalizedID != b.ID {
							toolCallIDMap[b.ID] = normalizedID
							normalizedToolCall.ID = normalizedID
						}
					}
					transformedContent = append(transformedContent, normalizedToolCall)
				default:
					transformedContent = append(transformedContent, b)
				}
			}
			m.Content = transformedContent
			transformed = append(transformed, m)
		}
	}

	// Second pass: insert synthetic empty tool results for orphaned tool calls
	var result []Message
	var pendingToolCalls []ToolCall
	existingToolResultIDs := make(map[string]struct{})

	for _, msg := range transformed {
		switch m := msg.(type) {
		case AssistantMessage:
			// Insert synthetic results for previous assistant's orphaned calls
			if len(pendingToolCalls) > 0 {
				for _, tc := range pendingToolCalls {
					if _, exists := existingToolResultIDs[tc.ID]; !exists {
						result = append(result, ToolResultMessage{
							ToolCallID: tc.ID,
							ToolName:   tc.Name,
							Content:    []Content{TextContent{Text: "No result provided"}},
							IsError:    true,
							Timestamp:  time.Now().UnixMilli(),
						})
					}
				}
				pendingToolCalls = nil
				existingToolResultIDs = make(map[string]struct{})
			}

			// Skip errored/aborted assistant messages entirely to avoid API errors during replay
			if m.StopReason == StopReasonError || m.StopReason == StopReasonAborted {
				continue
			}

			// Track tool calls from this assistant message
			var currentToolCalls []ToolCall
			for _, b := range m.Content {
				if tc, ok := b.(ToolCall); ok {
					currentToolCalls = append(currentToolCalls, tc)
				}
			}
			if len(currentToolCalls) > 0 {
				pendingToolCalls = currentToolCalls
				existingToolResultIDs = make(map[string]struct{})
			}
			result = append(result, m)

		case ToolResultMessage:
			existingToolResultIDs[m.ToolCallID] = struct{}{}
			result = append(result, m)

		case UserMessage:
			// User message interrupts tool flow - insert synthetic results for orphaned calls
			if len(pendingToolCalls) > 0 {
				for _, tc := range pendingToolCalls {
					if _, exists := existingToolResultIDs[tc.ID]; !exists {
						result = append(result, ToolResultMessage{
							ToolCallID: tc.ID,
							ToolName:   tc.Name,
							Content:    []Content{TextContent{Text: "No result provided"}},
							IsError:    true,
							Timestamp:  time.Now().UnixMilli(),
						})
					}
				}
				pendingToolCalls = nil
				existingToolResultIDs = make(map[string]struct{})
			}
			result = append(result, m)
		}
	}

	// Final check for orphaned tool calls at the end of the message list
	if len(pendingToolCalls) > 0 {
		for _, tc := range pendingToolCalls {
			if _, exists := existingToolResultIDs[tc.ID]; !exists {
				result = append(result, ToolResultMessage{
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
					Content:    []Content{TextContent{Text: "No result provided"}},
					IsError:    true,
					Timestamp:  time.Now().UnixMilli(),
				})
			}
		}
	}

	return result
}
