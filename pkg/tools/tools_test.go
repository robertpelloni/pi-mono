package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestCreateAllTools(t *testing.T) {
	toolList := CreateAllTools(".")
	if len(toolList) != 7 {
		t.Errorf("Expected 7 tools, got %d", len(toolList))
	}

	names := ToolNames()
	if len(names) != 7 {
		t.Errorf("Expected 7 tool names, got %d", len(names))
	}
}

func TestDefaultToolNames(t *testing.T) {
	names := DefaultToolNames()
	if len(names) != 4 {
		t.Errorf("Expected 4 default tool names, got %d", len(names))
	}
}

func TestCreateDefaultTools(t *testing.T) {
	toolList := CreateDefaultTools(".")
	if len(toolList) != 4 {
		t.Errorf("Expected 4 default tools, got %d", len(toolList))
	}

	expectedNames := map[string]bool{"read": true, "bash": true, "edit": true, "write": true}
	for _, tool := range toolList {
		if !expectedNames[tool.Name] {
			t.Errorf("Unexpected tool name: %s", tool.Name)
		}
	}
}

func TestReadToolIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello read tool"), 0644)

	toolList := CreateAllTools(tmpDir)
	var readTool *agent.AgentTool
	for i := range toolList {
		if toolList[i].Name == "read" {
			readTool = &toolList[i]
			break
		}
	}
	if readTool == nil {
		t.Fatal("read tool not found")
	}

	res, err := readTool.Execute(context.Background(), "call_1", map[string]any{"path": "test.txt"}, func(_ agent.AgentToolResult) {})
	if err != nil {
		t.Fatalf("ReadTool failed: %v", err)
	}

	content := res.Content[0].(ai.TextContent).Text
	if len(content) == 0 {
		t.Error("Expected non-empty content from read tool")
	}
}

func TestWriteToolIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	toolList := CreateAllTools(tmpDir)
	var writeTool *agent.AgentTool
	for i := range toolList {
		if toolList[i].Name == "write" {
			writeTool = &toolList[i]
			break
		}
	}
	if writeTool == nil {
		t.Fatal("write tool not found")
	}

	_, err = writeTool.Execute(context.Background(), "call_3", map[string]any{
		"path":    "newfile.txt",
		"content": "write test content",
	}, func(_ agent.AgentToolResult) {})
	if err != nil {
		t.Fatalf("WriteTool failed: %v", err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(contentBytes) != "write test content" {
		t.Errorf("Expected 'write test content', got '%s'", string(contentBytes))
	}
}
