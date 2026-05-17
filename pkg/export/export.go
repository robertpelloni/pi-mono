package export

import (
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// ExportHTMLOptions configures HTML export.
type ExportHTMLOptions struct {
	Title     string
	Theme     string // "light", "dark", or "auto" (default)
	FontSize  int    // Font size in pixels (default 14)
	MaxOutput int    // Max tool output characters (default 5000)
}

// ExportHTML exports conversation messages to an HTML file.
func ExportHTML(messages []ai.Message, outputPath string, options ExportHTMLOptions) error {
	if options.FontSize == 0 {
		options.FontSize = 14
	}
	if options.MaxOutput == 0 {
		options.MaxOutput = 5000
	}
	if options.Theme == "" {
		options.Theme = "auto"
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer f.Close()

	return WriteHTML(messages, f, options)
}

// WriteHTML writes HTML-serialized conversation to a writer.
func WriteHTML(messages []ai.Message, w io.Writer, options ExportHTMLOptions) error {
	title := options.Title
	if title == "" {
		title = fmt.Sprintf("Pi Session - %s", time.Now().Format("2006-01-02 15:04"))
	}

	// Write HTML header
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
`, html.EscapeString(title))

	writeCSS(w, options)

	fmt.Fprintf(w, `</style>
</head>
<body>
<div class="container">
<h1>%s</h1>
<div class="timestamp">%s</div>
`, html.EscapeString(title), time.Now().Format("Mon Jan 2 15:04:05 2006"))

	// Write messages
	for _, msg := range messages {
		switch msg.GetRole() {
		case ai.RoleUser:
			writeUserMessage(w, msg, options)
		case ai.RoleAssistant:
			writeAssistantMessage(w, msg, options)
		case ai.RoleTool:
			writeToolResultMessage(w, msg, options)
		}
	}

	fmt.Fprintf(w, `</div>
</body>
</html>
`)

	return nil
}

func writeCSS(w io.Writer, options ExportHTMLOptions) {
	themeCSS := getThemeCSS(options.Theme)
	fmt.Fprintf(w, `%s
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    font-size: %dpx;
    line-height: 1.6;
    margin: 0;
    padding: 20px;
}
.container {
    max-width: 900px;
    margin: 0 auto;
}
.message {
    margin: 16px 0;
    padding: 12px 16px;
    border-radius: 8px;
}
.user {
    background: var(--user-bg);
    border: 1px solid var(--user-border);
}
.assistant {
    background: var(--assistant-bg);
    border: 1px solid var(--assistant-border);
}
.tool-result {
    background: var(--tool-bg);
    border: 1px solid var(--tool-border);
    font-family: "SF Mono", "Fira Code", monospace;
    font-size: %dpx;
}
.role-label {
    font-weight: bold;
    margin-bottom: 8px;
    color: var(--label-color);
}
.content {
    white-space: pre-wrap;
    word-break: break-word;
}
.timestamp {
    color: var(--muted);
    font-size: 12px;
    margin-bottom: 20px;
}
h1 { color: var(--heading-color); }
`, themeCSS, options.FontSize, options.FontSize-1)
}

func getThemeCSS(theme string) string {
	switch theme {
	case "dark":
		return darkThemeCSS
	case "light":
		return lightThemeCSS
	default: // "auto"
		return autoThemeCSS
	}
}

const lightThemeCSS = `
:root {
    --bg: #ffffff;
    --text: #1a1a1a;
    --user-bg: #f0f4ff;
    --user-border: #c7d2fe;
    --assistant-bg: #f9fafb;
    --assistant-border: #e5e7eb;
    --tool-bg: #f8f9fa;
    --tool-border: #dee2e6;
    --label-color: #374151;
    --heading-color: #111827;
    --muted: #9ca3af;
}
body { background: var(--bg); color: var(--text); }
`

const darkThemeCSS = `
:root {
    --bg: #1a1a2e;
    --text: #e0e0e0;
    --user-bg: #1e2a3a;
    --user-border: #2d4a6f;
    --assistant-bg: #22223b;
    --assistant-border: #3a3a5c;
    --tool-bg: #1a1a2e;
    --tool-border: #333355;
    --label-color: #a0a0c0;
    --heading-color: #e0e0e0;
    --muted: #666688;
}
body { background: var(--bg); color: var(--text); }
`

const autoThemeCSS = lightThemeCSS + `
@media (prefers-color-scheme: dark) {
:root {
    --bg: #1a1a2e;
    --text: #e0e0e0;
    --user-bg: #1e2a3a;
    --user-border: #2d4a6f;
    --assistant-bg: #22223b;
    --assistant-border: #3a3a5c;
    --tool-bg: #1a1a2e;
    --tool-border: #333355;
    --label-color: #a0a0c0;
    --heading-color: #e0e0e0;
    --muted: #666688;
}
}
`

func writeUserMessage(w io.Writer, msg ai.Message, options ExportHTMLOptions) {
	fmt.Fprintf(w, `<div class="message user">
<div class="role-label">You</div>
<div class="content">`)
	text := extractTextContent(msg)
	fmt.Fprintf(w, "%s", html.EscapeString(text))
	fmt.Fprintf(w, `</div></div>`)
}

func writeAssistantMessage(w io.Writer, msg ai.Message, options ExportHTMLOptions) {
	fmt.Fprintf(w, `<div class="message assistant">
<div class="role-label">Assistant</div>
<div class="content">`)
	text := extractTextContent(msg)
	fmt.Fprintf(w, "%s", html.EscapeString(text))
	fmt.Fprintf(w, `</div></div>`)
}

func writeToolResultMessage(w io.Writer, msg ai.Message, options ExportHTMLOptions) {
	fmt.Fprintf(w, `<div class="message tool-result">
<div class="role-label">Tool Result</div>
<div class="content">`)
	text := extractTextContent(msg)
	if len(text) > options.MaxOutput {
		text = text[:options.MaxOutput] + "\n\n[... truncated ...]"
	}
	fmt.Fprintf(w, "%s", html.EscapeString(text))
	fmt.Fprintf(w, `</div></div>`)
}

// extractTextContent extracts all text from a message.
func extractTextContent(msg ai.Message) string {
	var parts []string

	switch m := msg.(type) {
	case ai.UserMessage:
		for _, c := range m.Content {
			if tc, ok := c.(ai.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
	case ai.AssistantMessage:
		for _, c := range m.Content {
			if tc, ok := c.(ai.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
	case ai.ToolResultMessage:
		for _, c := range m.Content {
			if tc, ok := c.(ai.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// AnsiToHTML converts ANSI escape sequences to HTML spans.
func AnsiToHTML(text string) string {
	// Strip all ANSI sequences
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(text, "")
}
