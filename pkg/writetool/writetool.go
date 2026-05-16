package writetool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// CreateWriteTool creates the write tool that writes content to files.
func CreateWriteTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "write",
		Label:       "write",
		Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Automatically creates parent directories.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to write (relative or absolute)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
		PromptSnippet: "Create or overwrite files",
		PromptGuidelines: []string{
			"Use write only for new files or complete rewrites.",
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, _ := params["path"].(string)
			content, _ := params["content"].(string)
			if path == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing path parameter")
			}

			// Resolve path
			absolutePath := path
			if !filepath.IsAbs(path) {
				absolutePath = filepath.Join(cwd, path)
			}

			// Create parent directories if needed
			dir := filepath.Dir(absolutePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("cannot create directory %s: %w", dir, err)
			}

			// Write the file
			if err := os.WriteFile(absolutePath, []byte(content), 0644); err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("cannot write file: %w", err)
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)},
				},
			}, nil
		},
	}
}
