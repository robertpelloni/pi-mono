package export

import (
	"bytes"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestAnsiToHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"with ANSI color", "\x1b[31mred text\x1b[0m", "red text"},
		{"with ANSI bold", "\x1b[1mbold\x1b[0m text", "bold text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnsiToHTML(tt.input)
			if got != tt.want {
				t.Errorf("AnsiToHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteHTML_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteHTML(nil, &buf, ExportHTMLOptions{Title: "Test"})
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("<!DOCTYPE html>")) {
		t.Error("Expected HTML document")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Test")) {
		t.Error("Expected title in output")
	}
}

func TestWriteHTML_WithMessages(t *testing.T) {
	messages := []ai.Message{
		ai.UserMessage{
			Content: []ai.Content{ai.TextContent{Text: "Hello, world!"}},
		},
	}

	var buf bytes.Buffer
	err := WriteHTML(messages, &buf, ExportHTMLOptions{Title: "Test"})
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Hello, world!")) {
		t.Errorf("Expected message content in output, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("You")) {
		t.Error("Expected user role label")
	}
}

func TestExportHTML_InvalidPath(t *testing.T) {
	err := ExportHTML(nil, "/nonexistent/dir/file.html", ExportHTMLOptions{})
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
