package readtool

import (
	"context"
	"os"
	"testing"
)

func TestExecute_BasicRead(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "readtool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("hello world\nline 2\nline 3")
	tmpFile.Close()

	result, err := Execute(context.Background(), ReadToolInput{
		Path: tmpFile.Name(),
	}, "", &defaultReadOps{})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestExecute_MissingFile(t *testing.T) {
	_, err := Execute(context.Background(), ReadToolInput{
		Path: "/nonexistent/file.txt",
	}, "", &defaultReadOps{})
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestExecute_WithOffsetLimit(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "readtool_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("line 1\nline 2\nline 3\nline 4\nline 5")
	tmpFile.Close()

	result, err := Execute(context.Background(), ReadToolInput{
		Path:   tmpFile.Name(),
		Offset: 2,
		Limit:  2,
	}, "", &defaultReadOps{})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestDefaultReadOps_Access(t *testing.T) {
	ops := &defaultReadOps{}

	// Existing file
	tmpFile, err := os.CreateTemp("", "readops_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := ops.Access(tmpFile.Name()); err != nil {
		t.Errorf("Expected no error for existing file, got %v", err)
	}

	// Non-existing file
	if err := ops.Access("/nonexistent/file.txt"); err == nil {
		t.Error("Expected error for non-existing file")
	}
}

func TestDefaultReadOps_ReadFile(t *testing.T) {
	ops := &defaultReadOps{}

	tmpFile, err := os.CreateTemp("", "readops_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	data, err := ops.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test content" {
		t.Errorf("Expected 'test content', got %q", string(data))
	}
}

func TestReadToolInput_Fields(t *testing.T) {
	input := ReadToolInput{
		Path:   "/tmp/test.txt",
		Offset: 10,
		Limit:  100,
	}
	if input.Path != "/tmp/test.txt" {
		t.Error("Path mismatch")
	}
	if input.Offset != 10 || input.Limit != 100 {
		t.Error("Offset/Limit mismatch")
	}
}
