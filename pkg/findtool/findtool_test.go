package findtool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type mockFindOps struct {
	existsFn func(path string) bool
	globFn   func(pattern string, cwd string, ignore []string, limit int) ([]string, error)
}

func (m *mockFindOps) Exists(path string) bool {
	if m.existsFn != nil {
		return m.existsFn(path)
	}
	return false
}

func (m *mockFindOps) Glob(pattern string, cwd string, ignore []string, limit int) ([]string, error) {
	if m.globFn != nil {
		return m.globFn(pattern, cwd, ignore, limit)
	}
	return nil, nil
}

func TestFindToolInput_Fields(t *testing.T) {
	input := FindToolInput{
		Pattern: "*.go",
		Path:    "/tmp",
		Limit:   50,
	}
	if input.Pattern != "*.go" {
		t.Error("Pattern mismatch")
	}
}

func TestFindToolResult_Fields(t *testing.T) {
	result := FindToolResult{
		Content:            "file1.go\nfile2.go",
		ResultLimitReached: 10,
	}
	if result.Content != "file1.go\nfile2.go" {
		t.Error("Content mismatch")
	}
}

func TestExecute_Basic(t *testing.T) {
	dir, _ := os.MkdirTemp("", "find_test")
	defer os.RemoveAll(dir)

	// Create some files
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("text"), 0644)

	ops := &mockFindOps{
		existsFn: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		globFn: func(pattern string, cwd string, ignore []string, limit int) ([]string, error) {
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				return nil, err
			}
			return matches, nil
		},
	}

	result, err := Execute(context.Background(), FindToolInput{
		Pattern: "*.go",
		Path:    dir,
	}, dir, ops)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestExecute_NoPattern(t *testing.T) {
	ops := &mockFindOps{}
	result, err := Execute(context.Background(), FindToolInput{
		Pattern: "",
	}, ".", ops)
	// Should handle empty pattern gracefully
	_ = result
	_ = err
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		match   bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.txt", false},
		{"test*", "testfile.go", true},
		{"test*", "other.go", false},
	}

	for _, tt := range tests {
		result := matchGlob(tt.pattern, tt.name)
		if result != tt.match {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, result, tt.match)
		}
	}
}
