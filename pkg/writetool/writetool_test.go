package writetool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
)

func TestCreateWriteTool_BasicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writetool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := CreateWriteTool(tmpDir)
	if tool.Name != "write" {
		t.Errorf("Expected tool name 'write', got %s", tool.Name)
	}
	if tool.Execute == nil {
		t.Error("Expected Execute function to be non-nil")
	}

	// Test writing a file
	result, err := tool.Execute(nil, "call_1", map[string]any{
		"path":    "test.txt",
		"content": "hello world",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("Expected no error in result")
	}

	// Verify the file was written
	data, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("Expected 'hello world', got %q", string(data))
	}
}

func TestCreateWriteTool_MissingPath(t *testing.T) {
	tool := CreateWriteTool("/tmp")
	_, err := tool.Execute(nil, "call_1", map[string]any{
		"content": "test",
	}, nil)
	if err == nil {
		t.Error("Expected error for missing path")
	}
}

func TestCreateWriteTool_AutoMkdir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writetool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := CreateWriteTool(tmpDir)

	// Write to a nested directory
	result, err := tool.Execute(nil, "call_1", map[string]any{
		"path":    "nested/dir/file.txt",
		"content": "nested content",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("Expected no error")
	}

	// Verify
	data, err := os.ReadFile(filepath.Join(tmpDir, "nested", "dir", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "nested content" {
		t.Errorf("Expected 'nested content', got %q", string(data))
	}
}

func TestCreateWriteTool_AbsolutePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writetool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := CreateWriteTool("/some/cwd")

	absPath := filepath.Join(tmpDir, "absolute_test.txt")
	_, err = tool.Execute(nil, "call_1", map[string]any{
		"path":    absPath,
		"content": "absolute content",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "absolute content" {
		t.Errorf("Expected 'absolute content', got %q", string(data))
	}
}

func TestCreateWriteTool_Parameters(t *testing.T) {
	tool := CreateWriteTool("/tmp")
	params, ok := tool.Parameters.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map parameters")
	}
	if params["type"] != "object" {
		t.Error("Expected object type")
	}
	if tool.PromptSnippet == "" {
		t.Error("Expected non-empty prompt snippet")
	}
}

func TestCreateWriteTool_ResultType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writetool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := CreateWriteTool(tmpDir)
	result, err := tool.Execute(nil, "call_1", map[string]any{
		"path":    "result_test.txt",
		"content": "test",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var _ agent.AgentToolResult = result
	if len(result.Content) == 0 {
		t.Error("Expected content in result")
	}
}
