package messages

import (
	"fmt"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

const (
	// CompactionSummaryPrefix is prepended to compaction summaries.
	CompactionSummaryPrefix = "The conversation history before this point was compacted into the following summary: <summary>\n"
	// CompactionSummarySuffix is appended to compaction summaries.
	CompactionSummarySuffix = "\n</summary>"
	// BranchSummaryPrefix is prepended to branch summaries.
	BranchSummaryPrefix = "The following is a summary of a branch that this conversation came back from: <summary>\n"
	// BranchSummarySuffix is appended to branch summaries.
	BranchSummarySuffix = "\n</summary>"
)

// BashExecutionMessage represents a bash execution in the conversation.
type BashExecutionMessage struct {
	Role              string `json:"role"`              // "bashExecution"
	Command           string `json:"command"`
	Output            string `json:"output"`
	ExitCode          int    `json:"exitCode"`
	Cancelled         bool   `json:"cancelled"`
	Truncated         bool   `json:"truncated"`
	FullOutputPath    string `json:"fullOutputPath,omitempty"`
	Timestamp         int64  `json:"timestamp"`
	ExcludeFromContext bool  `json:"excludeFromContext,omitempty"` // If true, excluded from LLM context (!! prefix)
}

// CustomMessage represents an extension-injected message.
type CustomMessage struct {
	Role       string      `json:"role"`       // "custom"
	CustomType string      `json:"customType"`
	Content    string      `json:"content"`
	Display    bool        `json:"display"`
	Details    interface{} `json:"details,omitempty"`
	Timestamp  int64       `json:"timestamp"`
}

// BranchSummaryMessage represents a branch summary.
type BranchSummaryMessage struct {
	Role      string `json:"role"` // "branchSummary"
	Summary   string `json:"summary"`
	FromID    string `json:"fromId"`
	Timestamp int64  `json:"timestamp"`
}

// CompactionSummaryMessage represents a compaction summary.
type CompactionSummaryMessage struct {
	Role        string `json:"role"` // "compactionSummary"
	Summary     string `json:"summary"`
	TokensBefore int   `json:"tokensBefore"`
	Timestamp   int64  `json:"timestamp"`
}

// CreateBranchSummaryMessage creates a user message wrapping a branch summary.
func CreateBranchSummaryMessage(summary, fromID string) ai.UserMessage {
	text := BranchSummaryPrefix + summary + BranchSummarySuffix
	return ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}
}

// CreateCompactionSummaryMessage creates a user message wrapping a compaction summary.
func CreateCompactionSummaryMessage(summary string, tokensBefore int) ai.UserMessage {
	text := CompactionSummaryPrefix + summary + CompactionSummarySuffix
	return ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}
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

// ConvertToLlm transforms AgentMessages (including custom types) to LLM-compatible Messages.
// This handles bashExecution, custom, branchSummary, and compactionSummary message types.
func ConvertToLlm(messages []ai.Message) []ai.Message {
	var result []ai.Message

	for _, m := range messages {
		switch m.GetRole() {
		case ai.RoleUser, ai.RoleAssistant, ai.RoleTool:
			result = append(result, m)
		default:
			// Custom message types would be converted to user messages here
			// For now, include them as-is
			result = append(result, m)
		}
	}

	return result
}

// FormatBashOutput formats bash output for display, including truncation and error info.
func FormatBashOutput(output string, exitCode int, cancelled, truncated bool, fullOutputPath string) string {
	var parts []string

	if output != "" {
		parts = append(parts, output)
	}

	if cancelled {
		parts = append(parts, "\n\nCommand aborted")
	} else if exitCode != 0 {
		parts = append(parts, fmt.Sprintf("\n\nCommand exited with code %d", exitCode))
	}

	if truncated && fullOutputPath != "" {
		parts = append(parts, fmt.Sprintf("\n\n[Output truncated. Full output: %s]", fullOutputPath))
	}

	return strings.Join(parts, "")
}
