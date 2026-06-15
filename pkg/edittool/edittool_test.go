package edittool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/editdiff"
)

func TestEditTool_MultipleDisjointEdits(t *testing.T) {
	// Create a temp dir
	dir := t.TempDir()
	filePath := filepath.Join(dir, "multi.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Prepare edits: replace "line2\n" with "LINE2\n", "line4\n" with "LINE4\n"
	edits := []editdiff.Edit{
		{OldText: "line2\n", NewText: "LINE2\n"},
		{OldText: "line4\n", NewText: "LINE4\n"},
	}

	// Call ApplyEditsToNormalizedContent directly
	normalized := editdiff.NormalizeToLF(content)
	result, err := editdiff.ApplyEditsToNormalizedContent(normalized, edits, filePath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify final content matches expected
	expected := "line1\nLINE2\nline3\nLINE4\nline5\n"
	if result.NewContent != expected {
		t.Errorf("New content mismatch:\nGot:\n%q\nExpected:\n%q", result.NewContent, expected)
	}
}

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
