package tools

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestReadTool(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	ioutil.WriteFile(filePath, []byte("hello read tool"), 0644)

	tool := ReadTool(tmpDir)
	res, err := tool.Execute(context.Background(), "call_1", map[string]any{"path": "test.txt"}, func(_ agent.AgentToolResult) {})
	if err != nil {
		t.Fatalf("ReadTool failed: %v", err)
	}

	content := res.Content[0].(ai.TextContent).Text
	if content != "hello read tool" {
		t.Errorf("Expected 'hello read tool', got '%s'", content)
	}
}

func TestBashTool_BlocksKillNode(t *testing.T) {
	tool := BashTool(".")

	_, err := tool.Execute(context.Background(), "call_2", map[string]any{"command": "pkill node"}, func(_ agent.AgentToolResult) {})
	if err == nil {
		t.Fatal("Expected BashTool to block 'pkill node' command, but got no error")
	}

	if err.Error() != "blocked: cannot taskkill node processes" {
		t.Errorf("Expected blocked error, got '%v'", err)
	}
}

func TestWriteTool(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := WriteTool(tmpDir)

	_, err = tool.Execute(context.Background(), "call_3", map[string]any{
		"path":    "newfile.txt",
		"content": "write test content",
	}, func(_ agent.AgentToolResult) {})
	if err != nil {
		t.Fatalf("WriteTool failed: %v", err)
	}

	contentBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(contentBytes) != "write test content" {
		t.Errorf("Expected 'write test content', got '%s'", string(contentBytes))
	}
}
