package ansitohtml

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ANSI color palette (0-15)
var ansiColors = []string{
	"#000000", "#800000", "#008000", "#808000",
	"#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00",
	"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

// TextStyle represents active ANSI text styling.
type TextStyle struct {
	Fg        *string
	Bg        *string
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
}

func newTextStyle() TextStyle {
	return TextStyle{}
}

func (s TextStyle) hasStyle() bool {
	return s.Fg != nil || s.Bg != nil || s.Bold || s.Dim || s.Italic || s.Underline
}

func (s TextStyle) toInlineCSS() string {
	var parts []string
	if s.Fg != nil {
		parts = append(parts, fmt.Sprintf("color:%s", *s.Fg))
	}
	if s.Bg != nil {
		parts = append(parts, fmt.Sprintf("background-color:%s", *s.Bg))
	}
	if s.Bold {
		parts = append(parts, "font-weight:bold")
	}
	if s.Dim {
		parts = append(parts, "opacity:0.6")
	}
	if s.Italic {
		parts = append(parts, "font-style:italic")
	}
	if s.Underline {
		parts = append(parts, "text-decoration:underline")
	}
	return strings.Join(parts, ";")
}

// color256ToHex converts a 256-color index to hex.
func color256ToHex(index int) string {
	if index < 16 {
		return ansiColors[index]
	}
	if index < 232 {
		cubeIndex := index - 16
		r := cubeIndex / 36
		g := (cubeIndex % 36) / 6
		b := cubeIndex % 6
		toComponent := func(n int) int {
			if n == 0 {
				return 0
			}
			return 55 + n*40
		}
		return fmt.Sprintf("#%02x%02x%02x", toComponent(r), toComponent(g), toComponent(b))
	}
	// Grayscale (232-255)
	gray := 8 + (index-232)*10
	return fmt.Sprintf("#%02x%02x%02x", gray, gray, gray)
}

// escapeHTML escapes HTML special characters.
func escapeHTML(text string) string {
	s := strings.ReplaceAll(text, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#039;")
	return s
}

// applySgrCode parses ANSI SGR codes and updates the style.
func applySgrCode(params []int, style *TextStyle) {
	i := 0
	for i < len(params) {
		code := params[i]
		switch {
		case code == 0:
			// Reset all
			style.Fg = nil
			style.Bg = nil
			style.Bold = false
			style.Dim = false
			style.Italic = false
			style.Underline = false
		case code == 1:
			style.Bold = true
		case code == 2:
			style.Dim = true
		case code == 3:
			style.Italic = true
		case code == 4:
			style.Underline = true
		case code == 22:
			style.Bold = false
			style.Dim = false
		case code == 23:
			style.Italic = false
		case code == 24:
			style.Underline = false
		case code >= 30 && code <= 37:
			c := ansiColors[code-30]
			style.Fg = &c
		case code == 38:
			if i+1 < len(params) && params[i+1] == 5 && i+2 < len(params) {
				c := color256ToHex(params[i+2])
				style.Fg = &c
				i += 2
			} else if i+1 < len(params) && params[i+1] == 2 && i+4 < len(params) {
				c := fmt.Sprintf("rgb(%d,%d,%d)", params[i+2], params[i+3], params[i+4])
				style.Fg = &c
				i += 4
			}
		case code == 39:
			style.Fg = nil
		case code >= 40 && code <= 47:
			c := ansiColors[code-40]
			style.Bg = &c
		case code == 48:
			if i+1 < len(params) && params[i+1] == 5 && i+2 < len(params) {
				c := color256ToHex(params[i+2])
				style.Bg = &c
				i += 2
			} else if i+1 < len(params) && params[i+1] == 2 && i+4 < len(params) {
				c := fmt.Sprintf("rgb(%d,%d,%d)", params[i+2], params[i+3], params[i+4])
				style.Bg = &c
				i += 4
			}
		case code == 49:
			style.Bg = nil
		case code >= 90 && code <= 97:
			c := ansiColors[code-90+8]
			style.Fg = &c
		case code >= 100 && code <= 107:
			c := ansiColors[code-100+8]
			style.Bg = &c
		}
		i++
	}
}

// ansiRegex matches ANSI escape sequences: ESC[ followed by params and ending with 'm'
var ansiRegex = regexp.MustCompile(`\x1b\[([\d;]*)m`)

// AnsiToHTML converts ANSI-escaped text to HTML with inline styles.
func AnsiToHTML(text string) string {
	style := newTextStyle()
	var result strings.Builder
	lastIndex := 0
	inSpan := false

	matches := ansiRegex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		// Add text before this escape sequence
		if match[0] > lastIndex {
			result.WriteString(escapeHTML(text[lastIndex:match[0]]))
		}

		// Parse SGR parameters
		paramStr := text[match[2]:match[3]]
		var params []int
		if paramStr != "" {
			for _, p := range strings.Split(paramStr, ";") {
				n, _ := strconv.Atoi(p)
				params = append(params, n)
			}
		} else {
			params = []int{0}
		}

		// Close existing span
		if inSpan {
			result.WriteString("</span>")
			inSpan = false
		}

		// Apply the codes
		applySgrCode(params, &style)

		// Open new span if we have styling
		if style.hasStyle() {
			result.WriteString(fmt.Sprintf(`<span style="%s">`, style.toInlineCSS()))
			inSpan = true
		}

		lastIndex = match[1]
	}

	// Add remaining text
	if lastIndex < len(text) {
		result.WriteString(escapeHTML(text[lastIndex:]))
	}

	// Close any open span
	if inSpan {
		result.WriteString("</span>")
	}

	return result.String()
}

// AnsiLinesToHTML converts an array of ANSI-escaped lines to HTML.
// Each line is wrapped in a div element.
func AnsiLinesToHTML(lines []string) string {
	var result strings.Builder
	for _, line := range lines {
		converted := AnsiToHTML(line)
		if converted == "" {
			converted = "&nbsp;"
		}
		result.WriteString(fmt.Sprintf(`<div class="ansi-line">%s</div>`, converted))
		result.WriteByte('\n')
	}
	return result.String()
}

// StripAnsi removes all ANSI escape sequences from text.
func StripAnsi(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}
