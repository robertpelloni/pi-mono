package ansitohtml

import (
	"strings"
	"testing"
)

func TestAnsiToHTML_PlainText(t *testing.T) {
	result := AnsiToHTML("Hello, world!")
	if result != "Hello, world!" {
		t.Errorf("Expected plain text unchanged, got %q", result)
	}
}

func TestAnsiToHTML_Bold(t *testing.T) {
	result := AnsiToHTML("\x1b[1mBold text\x1b[0m")
	if !strings.Contains(result, "font-weight:bold") {
		t.Errorf("Expected bold styling, got %q", result)
	}
	if !strings.Contains(result, "Bold text") {
		t.Errorf("Expected 'Bold text' in output, got %q", result)
	}
}

func TestAnsiToHTML_Color(t *testing.T) {
	result := AnsiToHTML("\x1b[31mRed text\x1b[0m")
	if !strings.Contains(result, "color:#800000") {
		t.Errorf("Expected red color, got %q", result)
	}
	if !strings.Contains(result, "Red text") {
		t.Errorf("Expected 'Red text' in output, got %q", result)
	}
}

func TestAnsiToHTML_EscapeHTML(t *testing.T) {
	result := AnsiToHTML("<script>alert('xss')</script>")
	if strings.Contains(result, "<script>") {
		t.Error("Expected HTML to be escaped")
	}
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Errorf("Expected escaped HTML, got %q", result)
	}
}

func TestAnsiToHTML_Reset(t *testing.T) {
	result := AnsiToHTML("\x1b[1mBold\x1b[0m Normal")
	if !strings.Contains(result, "</span>") {
		t.Error("Expected closing span after reset")
	}
	if !strings.Contains(result, "Normal") {
		t.Errorf("Expected 'Normal' after reset, got %q", result)
	}
}

func TestAnsiToHTML_256Color(t *testing.T) {
	result := AnsiToHTML("\x1b[38;5;196mColored\x1b[0m")
	if !strings.Contains(result, "color:") {
		t.Errorf("Expected 256-color styling, got %q", result)
	}
}

func TestStripAnsi(t *testing.T) {
	result := StripAnsi("\x1b[1;31mHello\x1b[0m World")
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}
}

func TestAnsiLinesToHTML(t *testing.T) {
	lines := []string{"Line 1", "\x1b[32mGreen Line\x1b[0m", "Line 3"}
	result := AnsiLinesToHTML(lines)
	if !strings.Contains(result, "ansi-line") {
		t.Error("Expected ansi-line class")
	}
	if !strings.Contains(result, "Line 1") {
		t.Error("Expected 'Line 1'")
	}
	if !strings.Contains(result, "Green Line") {
		t.Error("Expected 'Green Line'")
	}
}

func TestColor256ToHex(t *testing.T) {
	// Standard colors
	if color256ToHex(0) != "#000000" {
		t.Errorf("Expected #000000 for index 0, got %s", color256ToHex(0))
	}
	if color256ToHex(1) != "#800000" {
		t.Errorf("Expected #800000 for index 1, got %s", color256ToHex(1))
	}
	if color256ToHex(15) != "#ffffff" {
		t.Errorf("Expected #ffffff for index 15, got %s", color256ToHex(15))
	}
	// Color cube
	if color256ToHex(16) != "#000000" {
		t.Errorf("Expected #000000 for index 16, got %s", color256ToHex(16))
	}
	// Grayscale
	gray := color256ToHex(232)
	if !strings.HasPrefix(gray, "#") {
		t.Errorf("Expected hex for grayscale, got %s", gray)
	}
}
