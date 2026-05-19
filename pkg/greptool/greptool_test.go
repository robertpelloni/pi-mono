package greptool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockGrepOps struct {
	readFileFn    func(path string) (string, error)
	isDirectoryFn func(path string) (bool, error)
}

func (m *mockGrepOps) ReadFile(path string) (string, error) {
	if m.readFileFn != nil {
		return m.readFileFn(path)
	}
	return "", nil
}

func (m *mockGrepOps) IsDirectory(path string) (bool, error) {
	if m.isDirectoryFn != nil {
		return m.isDirectoryFn(path)
	}
	return false, nil
}

func TestGrepToolInput_Fields(t *testing.T) {
	input := GrepToolInput{
		Pattern:    "test",
		Path:       "/tmp",
		Glob:       "*.go",
		IgnoreCase: true,
		Literal:    true,
		Context:    3,
		Limit:      50,
	}
	if input.Pattern != "test" {
		t.Error("Pattern mismatch")
	}
	if input.Glob != "*.go" {
		t.Error("Glob mismatch")
	}
}

func TestGrepToolResult_Fields(t *testing.T) {
	result := GrepToolResult{
		Content:          "matched line",
		MatchLimitReached: 10,
		LinesTruncated:   true,
	}
	if result.Content != "matched line" {
		t.Error("Content mismatch")
	}
}

func TestExecute_NativeSearch(t *testing.T) {
	dir, _ := os.MkdirTemp("", "grep_test")
	defer os.RemoveAll(dir)

	// Create test files
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world\nfoo bar\nhello again"), 0644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("no match here"), 0644)

	ops := &mockGrepOps{
		isDirectoryFn: func(path string) (bool, error) {
			return path == dir, nil
		},
		readFileFn: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}

	result, err := Execute(context.Background(), GrepToolInput{
		Pattern: "hello",
		Path:    dir,
	}, dir, ops)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("Expected match in output, got %q", result.Content)
	}
}

func TestExecute_FileSearch(t *testing.T) {
	dir, _ := os.MkdirTemp("", "grep_file_test")
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "search.txt")
	os.WriteFile(filePath, []byte("find this pattern\nother line\nfind again"), 0644)

	ops := &mockGrepOps{
		isDirectoryFn: func(path string) (bool, error) {
			return false, nil
		},
		readFileFn: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}

	result, err := Execute(context.Background(), GrepToolInput{
		Pattern: "find",
		Path:    filePath,
	}, dir, ops)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestExecute_IgnoreCase(t *testing.T) {
	dir, _ := os.MkdirTemp("", "grep_icase_test")
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "mixed.txt"), []byte("Hello World"), 0644)

	ops := &mockGrepOps{
		isDirectoryFn: func(path string) (bool, error) {
			return path == dir, nil
		},
		readFileFn: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}

	result, err := Execute(context.Background(), GrepToolInput{
		Pattern:    "hello",
		Path:       dir,
		IgnoreCase: true,
	}, dir, ops)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	_ = result
}
