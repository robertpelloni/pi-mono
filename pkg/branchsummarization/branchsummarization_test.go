package branchsummarization

import (
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
)

func TestGetMessageFromEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    SessionEntry
		wantNil  bool
		wantRole string
	}{
		{
			name: "message entry",
			entry: SessionEntry{
				Type: "message",
				Message: ai.UserMessage{
					Content:   []ai.Content{ai.TextContent{Text: "hello"}},
					Timestamp: 1000,
				},
				Timestamp: 1000,
			},
			wantNil:  false,
			wantRole: "user",
		},
		{
			name: "branch_summary entry",
			entry: SessionEntry{
				Type:      "branch_summary",
				Summary:   "Previous work summary",
				Timestamp: 1000,
			},
			wantNil:  false,
			wantRole: "user",
		},
		{
			name: "compaction entry",
			entry: SessionEntry{
				Type:      "compaction",
				Summary:   "Compacted context",
				Timestamp: 1000,
			},
			wantNil:  false,
			wantRole: "user",
		},
		{
			name: "thinking_level_change entry",
			entry: SessionEntry{
				Type:      "thinking_level_change",
				Timestamp: 1000,
			},
			wantNil: true,
		},
		{
			name: "model_change entry",
			entry: SessionEntry{
				Type:      "model_change",
				Timestamp: 1000,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetMessageFromEntry(tt.entry)
			if tt.wantNil {
				if msg != nil {
					t.Error("Expected nil message")
				}
			} else {
				if msg == nil {
					t.Fatal("Expected non-nil message")
				}
				if msg.(ai.UserMessage).Content == nil && msg.(ai.AssistantMessage).Content == nil {
					t.Error("Expected message with content")
				}
			}
		})
	}
}

func TestGetMessageFromEntryForCompaction(t *testing.T) {
	// Compaction entries should return nil
	entry := SessionEntry{
		Type:      "compaction",
		Summary:   "old summary",
		Timestamp: 1000,
	}
	msg := GetMessageFromEntryForCompaction(entry)
	if msg != nil {
		t.Error("Expected nil for compaction entry")
	}

	// Message entries should return the message
	msgEntry := SessionEntry{
		Type: "message",
		Message: ai.UserMessage{
			Content:   []ai.Content{ai.TextContent{Text: "hello"}},
			Timestamp: 1000,
		},
		Timestamp: 1000,
	}
	msg = GetMessageFromEntryForCompaction(msgEntry)
	if msg == nil {
		t.Error("Expected non-nil for message entry")
	}
}

func TestPrepareBranchEntries(t *testing.T) {
	entries := []SessionEntry{
		{ID: "1", Type: "message", Message: ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "First message"}}, Timestamp: 1}, Timestamp: 1},
		{ID: "2", Type: "message", Message: ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: "First response"}}, StopReason: ai.StopReasonStop, Timestamp: 2}, Timestamp: 2},
		{ID: "3", Type: "message", Message: ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "Second message"}}, Timestamp: 3}, Timestamp: 3},
		{ID: "4", Type: "message", Message: ai.AssistantMessage{
			Content: []ai.Content{
				ai.ToolCall{Name: "read", Arguments: map[string]any{"path": "/tmp/test.go"}},
			},
			StopReason: ai.StopReasonStop,
			Timestamp:  4,
		}, Timestamp: 4},
	}

	prep := PrepareBranchEntries(entries, 0) // No budget limit

	if len(prep.Messages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(prep.Messages))
	}
	if !prep.FileOps.Read["/tmp/test.go"] {
		t.Error("Expected read file to be tracked")
	}
}

func TestPrepareBranchEntries_WithBudget(t *testing.T) {
	entries := []SessionEntry{
		{ID: "1", Type: "message", Message: ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: strings.Repeat("hello ", 100)}}, Timestamp: 1}, Timestamp: 1},
		{ID: "2", Type: "message", Message: ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: strings.Repeat("response ", 100)}}, StopReason: ai.StopReasonStop, Timestamp: 2}, Timestamp: 2},
		{ID: "3", Type: "message", Message: ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "short msg"}}, Timestamp: 3}, Timestamp: 3},
	}

	prep := PrepareBranchEntries(entries, 50) // Very small budget

	// With a tiny budget, only the last message(s) should be included
	if len(prep.Messages) >= 3 {
		t.Errorf("Expected fewer messages with budget, got %d", len(prep.Messages))
	}
}

