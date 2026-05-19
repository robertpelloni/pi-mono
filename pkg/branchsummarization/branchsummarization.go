package branchsummarization

import (
	"fmt"
	"sort"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
)

// ---------------------------------------------------------------------------
// Branch Summary Types
// ---------------------------------------------------------------------------

// BranchSummaryResult holds the result of a branch summarization.
type BranchSummaryResult struct {
	Summary       string   `json:"summary,omitempty"`
	ReadFiles     []string `json:"readFiles,omitempty"`
	ModifiedFiles []string `json:"modifiedFiles,omitempty"`
	Aborted       bool     `json:"aborted,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// BranchSummaryDetails stores file tracking info for a branch summary entry.
type BranchSummaryDetails struct {
	ReadFiles     []string `json:"readFiles"`
	ModifiedFiles []string `json:"modifiedFiles"`
}

// BranchPreparation holds the prepared data for branch summarization.
type BranchPreparation struct {
	Messages    []ai.Message              `json:"messages"`
	FileOps     compaction.FileOperations `json:"-"`
	TotalTokens int                       `json:"totalTokens"`
}

// GenerateBranchSummaryOptions configures branch summary generation.
type GenerateBranchSummaryOptions struct {
	Model            ai.ModelInfo
	APIKey           string
	Headers          map[string]string
	ReserveTokens    int
	CustomInstructions string
	ReplaceInstructions  bool
}

// ---------------------------------------------------------------------------
// Entry to Message Conversion
// ---------------------------------------------------------------------------

// SessionEntry represents a simplified session entry for branch summarization.
type SessionEntry struct {
	ID        string `json:"id"`
	ParentID  string `json:"parentId"`
	Type      string `json:"type"`
	Message   ai.Message `json:"message,omitempty"`
	Summary   string `json:"summary,omitempty"`
	FromID    string `json:"fromId,omitempty"`
	FromHook  bool   `json:"fromHook,omitempty"`
	Details   any    `json:"details,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// GetMessageFromEntry extracts an AgentMessage from a session entry.
// Returns nil for entries that don't contribute to LLM context.
func GetMessageFromEntry(entry SessionEntry) ai.Message {
	switch entry.Type {
	case "message":
		return entry.Message
	case "branch_summary":
		return ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: BranchSummaryPrefix + entry.Summary + BranchSummarySuffix},
			},
			Timestamp: entry.Timestamp,
		}
	case "compaction":
		return ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: CompactionSummaryPrefix + entry.Summary + CompactionSummarySuffix},
			},
			Timestamp: entry.Timestamp,
		}
	case "custom_message":
		return ai.UserMessage{
			Content:   []ai.Content{ai.TextContent{Text: fmt.Sprintf("[%s]", entry.Type)}},
			Timestamp: entry.Timestamp,
		}
	default:
		return nil
	}
}

// GetMessageFromEntryForCompaction is like GetMessageFromEntry but skips compaction entries.
func GetMessageFromEntryForCompaction(entry SessionEntry) ai.Message {
	if entry.Type == "compaction" {
		return nil
	}
	return GetMessageFromEntry(entry)
}

// ---------------------------------------------------------------------------
// Branch Preparation
// ---------------------------------------------------------------------------

// PrepareBranchEntries prepares entries for summarization with a token budget.
// Walks from newest to oldest, adding messages until hitting the budget.
// Also collects file operations from tool calls and existing summaries.
func PrepareBranchEntries(entries []SessionEntry, tokenBudget int) BranchPreparation {
	var messages []ai.Message
	fileOps := compaction.NewFileOps()
	totalTokens := 0

	// First pass: collect file ops from ALL entries (even beyond token budget)
	for _, entry := range entries {
		if entry.Type == "branch_summary" && !entry.FromHook {
			if details, ok := entry.Details.(BranchSummaryDetails); ok {
				for _, f := range details.ReadFiles {
					fileOps.Read[f] = true
				}
				for _, f := range details.ModifiedFiles {
					fileOps.Edited[f] = true
				}
			}
		}
	}

	// Second pass: walk from newest to oldest, adding messages until budget
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		msg := GetMessageFromEntry(entry)
		if msg == nil {
			continue
		}

		// Extract file ops from assistant messages (tool calls)
		compaction.ExtractFileOpsFromMessage(msg, fileOps)

		tokens := compaction.EstimateTokens(msg)

		// Check budget before adding
		if tokenBudget > 0 && totalTokens+tokens > tokenBudget {
			// If this is a summary entry, try to fit it anyway
			if entry.Type == "compaction" || entry.Type == "branch_summary" {
				if float64(totalTokens) < float64(tokenBudget)*0.9 {
					messages = append([]ai.Message{msg}, messages...)
					totalTokens += tokens
				}
			}
			break
		}

		messages = append([]ai.Message{msg}, messages...)
		totalTokens += tokens
	}

	return BranchPreparation{
		Messages:    messages,
		FileOps:     fileOps,
		TotalTokens: totalTokens,
	}
}

