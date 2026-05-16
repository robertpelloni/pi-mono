package messages

import (
	"fmt"

	"github.com/badlogic/pi-mono/pkg/ai"
)

const (
	CompactionSummaryPrefix = "The conversation history before this point was compacted into the following summary:\n<summary>\n"
	CompactionSummarySuffix = "\n</summary>"
	BranchSummaryPrefix     = "The following is a summary of a branch that this conversation came back from:\n<summary>\n"
	BranchSummarySuffix     = "\n</summary>"
)

// BashExecutionMessage represents a bash command executed via the ! command.
type BashExecutionMessage struct {
	Role               string `json:"role"` // "bashExecution"
	Command            string `json:"command"`
	Output             string `json:"output"`
	ExitCode           int    `json:"exitCode"`
	Cancelled          bool   `json:"cancelled"`
	Truncated          bool   `json:"truncated"`
	FullOutputPath     string `json:"fullOutputPath,omitempty"`
	Timestamp          int64  `json:"timestamp"`
	ExcludeFromContext bool   `json:"excludeFromContext,omitempty"`
}

// CustomMessage represents an extension-injected message.
type CustomMessage struct {
	Role       string      `json:"role"` // "custom"
	CustomType string      `json:"customType"`
	Content    any         `json:"content"` // string or []Content
	Display    bool        `json:"display"`
	Details    any         `json:"details,omitempty"`
	Timestamp  int64       `json:"timestamp"`
}

// BranchSummaryMessage represents a summary from a returned-from branch.
type BranchSummaryMessage struct {
	Role      string `json:"role"` // "branchSummary"
	Summary   string `json:"summary"`
	FromID    string `json:"fromId"`
	Timestamp int64  `json:"timestamp"`
}

// CompactionSummaryMessage represents a summary from context compaction.
type CompactionSummaryMessage struct {
	Role        string `json:"role"` // "compactionSummary"
	Summary     string `json:"summary"`
	TokensBefore int   `json:"tokensBefore"`
	Timestamp   int64  `json:"timestamp"`
}

// BashExecutionToText converts a BashExecutionMessage to user message text for LLM context.
func BashExecutionToText(msg BashExecutionMessage) string {
	text := fmt.Sprintf("Ran `%s`\n", msg.Command)
	if msg.Output != "" {
		text += fmt.Sprintf("```\n%s\n```", msg.Output)
	} else {
		text += "(no output)"
	}
	if msg.Cancelled {
		text += "\n\n(command cancelled)"
	} else if msg.ExitCode != 0 {
		text += fmt.Sprintf("\n\nCommand exited with code %d", msg.ExitCode)
	}
	if msg.Truncated && msg.FullOutputPath != "" {
		text += fmt.Sprintf("\n\n[Output truncated. Full output: %s]", msg.FullOutputPath)
	}
	return text
}

// CreateBranchSummaryMessage creates a branch summary message.
func CreateBranchSummaryMessage(summary, fromID string, timestamp int64) BranchSummaryMessage {
	return BranchSummaryMessage{
		Role:      "branchSummary",
		Summary:   summary,
		FromID:    fromID,
		Timestamp: timestamp,
	}
}

// CreateCompactionSummaryMessage creates a compaction summary message.
func CreateCompactionSummaryMessage(summary string, tokensBefore int, timestamp int64) CompactionSummaryMessage {
	return CompactionSummaryMessage{
		Role:         "compactionSummary",
		Summary:      summary,
		TokensBefore: tokensBefore,
		Timestamp:    timestamp,
	}
}

// ConvertCustomToLlm converts custom message types to LLM-compatible ai.Messages.
func ConvertCustomToLlm(messages []any) []ai.Message {
	var result []ai.Message
	for _, m := range messages {
		switch msg := m.(type) {
		case BashExecutionMessage:
			if msg.ExcludeFromContext {
				continue
			}
			result = append(result, ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: BashExecutionToText(msg)}},
				Timestamp: msg.Timestamp,
			})
		case CustomMessage:
			var content []ai.Content
			switch c := msg.Content.(type) {
			case string:
				content = []ai.Content{ai.TextContent{Text: c}}
			case []ai.Content:
				content = c
			default:
				content = []ai.Content{ai.TextContent{Text: fmt.Sprintf("%v", c)}}
			}
			result = append(result, ai.UserMessage{
				Content:   content,
				Timestamp: msg.Timestamp,
			})
		case BranchSummaryMessage:
			result = append(result, ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: BranchSummaryPrefix + msg.Summary + BranchSummarySuffix}},
				Timestamp: msg.Timestamp,
			})
		case CompactionSummaryMessage:
			result = append(result, ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: CompactionSummaryPrefix + msg.Summary + CompactionSummarySuffix}},
				Timestamp: msg.Timestamp,
			})
		case ai.UserMessage, ai.AssistantMessage, ai.ToolResultMessage:
			result = append(result, m.(ai.Message))
		}
	}
	return result
}
