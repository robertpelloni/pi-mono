package toolrenderer

import (
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ansitohtml"
)

// ToolDefinitionInfo provides tool rendering metadata.
type ToolDefinitionInfo struct {
	Name        string
	Label       string
	Description string
}

// ToolHtmlRenderer renders tool calls and results to HTML for export.
type ToolHtmlRenderer struct {
	GetToolDefinition func(name string) *ToolDefinitionInfo
	Theme             map[string]string
	CWD               string
	Width             int
}

// ToolHtmlRendererDeps configures the tool renderer.
type ToolHtmlRendererDeps struct {
	GetToolDefinition func(name string) *ToolDefinitionInfo
	Theme             map[string]string
	CWD               string
	Width             int
}

// NewToolHtmlRenderer creates a tool HTML renderer.
func NewToolHtmlRenderer(deps ToolHtmlRendererDeps) *ToolHtmlRenderer {
	width := deps.Width
	if width == 0 {
		width = 100
	}
	return &ToolHtmlRenderer{
		GetToolDefinition: deps.GetToolDefinition,
		Theme:             deps.Theme,
		CWD:               deps.CWD,
		Width:             width,
	}
}

// RenderResultOutput renders tool result content to HTML.
func (r *ToolHtmlRenderer) RenderResultOutput(
	toolName string,
	content []map[string]interface{},
	isError bool,
) (collapsed, expanded string) {
	toolDef := r.GetToolDefinition(toolName)
	label := toolName
	if toolDef != nil && toolDef.Label != "" {
		label = toolDef.Label
	}

	collapsed = renderToolOutputCollapsed(label, content, isError)
	expanded = renderToolOutputExpanded(label, content, isError)
	return collapsed, expanded
}

func renderToolOutputCollapsed(toolName string, content []map[string]interface{}, isError bool) string {
	var textParts []string
	var imageCount int

	for _, c := range content {
		cType, _ := c["type"].(string)
		if cType == "text" {
			if text, ok := c["text"].(string); ok {
				lines := strings.Split(text, "\n")
				if len(lines) > 0 {
					preview := lines[0]
					if len(preview) > 100 {
						preview = preview[:100] + "..."
					}
					textParts = append(textParts, preview)
				}
			}
		} else if cType == "image" {
			imageCount++
		}
	}

	cls := "tool-result"
	if isError {
		cls = "tool-result error"
	}

	result := fmt.Sprintf(`<div class="%s">`, cls)
	if len(textParts) > 0 {
		result += ansitohtml.AnsiToHTML(textParts[0])
	}
	if imageCount > 0 {
		result += fmt.Sprintf(` <span class="image-count">[%d image(s)]</span>`, imageCount)
	}
	result += `</div>`
	return result
}

func renderToolOutputExpanded(toolName string, content []map[string]interface{}, isError bool) string {
	var textParts []string
	var imageCount int

	for _, c := range content {
		cType, _ := c["type"].(string)
		if cType == "text" {
			if text, ok := c["text"].(string); ok {
				textParts = append(textParts, text)
			}
		} else if cType == "image" {
			imageCount++
		}
	}

	cls := "tool-result expanded"
	if isError {
		cls = "tool-result expanded error"
	}

	result := fmt.Sprintf(`<div class="%s">`, cls)
	allText := joinNonEmpty(textParts, "\n")
	if allText != "" {
		result += ansitohtml.AnsiToHTML(allText)
	}
	if imageCount > 0 {
		result += fmt.Sprintf(`\n<span class="image-count">[%d image(s)]</span>`, imageCount)
	}
	result += `</div>`
	return result
}

// RenderToolCall renders a tool call invocation to HTML.
func (r *ToolHtmlRenderer) RenderToolCall(toolName string, args map[string]interface{}) string {
	label := toolName
	if toolDef := r.GetToolDefinition(toolName); toolDef != nil && toolDef.Label != "" {
		label = toolDef.Label
	}

	var argPreview string
	if cmd, ok := args["command"].(string); ok {
		argPreview = cmd
	} else if filePath, ok := args["file_path"].(string); ok {
		argPreview = filePath
	} else if pattern, ok := args["pattern"].(string); ok {
		argPreview = pattern
	} else if path, ok := args["path"].(string); ok {
		argPreview = path
	}

	if argPreview != "" {
		if len(argPreview) > 80 {
			argPreview = argPreview[:80] + "..."
		}
		return fmt.Sprintf(`<div class="tool-call"><span class="tool-name">%s</span> <span class="tool-args">%s</span></div>`,
			ansitohtml.AnsiToHTML(label), escapeHTML(argPreview))
	}
	return fmt.Sprintf(`<div class="tool-call"><span class="tool-name">%s</span></div>`,
		ansitohtml.AnsiToHTML(label))
}

func escapeHTML(text string) string {
	s := strings.ReplaceAll(text, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#039;")
	return s
}

func joinNonEmpty(parts []string, sep string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}
