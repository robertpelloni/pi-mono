// Package termdetect detects terminal capabilities by inspecting environment
// variables. It mirrors the TypeScript TUI's detectCapabilities() logic.
package termdetect

import (
	"fmt"
	"os"
	"strings"
)

// ImageProtocol identifies the terminal image protocol, if any.
type ImageProtocol string

const (
	ImageProtocolNone ImageProtocol = ""
	ImageProtocolKitty ImageProtocol = "kitty"
	ImageProtocolITerm2 ImageProtocol = "iterm2"
)

// TerminalCapabilities describes what a terminal supports.
type TerminalCapabilities struct {
	Images    ImageProtocol
	TrueColor bool
	Hyperlinks bool
}

var cachedCaps *TerminalCapabilities

// DetectCapabilities reads environment variables to determine terminal capabilities.
func DetectCapabilities() TerminalCapabilities {
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))

	// Kitty — check KITTY_WINDOW_ID or TERM_PROGRAM
	if os.Getenv("KITTY_WINDOW_ID") != "" || termProgram == "kitty" {
		return TerminalCapabilities{
			Images:     ImageProtocolKitty,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// Ghostty
	if termProgram == "ghostty" || strings.Contains(term, "ghostty") || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return TerminalCapabilities{
			Images:     ImageProtocolKitty,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// WezTerm
	if os.Getenv("WEZTERM_PANE") != "" || termProgram == "wezterm" {
		return TerminalCapabilities{
			Images:     ImageProtocolKitty,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" || termProgram == "iterm.app" {
		return TerminalCapabilities{
			Images:     ImageProtocolITerm2,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// VS Code integrated terminal
	if termProgram == "vscode" {
		return TerminalCapabilities{
			Images:     ImageProtocolNone,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// Alacritty
	if termProgram == "alacritty" {
		return TerminalCapabilities{
			Images:     ImageProtocolNone,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// Windows Terminal (commonly detected via WT_SESSION)
	if os.Getenv("WT_SESSION") != "" || termProgram == "windowsterminal" || term == "win32-terminal" {
		return TerminalCapabilities{
			Images:     ImageProtocolNone,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// tmux — hyperlinks depend on the outer terminal; assume true for modern tmux
	if strings.HasPrefix(term, "tmux") || term == "screen-256color" {
		return TerminalCapabilities{
			Images:     ImageProtocolNone,
			TrueColor:  true,
			Hyperlinks: true,
		}
	}

	// Default: check COLORTERM for true color, hyperlinks only for known-capable TERMs
	trueColor := colorTerm == "truecolor" || colorTerm == "24bit"

	// xterm-256color and other modern xterm variants support OSC 8
	isXTerm := strings.HasPrefix(term, "xterm") || term == "gnome-terminal" || term == "konsole"

	return TerminalCapabilities{
		Images:     ImageProtocolNone,
		TrueColor:  trueColor,
		Hyperlinks: isXTerm,
	}
}

// GetCapabilities returns cached terminal capabilities, detecting on first call.
func GetCapabilities() TerminalCapabilities {
	if cachedCaps == nil {
		caps := DetectCapabilities()
		cachedCaps = &caps
	}
	return *cachedCaps
}

// ResetCaps clears the capabilities cache (useful in tests).
func ResetCaps() {
	cachedCaps = nil
}

// SupportsHyperlinks returns true if the terminal supports OSC 8 hyperlinks.
func SupportsHyperlinks() bool {
	return GetCapabilities().Hyperlinks
}

// SupportsTrueColor returns true if the terminal supports 24-bit true color.
func SupportsTrueColor() bool {
	return GetCapabilities().TrueColor
}

// SupportsImages returns the image protocol supported by the terminal, or "" if none.
func SupportsImages() ImageProtocol {
	return GetCapabilities().Images
}

// WrapHyperlink wraps text in an OSC 8 hyperlink escape sequence.
// If the terminal does not support hyperlinks, it returns the text unchanged.
// Format: ESC ]8 ; params ; uri ST text ESC ]8 ; ; ST
func WrapHyperlink(text, uri string) string {
	if !SupportsHyperlinks() {
		return text
	}
	// OSC 8: \x1b]8;<params>;<uri>\x07 text \x1b]8;;\x07
	// \x07 is the ST (string terminator), also \x1b\\ works
	return fmt.Sprintf("\x1b]8;;%s\x07%s\x1b]8;;\x07", uri, text)
}

// WrapHyperlinkPiped is like WrapHyperlink but uses ST as ESC \ (for pipes).
func WrapHyperlinkPiped(text, uri string) string {
	if !SupportsHyperlinks() {
		return text
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", uri, text)
}