// ---------------------------------------------------------------------------
// Summary Generation
// ---------------------------------------------------------------------------

const BranchSummaryPreamble = "The user explored a different conversation branch before returning here. Summary of that exploration:\n\n"

const BranchSummaryPrompt = `Create a structured summary of this conversation branch for context when returning later.
Use this EXACT format:

## Goal
[What was the user trying to accomplish in this branch?]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]
### In Progress
- [ ] [Work that was started but not finished]
### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [What should happen next to continue this work]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

// Compaction/branch summary delimiters used when converting to LLM messages.
const (
	CompactionSummaryPrefix = "The conversation history before this point was compacted into the following summary:\n<summary>\n"
	CompactionSummarySuffix = "\n</summary>"
	BranchSummaryPrefix     = "The following is a summary of a branch that this conversation came back from:\n<summary>\n"
	BranchSummarySuffix     = "\n</summary>"
)

// GenerateBranchSummaryText generates a text-based branch summary without LLM.
// Used as a fallback when no LLM is available.
func GenerateBranchSummaryText(entries []SessionEntry, tokenBudget int) BranchSummaryResult {
	prep := PrepareBranchEntries(entries, tokenBudget)

	if len(prep.Messages) == 0 {
		return BranchSummaryResult{Summary: "No content to summarize"}
	}

	// Default: create a simple chronological summary
	var sb strings.Builder
	sb.WriteString(BranchSummaryPreamble)

	userMsgs := 0
	assistantMsgs := 0
	toolCalls := 0
	var keyActions []string

	for _, msg := range prep.Messages {
		switch m := msg.(type) {
		case ai.UserMessage:
			userMsgs++
			for _, c := range m.Content {
				if txt, ok := c.(ai.TextContent); ok {
					if len(keyActions) < 10 && len(txt.Text) > 0 {
						preview := txt.Text
						if len(preview) > 100 {
							preview = preview[:97] + "..."
						}
						keyActions = append(keyActions, fmt.Sprintf("User: %s", preview))
					}
				}
			}
		case ai.AssistantMessage:
			assistantMsgs++
			for _, c := range m.Content {
				if tc, ok := c.(ai.ToolCall); ok {
					toolCalls++
					if len(keyActions) < 10 {
						keyActions = append(keyActions, fmt.Sprintf("Tool: %s", tc.Name))
					}
				}
			}
		}
	}

	sb.WriteString(fmt.Sprintf("## Goal\nExplored branch with %d user messages, %d assistant messages, %d tool calls\n\n", userMsgs, assistantMsgs, toolCalls))
	sb.WriteString("## Progress\n")
	for _, action := range keyActions {
		sb.WriteString(fmt.Sprintf("- %s\n", action))
	}

	// Add file operations
	lists := compaction.ComputeFileLists(prep.FileOps)
	summary := sb.String() + compaction.FormatFileOperations(lists.ReadFiles, lists.ModifiedFiles)

	return BranchSummaryResult{
		Summary:       summary,
		ReadFiles:     lists.ReadFiles,
		ModifiedFiles: lists.ModifiedFiles,
	}
}

// CollectEntriesForBranchSummary collects entries to summarize when navigating
// from oldLeafId to targetId. Returns entries in chronological order.
func CollectEntriesForBranchSummary(
	getEntry func(id string) *SessionEntry,
	getBranch func(id string) []SessionEntry,
	oldLeafID string,
	targetID string,
) []SessionEntry {
	if oldLeafID == "" {
		return nil
	}

	// Get the old branch path
	oldPath := make(map[string]bool)
	for _, entry := range getBranch(oldLeafID) {
		oldPath[entry.ID] = true
	}

	// Find common ancestor (deepest node on both paths)
	var commonAncestorID string
	targetPath := getBranch(targetID)
	for i := len(targetPath) - 1; i >= 0; i-- {
		if oldPath[targetPath[i].ID] {
			commonAncestorID = targetPath[i].ID
			break
		}
	}

	// Collect entries from old leaf back to common ancestor
	var entries []SessionEntry
	current := oldLeafID
	for current != "" && current != commonAncestorID {
		entry := getEntry(current)
		if entry == nil {
			break
		}
		entries = append(entries, *entry)
		current = entry.ParentID
	}

	// Reverse to get chronological order
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

// ConvertToLlm converts custom message types to LLM-compatible messages.
// This is used for summarization and prompt construction.
func ConvertToLlm(messages []ai.Message) []ai.Message {
	var result []ai.Message
	for _, msg := range messages {
		switch m := msg.(type) {
		case ai.UserMessage, ai.AssistantMessage, ai.ToolResultMessage:
			result = append(result, m)
		default:
			// Skip unknown types
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// SortStrings returns a sorted copy of the input slice.
func SortStrings(s []string) []string {
	result := make([]string, len(s))
	copy(result, s)
	sort.Strings(result)
	return result
}
