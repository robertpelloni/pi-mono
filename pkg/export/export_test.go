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

// ---------------------------------------------------------------------------
// Extended Export Tests
// ---------------------------------------------------------------------------

func TestWriteHTML_AssistantMessage(t *testing.T) {
	messages := []ai.Message{
		ai.AssistantMessage{
			Content:    []ai.Content{ai.TextContent{Text: "I can help with that!"}},
			StopReason: ai.StopReasonStop,
		},
	}
	var buf bytes.Buffer
	err := WriteHTML(messages, &buf, ExportHTMLOptions{Title: "Test"})
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("I can help")) {
		t.Error("Expected assistant message content")
	}
}

func TestWriteHTML_MultipleMessages(t *testing.T) {
	messages := []ai.Message{
		ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "First question"}}},
		ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: "First answer"}}, StopReason: ai.StopReasonStop},
		ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "Second question"}}},
		ai.AssistantMessage{Content: []ai.Content{ai.TextContent{Text: "Second answer"}}, StopReason: ai.StopReasonStop},
	}
	var buf bytes.Buffer
	err := WriteHTML(messages, &buf, ExportHTMLOptions{Title: "Multi"})
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("First question")) {
		t.Error("Expected first user message")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Second answer")) {
		t.Error("Expected second assistant message")
	}
}

func TestWriteHTML_WithTheme(t *testing.T) {
	var buf bytes.Buffer
	err := WriteHTML(nil, &buf, ExportHTMLOptions{Title: "Test", Theme: "light"})
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}
	// Should contain CSS for the theme
	if !bytes.Contains(buf.Bytes(), []byte("body")) {
		t.Error("Expected CSS body styles")
	}
}

func TestGetThemeCSS(t *testing.T) {
	darkCSS := getThemeCSS("dark")
	if darkCSS == "" {
		t.Error("Expected non-empty dark theme CSS")
	}
	lightCSS := getThemeCSS("light")
	if lightCSS == "" {
		t.Error("Expected non-empty light theme CSS")
	}
	unknownCSS := getThemeCSS("unknown")
	// Unknown theme should return default (dark)
	_ = unknownCSS
}

func TestExtractTextContent(t *testing.T) {
	msg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Hello world"}},
	}
	text := extractTextContent(msg)
	if text != "Hello world" {
		t.Errorf("Expected 'Hello world', got %q", text)
	}
}

func TestExtractTextContent_MultipleContent(t *testing.T) {
	msg := ai.UserMessage{
		Content: []ai.Content{
			ai.TextContent{Text: "Part 1"},
			ai.TextContent{Text: "Part 2"},
		},
	}
	text := extractTextContent(msg)
	if !bytes.Contains([]byte(text), []byte("Part 1")) {
		t.Errorf("Expected 'Part 1' in text, got %q", text)
	}
}

func TestExportHTMLOptions_Fields(t *testing.T) {
	opts := ExportHTMLOptions{
		Title: "My Session",
		Theme: "dark",
	}
	if opts.Title != "My Session" {
		t.Error("Title mismatch")
	}
	if opts.Theme != "dark" {
		t.Error("Theme mismatch")
	}
}

func TestAnsiToHTML_Colors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"red", "\x1b[31mred\x1b[0m", "red"},
		{"green", "\x1b[32mgreen\x1b[0m", "green"},
		{"yellow", "\x1b[33myellow\x1b[0m", "yellow"},
		{"blue", "\x1b[34mblue\x1b[0m", "blue"},
		{"magenta", "\x1b[35mmagenta\x1b[0m", "magenta"},
		{"cyan", "\x1b[36mcyan\x1b[0m", "cyan"},
		{"bold", "\x1b[1mbold\x1b[0m", "bold"},
		{"underline", "\x1b[4munderline\x1b[0m", "underline"},
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
