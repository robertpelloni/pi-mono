package export

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// ExportSession exports a conversation to HTML format.
func ExportSessionHTML(messages []ai.Message, outputPath string, meta SessionMeta) error {
	var body strings.Builder

	body.WriteString(htmlHeader(meta))

	for _, msg := range messages {
		switch m := msg.(type) {
		case ai.UserMessage:
			body.WriteString(renderUserMessage(m))
		case ai.AssistantMessage:
			body.WriteString(renderAssistantMessage(m))
		case ai.ToolResultMessage:
			body.WriteString(renderToolResult(m))
		}
	}

	body.WriteString(htmlFooter(meta))

	return os.WriteFile(outputPath, []byte(body.String()), 0644)
}

// ExportSessionJSONL exports a conversation to JSONL format.
func ExportSessionJSONL(messages []ai.Message, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, msg := range messages {
		wrapper := struct {
			Role    string     `json:"role"`
			Message ai.Message `json:"message"`
		}{
			Role:    string(msg.GetRole()),
			Message: msg,
		}

		data, err := json.Marshal(wrapper)
		if err != nil {
			continue
		}
		f.Write(data)
		f.Write([]byte("\n"))
	}

	return nil
}

// SessionMeta contains metadata for the exported session.
type SessionMeta struct {
	Title   string `json:"title"`
	Model   string `json:"model"`
	Provider string `json:"provider"`
	Version string `json:"version"`
	Date    string `json:"date"`
	MessageCount int `json:"messageCount"`
}

// --- HTML Rendering ---

func htmlHeader(meta SessionMeta) string {
	if meta.Date == "" {
		meta.Date = time.Now().Format("2006-01-02 15:04:05")
	}
	if meta.Version == "" {
		meta.Version = "pi-go"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  :root { --bg: #1a1a2e; --surface: #16213e; --text: #e0e0e0; --accent: #0f3460; --user: #7b2ff7; --assistant: #00b4d8; --tool: #f77f00; --error: #ef476f; --thinking: #6c63ff; }
  body { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace; background: var(--bg); color: var(--text); max-width: 900px; margin: 0 auto; padding: 20px; }
  .header { background: var(--surface); border-radius: 8px; padding: 16px; margin-bottom: 20px; }
  .header h1 { margin: 0 0 8px 0; font-size: 18px; }
  .header .meta { font-size: 12px; color: #888; }
  .message { margin: 12px 0; border-radius: 8px; padding: 12px 16px; }
  .user { background: linear-gradient(135deg, rgba(123,47,247,0.15), rgba(123,47,247,0.05)); border-left: 3px solid var(--user); }
  .assistant { background: linear-gradient(135deg, rgba(0,180,216,0.15), rgba(0,180,216,0.05)); border-left: 3px solid var(--assistant); }
  .tool { background: linear-gradient(135deg, rgba(247,127,0,0.15), rgba(247,127,0,0.05)); border-left: 3px solid var(--tool); }
  .error { border-left-color: var(--error); }
  .role { font-weight: bold; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
  .user .role { color: var(--user); }
  .assistant .role { color: var(--assistant); }
  .tool .role { color: var(--tool); }
  .content { white-space: pre-wrap; word-wrap: break-word; font-size: 13px; line-height: 1.5; }
  .thinking { color: var(--thinking); font-style: italic; }
  .tool-name { font-weight: bold; }
  .stats { font-size: 11px; color: #666; margin-top: 20px; text-align: center; }
  code { background: rgba(255,255,255,0.05); padding: 2px 6px; border-radius: 3px; font-size: 12px; }
</style>
</head>
<body>
<div class="header">
  <h1>%s</h1>
  <div class="meta">%s | %s | %s | %d messages</div>
</div>
`, html.EscapeString(meta.Title), html.EscapeString(meta.Title), html.EscapeString(meta.Version), html.EscapeString(meta.Model), html.EscapeString(meta.Provider), meta.MessageCount)
}

func htmlFooter(meta SessionMeta) string {
	return fmt.Sprintf(`
<div class="stats">
  Exported by %s on %s | %d messages
</div>
</body>
</html>
`, html.EscapeString(meta.Version), html.EscapeString(meta.Date), meta.MessageCount)
}

func renderUserMessage(msg ai.UserMessage) string {
	var content strings.Builder
	for _, c := range msg.Content {
		switch v := c.(type) {
		case ai.TextContent:
			content.WriteString(html.EscapeString(v.Text))
		case ai.ImageContent:
			content.WriteString(fmt.Sprintf(`<img src="%s" alt="image" style="max-width:100%%;border-radius:4px;">`, html.EscapeString(v.Data)))
		}
	}

	return fmt.Sprintf(`
<div class="message user">
  <div class="role">User</div>
  <div class="content">%s</div>
</div>
`, content.String())
}

func renderAssistantMessage(msg ai.AssistantMessage) string {
	var content strings.Builder

	for _, c := range msg.Content {
		switch v := c.(type) {
		case ai.TextContent:
			content.WriteString(html.EscapeString(v.Text))
		case ai.ThinkingContent:
			content.WriteString(fmt.Sprintf(`<span class="thinking">[Thinking] %s</span>`, html.EscapeString(v.Thinking)))
		case ai.ToolCall:
			argBytes, _ := json.Marshal(v.Arguments)
			content.WriteString(fmt.Sprintf(`<span class="tool-name">[%s]</span>(%s)`, html.EscapeString(v.Name), html.EscapeString(string(argBytes))))
		}
	}

	model := html.EscapeString(msg.Model)
	if model == "" {
		model = "assistant"
	}

	return fmt.Sprintf(`
<div class="message assistant">
  <div class="role">Assistant (%s)</div>
  <div class="content">%s</div>
</div>
`, model, content.String())
}

func renderToolResult(msg ai.ToolResultMessage) string {
	var content strings.Builder
	for _, c := range msg.Content {
		if txt, ok := c.(ai.TextContent); ok {
			content.WriteString(html.EscapeString(txt.Text))
		}
	}

	errorClass := ""
	if msg.IsError {
		errorClass = " error"
	}

	return fmt.Sprintf(`
<div class="message tool%s">
  <div class="role">Tool: %s</div>
  <div class="content">%s</div>
</div>
`, errorClass, html.EscapeString(msg.ToolName), content.String())
}

// DetectExportFormat determines the export format from a file extension.
func DetectExportFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		return "html"
	case ".jsonl", ".json":
		return "jsonl"
	default:
		return "html"
	}
}
