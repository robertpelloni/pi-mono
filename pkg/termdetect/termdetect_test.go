package termdetect

import (
	"os"
	"testing"
)

func TestDetectCapabilities_Kitty(t *testing.T) {
	ResetCaps()
	os.Setenv("KITTY_WINDOW_ID", "1")
	defer os.Unsetenv("KITTY_WINDOW_ID")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for Kitty")
	}
	if !caps.TrueColor {
		t.Error("Expected truecolor for Kitty")
	}
	if caps.Images != ImageProtocolKitty {
		t.Errorf("Expected Kitty image protocol, got %s", caps.Images)
	}
}

func TestDetectCapabilities_KittyTermProgram(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "kitty")
	defer os.Unsetenv("TERM_PROGRAM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for TERM_PROGRAM=kitty")
	}
}

func TestDetectCapabilities_Ghostty(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "ghostty")
	defer os.Unsetenv("TERM_PROGRAM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for Ghostty")
	}
	if caps.Images != ImageProtocolKitty {
		t.Errorf("Expected Kitty image protocol for Ghostty, got %s", caps.Images)
	}
}

func TestDetectCapabilities_GhosttyEnv(t *testing.T) {
	ResetCaps()
	os.Setenv("GHOSTTY_RESOURCES_DIR", "/some/path")
	defer os.Unsetenv("GHOSTTY_RESOURCES_DIR")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for GHOSTTY_RESOURCES_DIR")
	}
}

func TestDetectCapabilities_WezTerm(t *testing.T) {
	ResetCaps()
	os.Setenv("WEZTERM_PANE", "0")
	defer os.Unsetenv("WEZTERM_PANE")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for WezTerm")
	}
	if caps.Images != ImageProtocolKitty {
		t.Errorf("Expected Kitty image protocol for WezTerm, got %s", caps.Images)
	}
}

func TestDetectCapabilities_ITerm2(t *testing.T) {
	ResetCaps()
	os.Setenv("ITERM_SESSION_ID", "test123")
	defer os.Unsetenv("ITERM_SESSION_ID")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for iTerm2")
	}
	if caps.Images != ImageProtocolITerm2 {
		t.Errorf("Expected iTerm2 image protocol, got %s", caps.Images)
	}
}

func TestDetectCapabilities_VSCode(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "vscode")
	defer os.Unsetenv("TERM_PROGRAM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for VS Code terminal")
	}
	if caps.Images != ImageProtocolNone {
		t.Errorf("Expected no image protocol for VS Code, got %s", caps.Images)
	}
}

func TestDetectCapabilities_Alacritty(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "alacritty")
	defer os.Unsetenv("TERM_PROGRAM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for Alacritty")
	}
}

func TestDetectCapabilities_WindowsTerminal(t *testing.T) {
	ResetCaps()
	os.Setenv("WT_SESSION", "test")
	defer os.Unsetenv("WT_SESSION")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for Windows Terminal")
	}
	if !caps.TrueColor {
		t.Error("Expected truecolor for Windows Terminal")
	}
}

func TestDetectCapabilities_Tmux(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM", "tmux-256color")
	defer os.Unsetenv("TERM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for tmux")
	}
}

func TestDetectCapabilities_Screen(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM", "screen-256color")
	defer os.Unsetenv("TERM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for screen-256color")
	}
}

func TestDetectCapabilities_XTerm(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM", "xterm-256color")
	defer os.Unsetenv("TERM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for xterm-256color")
	}
}

func TestDetectCapabilities_XTermDirect(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM", "xterm-direct")
	defer os.Unsetenv("TERM")
	os.Setenv("COLORTERM", "truecolor")
	defer os.Unsetenv("COLORTERM")

	caps := DetectCapabilities()
	if !caps.Hyperlinks {
		t.Error("Expected hyperlinks for xterm-direct")
	}
	if !caps.TrueColor {
		t.Error("Expected truecolor for xterm-direct")
	}
}

func TestDetectCapabilities_Unknown(t *testing.T) {
	ResetCaps()
	os.Unsetenv("TERM")
	os.Unsetenv("TERM_PROGRAM")
	os.Unsetenv("COLORTERM")
	os.Unsetenv("KITTY_WINDOW_ID")
	os.Unsetenv("GHOSTTY_RESOURCES_DIR")
	os.Unsetenv("WEZTERM_PANE")
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("WT_SESSION")

	caps := DetectCapabilities()
	if caps.Hyperlinks {
		t.Error("Expected no hyperlinks for unknown terminal")
	}
	if caps.TrueColor {
		t.Error("Expected no truecolor for unknown terminal without COLORTERM")
	}
}

func TestDetectCapabilities_TrueColorFallback(t *testing.T) {
	ResetCaps()
	os.Unsetenv("TERM")
	os.Unsetenv("TERM_PROGRAM")
	os.Unsetenv("KITTY_WINDOW_ID")
	os.Unsetenv("GHOSTTY_RESOURCES_DIR")
	os.Unsetenv("WEZTERM_PANE")
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("WT_SESSION")
	os.Setenv("COLORTERM", "truecolor")
	defer os.Unsetenv("COLORTERM")

	caps := DetectCapabilities()
	if caps.Hyperlinks {
		t.Error("Expected no hyperlinks for unknown terminal even with COLORTERM")
	}
	if !caps.TrueColor {
		t.Error("Expected truecolor when COLORTERM=truecolor")
	}
}

func TestWrapHyperlink_Unsupported(t *testing.T) {
	// For an unsupported terminal, WrapHyperlink should return text unchanged
	ResetCaps()
	os.Unsetenv("TERM")
	os.Unsetenv("TERM_PROGRAM")
	os.Unsetenv("COLORTERM")

	result := WrapHyperlink("click me", "https://example.com")
	if result != "click me" {
		t.Errorf("Expected 'click me', got '%s'", result)
	}
}

func TestWrapHyperlink_Supported(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "kitty")
	defer os.Unsetenv("TERM_PROGRAM")

	result := WrapHyperlink("click me", "https://example.com")
	expected := "\x1b]8;;https://example.com\x07click me\x1b]8;;\x07"
	if result != expected {
		t.Errorf("Expected OSC 8 wrapped link, got %q", result)
	}
}

func TestSupportsHyperlinks_Cached(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM_PROGRAM", "kitty")
	defer os.Unsetenv("TERM_PROGRAM")

	// First call detects and caches
	if !SupportsHyperlinks() {
		t.Error("Expected hyperlinks for Kitty")
	}

	// Change env - should NOT affect cached result
	os.Unsetenv("TERM_PROGRAM")
	if !SupportsHyperlinks() {
		t.Error("Expected cached hyperlinks to remain true")
	}
}

func TestGetCapabilities_MultipleCallsSame(t *testing.T) {
	ResetCaps()
	os.Setenv("TERM", "xterm-256color")
	defer os.Unsetenv("TERM")

	// Detect capabilities
	caps1 := GetCapabilities()
	caps2 := GetCapabilities()

	if caps1 != caps2 {
		t.Error("Expected same capabilities on multiple calls")
	}
}
