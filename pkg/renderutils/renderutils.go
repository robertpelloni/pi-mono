package renderutils

import (
	"os"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ansitohtml"
)

// ShortenPath shortens a path for display by replacing home dir with ~.
func ShortenPath(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// Str converts an unknown value to a string, returning nil if not a string.
func Str(value interface{}) *string {
	if value == nil {
		empty := ""
		return &empty
	}
	if s, ok := value.(string); ok {
		return &s
	}
	return nil
}

// ReplaceTabs replaces tabs with spaces.
func ReplaceTabs(text string) string {
	return strings.ReplaceAll(text, "\t", " ")
}

// NormalizeDisplayText removes carriage returns.
func NormalizeDisplayText(text string) string {
	return strings.ReplaceAll(text, "\r", "")
}

// GetTextOutput extracts text output from a tool result content array.
// Returns the combined text content, with image indicators if images are present
// but showImages is false.
func GetTextOutput(
	content []map[string]interface{},
	showImages bool,
) string {
	if content == nil {
		return ""
	}

	var textBlocks []map[string]interface{}
	var imageBlocks []map[string]interface{}

	for _, c := range content {
		cType, _ := c["type"].(string)
		if cType == "text" {
			textBlocks = append(textBlocks, c)
		} else if cType == "image" {
			imageBlocks = append(imageBlocks, c)
		}
	}

	var textParts []string
	for _, c := range textBlocks {
		if text, ok := c["text"].(string); ok {
			sanitized := SanitizeBinaryOutput(ansitohtml.StripAnsi(text))
			sanitized = strings.ReplaceAll(sanitized, "\r", "")
			textParts = append(textParts, sanitized)
		}
	}
	output := strings.Join(textParts, "\n")

	if len(imageBlocks) > 0 && !showImages {
		for _, img := range imageBlocks {
			mimeType, _ := img["mimeType"].(string)
			if mimeType == "" {
				mimeType = "image/unknown"
			}
			output += "\n[" + mimeType + "]"
		}
	}

	return output
}

// SanitizeBinaryOutput removes characters that cause display issues.
func SanitizeBinaryOutput(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == 0x09, r == 0x0a, r == 0x0d: // tab, newline, carriage return
			b.WriteRune(r)
		case r <= 0x1f: // control chars
			// skip
		case r >= 0xfff9 && r <= 0xfffb: // Unicode format characters
			// skip
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
