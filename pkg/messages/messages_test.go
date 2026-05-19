package messages

import (
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestBashExecutionToText(t *testing.T) {
	tests := []struct {
		name     string
		msg      BashExecutionMessage
		contains []string
	}{
		{
			name: "basic execution",
			msg: BashExecutionMessage{
				Command:  "ls -la",
				Output:   "file1.txt\nfile2.txt",
				ExitCode: 0,
			},
			contains: []string{"ls -la", "file1.txt"},
		},
		{
			name: "no output",
			msg: BashExecutionMessage{
				Command:  "true",
				Output:   "",
				ExitCode: 0,
			},
			contains: []string{"(no output)"},
		},
		{
			name: "non-zero exit code",
			msg: BashExecutionMessage{
				Command:  "false",
				Output:   "error",
				ExitCode: 1,
			},
			contains: []string{"exited with code 1"},
		},
		{
			name: "cancelled command",
			msg: BashExecutionMessage{
				Command:   "sleep 100",
				Output:    "",
				Cancelled: true,
			},
			contains: []string{"cancelled"},
		},
		{
			name: "truncated output",
			msg: BashExecutionMessage{
				Command:       "cat bigfile",
				Output:        "partial output",
				Truncated:     true,
				FullOutputPath: "/tmp/full-output.txt",
			},
			contains: []string{"truncated", "/tmp/full-output.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := BashExecutionToText(tt.msg)
			for _, s := range tt.contains {
				if !strings.Contains(text, s) {
					t.Errorf("Expected text to contain %q, got: %s", s, text)
				}
			}
		})
	}
}

func TestCreateBranchSummaryMessage(t *testing.T) {
	msg := CreateBranchSummaryMessage("Branch explored different approach", "entry-123")
	if msg.GetRole() != ai.RoleUser {
		t.Errorf("Expected user role, got %s", msg.GetRole())
	}
	txt, ok := msg.Content[0].(ai.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if !strings.Contains(txt.Text, "branch") {
		t.Error("Expected branch summary prefix")
	}
	if !strings.Contains(txt.Text, "Branch explored different approach") {
		t.Error("Expected summary content")
	}
}

func TestCreateCompactionSummaryMessage(t *testing.T) {
	msg := CreateCompactionSummaryMessage("Context was compacted", 50000)
	if msg.GetRole() != ai.RoleUser {
		t.Errorf("Expected user role, got %s", msg.GetRole())
	}
	txt, ok := msg.Content[0].(ai.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if !strings.Contains(txt.Text, "compacted") {
		t.Error("Expected compaction summary prefix")
	}
}

func TestConvertToLlm(t *testing.T) {
	messages := []ai.Message{
		ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "hi"}}, Timestamp: 1},
		ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: "hello"}}, StopReason: ai.StopReasonStop, Timestamp: 2},
	}

	result := ConvertToLlm(messages)
	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}
}

func TestFormatBashOutput(t *testing.T) {
	result := FormatBashOutput("test output", 0, false, false, "")
	if !strings.Contains(result, "test output") {
		t.Error("Expected output in result")
	}

	// With exit code
	result = FormatBashOutput("error", 1, false, false, "")
	if !strings.Contains(result, "code 1") {
		t.Error("Expected exit code in result")
	}

	// Cancelled
	result = FormatBashOutput("", 0, true, false, "")
	if !strings.Contains(result, "aborted") {
		t.Error("Expected aborted in result")
	}

	// Truncated
	result = FormatBashOutput("partial", 0, false, true, "/tmp/full.txt")
	if !strings.Contains(result, "truncated") {
		t.Error("Expected truncation notice")
	}
}

func TestSummaryDelimiters(t *testing.T) {
	if !strings.Contains(CompactionSummaryPrefix, "compacted") {
		t.Error("CompactionSummaryPrefix should mention compacted")
	}
	if !strings.Contains(CompactionSummarySuffix, "</summary>") {
		t.Error("CompactionSummarySuffix should close summary tag")
	}
	if !strings.Contains(BranchSummaryPrefix, "branch") {
		t.Error("BranchSummaryPrefix should mention branch")
	}
	if !strings.Contains(BranchSummarySuffix, "</summary>") {
		t.Error("BranchSummarySuffix should close summary tag")
	}
}
