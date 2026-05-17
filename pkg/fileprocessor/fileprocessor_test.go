package fileprocessor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildInitialMessage(t *testing.T) {
	tests := []struct {
		name         string
		stdinContent string
		fileText     string
		messages     []string
		expected     string
	}{
		{"empty", "", "", nil, ""},
		{"stdin only", "stdin text", "", nil, "stdin text"},
		{"file only", "", "file text", nil, "file text"},
		{"message only", "", "", []string{"msg text"}, "msg text"},
		{"all combined", "stdin", "file", []string{"msg"}, "stdinfilemsg"},
		{"multiple messages", "", "", []string{"first", "second"}, "first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildInitialMessage(tt.stdinContent, tt.fileText, tt.messages)
			if got != tt.expected {
				t.Errorf("BuildInitialMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessFileArgs_TextFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileproc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0644)

	result, err := ProcessFileArgs(tmpDir, []string{"test.txt"})
	if err != nil {
		t.Fatalf("ProcessFileArgs failed: %v", err)
	}

	if result.Text != "hello world" {
		t.Errorf("Expected 'hello world', got %q", result.Text)
	}
}

func TestProcessFileArgs_ImageFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileproc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a minimal PNG file (1x1 pixel)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG magic bytes
	testFile := filepath.Join(tmpDir, "test.png")
	os.WriteFile(testFile, pngData, 0644)

	result, err := ProcessFileArgs(tmpDir, []string{"test.png"})
	if err != nil {
		t.Fatalf("ProcessFileArgs failed: %v", err)
	}

	if len(result.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(result.Images))
	}
	if result.Images[0].MimeType != "image/png" {
		t.Errorf("Expected image/png, got %s", result.Images[0].MimeType)
	}
}

func TestProcessFileArgs_NonExistent(t *testing.T) {
	_, err := ProcessFileArgs("/tmp", []string{"nonexistent.txt"})
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestProcessFileArgs_Directory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileproc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files in the directory
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)

	result, err := ProcessFileArgs(filepath.Dir(tmpDir), []string{filepath.Base(tmpDir)})
	if err != nil {
		t.Fatalf("ProcessFileArgs failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty text from directory listing")
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte{}, ""},
		{[]byte("hello"), "aGVsbG8="},
		{[]byte("test"), "dGVzdA=="},
	}

	for _, tt := range tests {
		got := encodeBase64(tt.input)
		if got != tt.expected {
			t.Errorf("encodeBase64(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
