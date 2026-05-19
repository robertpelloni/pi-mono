package toolrenderer

import (
	"strings"
	"testing"
)

func TestNewToolHtmlRenderer(t *testing.T) {
	renderer := NewToolHtmlRenderer(ToolHtmlRendererDeps{
		GetToolDefinition: func(name string) *ToolDefinitionInfo {
			return &ToolDefinitionInfo{Name: name, Label: name}
		},
		CWD:   "/tmp",
		Width: 100,
	})
	if renderer == nil {
		t.Fatal("Expected non-nil renderer")
	}
	if renderer.Width != 100 {
		t.Errorf("Expected width 100, got %d", renderer.Width)
	}
}

func TestNewToolHtmlRenderer_DefaultWidth(t *testing.T) {
	renderer := NewToolHtmlRenderer(ToolHtmlRendererDeps{
		GetToolDefinition: func(name string) *ToolDefinitionInfo { return nil },
		CWD:   "/tmp",
		Width: 0,
	})
	if renderer.Width != 100 {
		t.Errorf("Expected default width 100, got %d", renderer.Width)
	}
}

func TestRenderResultOutput_TextContent(t *testing.T) {
	renderer := NewToolHtmlRenderer(ToolHtmlRendererDeps{
		GetToolDefinition: func(name string) *ToolDefinitionInfo {
			return &ToolDefinitionInfo{Name: "read", Label: "Read"}
		},
		CWD:   "/tmp",
		Width: 100,
	})

	collapsed, expanded := renderer.RenderResultOutput("read", []map[string]interface{}{
		{"type": "text", "text": "file contents here"},
	}, false)

	if collapsed == "" {
		t.Error("Expected non-empty collapsed output")
	}
	if expanded == "" {
		t.Error("Expected non-empty expanded output")
	}
	if !strings.Contains(collapsed, "tool-result") {
		t.Error("Expected tool-result class in collapsed output")
	}
}

func TestRenderResultOutput_Error(t *testing.T) {
	renderer := NewToolHtmlRenderer(ToolHtmlRendererDeps{
		GetToolDefinition: func(name string) *ToolDefinitionInfo {
			return &ToolDefinitionInfo{Name: "bash", Label: "Bash"}
		},
		CWD:   "/tmp",
		Width: 100,
	})

	collapsed, _ := renderer.RenderResultOutput("bash", []map[string]interface{}{
		{"type": "text", "text": "command not found"},
	}, true)

	if collapsed == "" {
		t.Error("Expected non-empty collapsed output for error")
	}
}

func TestRenderResultOutput_NoToolDef(t *testing.T) {
	renderer := NewToolHtmlRenderer(ToolHtmlRendererDeps{
		GetToolDefinition: func(name string) *ToolDefinitionInfo { return nil },
		CWD:   "/tmp",
		Width: 100,
	})

	collapsed, _ := renderer.RenderResultOutput("unknown_tool", []map[string]interface{}{
		{"type": "text", "text": "result"},
	}, false)

	if !strings.Contains(collapsed, "tool-result") {
		t.Error("Expected tool-result class in output when no tool def")
	}
}

func TestToolDefinitionInfo_Fields(t *testing.T) {
	td := ToolDefinitionInfo{
		Name:        "read",
		Label:       "Read",
		Description: "Read a file",
	}
	if td.Name != "read" || td.Label != "Read" || td.Description != "Read a file" {
		t.Error("ToolDefinitionInfo fields mismatch")
	}
}
