package nativetools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
)

func TestNativeReadToolText(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	tool := NativeReadTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"path": "test.txt"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	// Content should contain text about the file
	_ = result // basic sanity check that no error occurred
}

func TestNativeReadToolWithOffset(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	tool := NativeReadTool(tmpDir)
	params := map[string]any{
		"path":   "test.txt",
		"offset": float64(3),
		"limit":  float64(2),
	}
	result, err := tool.Execute(context.Background(), "test-id", params, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
}

func TestNativeReadToolDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)

	tool := NativeReadTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"path": "."}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content for directory listing")
	}
}

func TestNativeReadToolNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NativeReadTool(tmpDir)
	_, err := tool.Execute(context.Background(), "test-id", map[string]any{"path": "nonexistent.txt"}, nil)

	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestNativeGlobTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.ts"), []byte("const x = 1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.go"), []byte("package util"), 0644)

	tool := NativeGlobTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"pattern": "*.go"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
}

func TestNativeGrepTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package util\n\nfunc Helper() {\n\treturn\n}\n"), 0644)

	tool := NativeGrepTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"pattern": "func"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
}

func TestNativeGrepToolLiteral(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello (world)\")\n}\n"), 0644)

	tool := NativeGrepTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{
		"pattern": "(world)",
		"literal": true,
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content for literal search")
	}
}

func TestNativeGrepToolIgnoreCase(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main\n\nFUNC helper() {}\n"), 0644)

	tool := NativeGrepTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{
		"pattern":    "func",
		"ignoreCase": true,
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content for case-insensitive search")
	}
}

func TestNativeGrepToolNoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main\n"), 0644)

	tool := NativeGrepTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"pattern": "nonexistent_pattern"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not error, just return "no matches"
	txt := extractText(result)
	if !strings.Contains(txt, "No matches") {
		t.Errorf("expected 'No matches', got: %s", txt)
	}
}

func TestNativeBashTool(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NativeBashTool(tmpDir)

	result, err := tool.Execute(context.Background(), "test-id", map[string]any{"command": "echo hello"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	txt := extractText(result)
	if !strings.Contains(txt, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", txt)
	}
}

func TestNativeBashToolTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NativeBashTool(tmpDir)

	// This should complete quickly
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{
		"command": "echo fast",
		"timeout": float64(5),
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	txt := extractText(result)
	if !strings.Contains(txt, "fast") {
		t.Errorf("expected 'fast' in output, got: %s", txt)
	}
}

func TestNativeLsTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	tool := NativeLsTool(tmpDir)
	result, err := tool.Execute(context.Background(), "test-id", map[string]any{}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	txt := extractText(result)
	if !strings.Contains(txt, "file1.txt") {
		t.Errorf("expected 'file1.txt' in listing, got: %s", txt)
	}
	if !strings.Contains(txt, "subdir") {
		t.Errorf("expected 'subdir' in listing, got: %s", txt)
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		path    string
		binary  bool
	}{
		{"test.exe", true},
		{"image.png", true},
		{"archive.zip", true},
		{"main.go", false},
		{"config.json", false},
		{"readme.md", false},
	}

	for _, tc := range tests {
		result := isBinaryFile(tc.path)
		if result != tc.binary {
			t.Errorf("isBinaryFile(%q) = %v, expected %v", tc.path, result, tc.binary)
		}
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.ts", false},
		{"test_*.go", "test_main.go", true},
		{"?.go", "a.go", true},
		{"?.go", "ab.go", false},
	}

	for _, tc := range tests {
		regex := globToRegex(tc.pattern)
		matched, _ := regexp.MatchString(regex, tc.path)
		if matched != tc.match {
			t.Errorf("globToRegex(%q) matching %q: got %v, expected %v (regex: %s)", tc.pattern, tc.path, matched, tc.match, regex)
		}
	}
}

// Helper to extract text content from a tool result
func extractText(result agent.AgentToolResult) string {
	var sb strings.Builder
	for _, c := range result.Content {
		sb.WriteString(fmt.Sprintf("%v", c))
	}
	return sb.String()
}
