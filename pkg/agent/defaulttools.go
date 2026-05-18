package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// DefaultLoopConfig returns a sensible default AgentLoopConfig.
func DefaultLoopConfig() AgentLoopConfig {
	return AgentLoopConfig{
		ToolExecution: ToolExecutionParallel,
	}
}

// DefaultTools returns the default set of tools (read, bash, edit, write).
// These are minimal implementations; the full tool implementations
// are in pkg/nativetools/ and should be used instead when available.
func DefaultTools() []AgentTool {
	return []AgentTool{
		{
			Name:        "read",
			Description: "Read the contents of a file.",
			Execute: func(ctx context.Context, toolCallId string, args map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
				filePath, _ := args["file_path"].(string)
				if filePath == "" {
					errMsg := "Error: file_path is required"
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, fmt.Errorf("file_path is required")
				}
				data, err := os.ReadFile(filePath)
				if err != nil {
					errMsg := fmt.Sprintf("Error reading file: %v", err)
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, err
				}
				return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: string(data)}}}, nil
			},
		},
		{
			Name:        "bash",
			Description: "Execute a bash command.",
			Execute: func(ctx context.Context, toolCallId string, args map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
				command, _ := args["command"].(string)
				if command == "" {
					errMsg := "Error: command is required"
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, fmt.Errorf("command is required")
				}
				out, err := exec.Command("sh", "-c", command).CombinedOutput()
				if err != nil {
					errMsg := fmt.Sprintf("Error: %v\n%s", err, string(out))
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, err
				}
				return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: strings.TrimSpace(string(out))}}}, nil
			},
		},
		{
			Name:        "edit",
			Description: "Edit a file using find/replace.",
			Execute: func(ctx context.Context, toolCallId string, args map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
				return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Edit tool placeholder - use pkg/nativetools for full implementation"}}}, nil
			},
		},
		{
			Name:        "write",
			Description: "Write content to a file.",
			Execute: func(ctx context.Context, toolCallId string, args map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
				filePath, _ := args["file_path"].(string)
				content, _ := args["content"].(string)
				if filePath == "" {
					errMsg := "Error: file_path is required"
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, fmt.Errorf("file_path is required")
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					errMsg := fmt.Sprintf("Error writing file: %v", err)
					return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: errMsg}}}, err
				}
				return AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "File written successfully"}}}, nil
			},
		},
	}
}