func TestGenerateBranchSummaryText(t *testing.T) {
	entries := []SessionEntry{
		{ID: "1", Type: "message", Message: ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "What does this code do?"}}, Timestamp: 1}, Timestamp: 1},
		{ID: "2", Type: "message", Message: ai.AssistantMessage{
			Content: []ai.Content{
				ai.TextContent{Text: "Let me check"},
				ai.ToolCall{Name: "read", Arguments: map[string]any{"path": "/tmp/main.go"}},
			},
			StopReason: ai.StopReasonStop,
			Timestamp:  2,
		}, Timestamp: 2},
	}

	result := GenerateBranchSummaryText(entries, 0)

	if result.Summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !strings.Contains(result.Summary, "branch") {
		t.Error("Expected branch summary preamble")
	}
	if len(result.ReadFiles) == 0 {
		t.Error("Expected read files to be tracked")
	}
}

func TestGenerateBranchSummaryText_EmptyEntries(t *testing.T) {
	result := GenerateBranchSummaryText(nil, 0)
	if result.Summary != "No content to summarize" {
		t.Errorf("Expected 'No content to summarize', got %q", result.Summary)
	}
}

func TestCollectEntriesForBranchSummary(t *testing.T) {
	// Build a simple tree: root -> A -> B (old leaf), root -> C (target)
	entries := map[string]*SessionEntry{
		"root": {ID: "root", ParentID: "", Type: "message"},
		"A":    {ID: "A", ParentID: "root", Type: "message"},
		"B":    {ID: "B", ParentID: "A", Type: "message"},
		"C":    {ID: "C", ParentID: "root", Type: "message"},
	}

	getEntry := func(id string) *SessionEntry { return entries[id] }
	getBranch := func(id string) []SessionEntry {
		var branch []SessionEntry
		current := id
		for current != "" {
			if e, ok := entries[current]; ok {
				branch = append(branch, *e)
				current = e.ParentID
			} else {
				break
			}
		}
		// Reverse
		for i, j := 0, len(branch)-1; i < j; i, j = i+1, j-1 {
			branch[i], branch[j] = branch[j], branch[i]
		}
		return branch
	}

	// Navigate from B to C — common ancestor is "root"
	result := CollectEntriesForBranchSummary(getEntry, getBranch, "B", "C")

	if len(result) != 2 {
		t.Errorf("Expected 2 entries (A, B), got %d", len(result))
	}

	// No old leaf — should return empty
	result = CollectEntriesForBranchSummary(getEntry, getBranch, "", "C")
	if len(result) != 0 {
		t.Error("Expected empty result for no old leaf")
	}
}

func TestConvertToLlm(t *testing.T) {
	messages := []ai.Message{
		ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "hi"}}, Timestamp: 1},
		ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: "hello"}}, StopReason: ai.StopReasonStop, Timestamp: 2},
		ai.ToolResultMessage{Content: []ai.Content{ai.TextContent{Text: "result"}}, ToolCallID: "c1", Timestamp: 3},
	}

	result := ConvertToLlm(messages)
	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}
}

func TestCompactionSummaryDelimiters(t *testing.T) {
	if !strings.Contains(CompactionSummaryPrefix, "compacted") {
		t.Error("CompactionSummaryPrefix should mention compacted")
	}
	if !strings.Contains(BranchSummaryPrefix, "branch") {
		t.Error("BranchSummaryPrefix should mention branch")
	}
}

func TestFileOpsIntegration(t *testing.T) {
	// Test that file operations are properly extracted and tracked
	entries := []SessionEntry{
		{ID: "1", Type: "message", Message: ai.AssistantMessage{
			Content: []ai.Content{
				ai.ToolCall{Name: "read", Arguments: map[string]any{"path": "/tmp/a.go"}},
				ai.ToolCall{Name: "edit", Arguments: map[string]any{"file_path": "/tmp/b.go"}},
			},
			StopReason: ai.StopReasonStop,
			Timestamp:  1,
		}, Timestamp: 1},
		{ID: "2", Type: "message", Message: ai.AssistantMessage{
			Content: []ai.Content{
				ai.ToolCall{Name: "write", Arguments: map[string]any{"path": "/tmp/c.go"}},
			},
			StopReason: ai.StopReasonStop,
			Timestamp:  2,
		}, Timestamp: 2},
	}

	prep := PrepareBranchEntries(entries, 0)

	lists := compaction.ComputeFileLists(prep.FileOps)
	if len(lists.ReadFiles) != 1 || lists.ReadFiles[0] != "/tmp/a.go" {
		t.Errorf("Expected read-only [/tmp/a.go], got %v", lists.ReadFiles)
	}
	if len(lists.ModifiedFiles) != 2 {
		t.Errorf("Expected 2 modified files, got %v", lists.ModifiedFiles)
	}
}
