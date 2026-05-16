package truncate

import (
	"strings"
	"testing"
)

func TestTruncateHead_NoTruncation(t *testing.T) {
	result := TruncateHead("hello\nworld", TruncationOptions{MaxLines: 10, MaxBytes: 1024})
	if result.Truncated {
		t.Error("Expected no truncation")
	}
	if result.Content != "hello\nworld" {
		t.Errorf("Expected original content, got %q", result.Content)
	}
}

func TestTruncateHead_ByLines(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	content := strings.Join(lines, "\n")

	result := TruncateHead(content, TruncationOptions{MaxLines: 10, MaxBytes: 1024 * 1024})
	if !result.Truncated {
		t.Error("Expected truncation")
	}
	if result.TruncatedBy != "lines" {
		t.Errorf("Expected truncation by lines, got %q", result.TruncatedBy)
	}
	if result.OutputLines != 10 {
		t.Errorf("Expected 10 output lines, got %d", result.OutputLines)
	}
}

func TestTruncateHead_ByBytes(t *testing.T) {
	content := strings.Repeat("a", 1000)

	result := TruncateHead(content, TruncationOptions{MaxLines: 2000, MaxBytes: 100})
	if !result.Truncated {
		t.Error("Expected truncation")
	}
	if result.TruncatedBy != "bytes" {
		t.Errorf("Expected truncation by bytes, got %q", result.TruncatedBy)
	}
}

func TestTruncateTail_NoTruncation(t *testing.T) {
	result := TruncateTail("hello\nworld", TruncationOptions{MaxLines: 10, MaxBytes: 1024})
	if result.Truncated {
		t.Error("Expected no truncation")
	}
}

func TestTruncateTail_ByLines(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	content := strings.Join(lines, "\n")

	result := TruncateTail(content, TruncationOptions{MaxLines: 10, MaxBytes: 1024 * 1024})
	if !result.Truncated {
		t.Error("Expected truncation")
	}
	if result.OutputLines != 10 {
		t.Errorf("Expected 10 output lines, got %d", result.OutputLines)
	}
	// Should keep the last lines
	if !strings.HasPrefix(result.Content, "line") {
		t.Errorf("Expected content to start with 'line', got %q", result.Content[:20])
	}
}

func TestTruncateLine(t *testing.T) {
	short := "hello"
	text, truncated := TruncateLine(short, 10)
	if truncated {
		t.Error("Expected no truncation for short line")
	}
	if text != short {
		t.Errorf("Expected %q, got %q", short, text)
	}

	long := strings.Repeat("a", 1000)
	text2, truncated2 := TruncateLine(long, 100)
	if !truncated2 {
		t.Error("Expected truncation for long line")
	}
	if !strings.HasSuffix(text2, "... [truncated]") {
		t.Errorf("Expected truncation suffix, got %q", text2[len(text2)-20:])
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int
		expected string
	}{
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
	}
	for _, tt := range tests {
		if got := FormatSize(tt.bytes); got != tt.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}
