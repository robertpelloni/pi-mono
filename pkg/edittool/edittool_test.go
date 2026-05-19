package edittool

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/editdiff"
)

func TestCreateEditTool(t *testing.T) {
	tool := CreateEditTool(".")
	if tool.Name != "edit" {
		t.Errorf("Expected tool name 'edit', got %q", tool.Name)
	}
	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestEditToolDetails_Fields(t *testing.T) {
	details := EditToolDetails{
		Diff:             "--- a/file\n+++ b/file",
		FirstChangedLine: 10,
	}
	if details.Diff == "" {
		t.Error("Diff should not be empty")
	}
	if details.FirstChangedLine != 10 {
		t.Error("FirstChangedLine mismatch")
	}
}

func TestComputeEditsDiff_InvalidPath(t *testing.T) {
	_, err := ComputeEditsDiff("/nonexistent/file.txt", []editdiff.Edit{
		{OldText: "a", NewText: "b"},
	}, ".")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestToolIsAgentTool(t *testing.T) {
	tool := CreateEditTool(".")
	// Verify it satisfies the interface
	var _ agent.AgentTool = tool
}
