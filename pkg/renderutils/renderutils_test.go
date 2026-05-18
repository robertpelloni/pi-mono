package renderutils

import (
	"testing"
)

func TestShortenPath(t *testing.T) {
	tests := []struct {
		input    string
		contains string // Check if output contains this
	}{
		{"", ""},
		{"/some/path", "/some/path"},
	}

	for _, tt := range tests {
		result := ShortenPath(tt.input)
		if tt.input == "" && result != "" {
			t.Errorf("Expected empty for empty input, got %q", result)
		}
	}
}

func TestReplaceTabs(t *testing.T) {
	result := ReplaceTabs("hello\tworld\ttab")
	if result != "hello world tab" {
		t.Errorf("Expected tabs replaced, got %q", result)
	}
}

func TestNormalizeDisplayText(t *testing.T) {
	result := NormalizeDisplayText("hello\r\nworld\rtest")
	if result != "hello\nworldtest" {
		t.Errorf("Expected carriage returns removed, got %q", result)
	}
}

func TestSanitizeBinaryOutput(t *testing.T) {
	// Control characters should be removed
	input := "hello\x00\x01\x02world"
	result := SanitizeBinaryOutput(input)
	if result != "helloworld" {
		t.Errorf("Expected control chars removed, got %q", result)
	}

	// Tab, newline, carriage return should be kept
	input2 := "line1\nline2\ttab\rreturn"
	result2 := SanitizeBinaryOutput(input2)
	if result2 != input2 {
		t.Errorf("Expected whitespace preserved, got %q", result2)
	}
}

func TestStr(t *testing.T) {
	// String value
	s := Str("hello")
	if s == nil || *s != "hello" {
		t.Errorf("Expected 'hello', got %v", s)
	}

	// Non-string value
	n := Str(42)
	if n != nil {
		t.Errorf("Expected nil for non-string, got %v", n)
	}

	// Nil value
	nilResult := Str(nil)
	if nilResult == nil || *nilResult != "" {
		t.Errorf("Expected empty string for nil, got %v", nilResult)
	}
}

func TestGetTextOutput(t *testing.T) {
	content := []map[string]interface{}{
		{"type": "text", "text": "Hello world"},
		{"type": "image", "mimeType": "image/png", "data": "base64data"},
	}

	result := GetTextOutput(content, false)
	if result == "" {
		t.Error("Expected non-empty text output")
	}
}
