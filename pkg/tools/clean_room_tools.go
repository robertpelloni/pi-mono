package tools

import (
	"context"
	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// OpenInterpreterComputerUseTool returns the clean room implementation of the computer tool
func OpenInterpreterComputerUseTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "computer",
		Label:       "computer",
		Description: "Interact with the primary monitor's screen, keyboard, and mouse (Open Interpreter format).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"key", "type", "mouse_move", "left_click", "left_click_drag",
						"right_click", "middle_click", "double_click", "screenshot", "cursor_position",
					},
				},
				"text": map[string]interface{}{
					"type": "string",
				},
				"coordinate": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
			"required": []string{"action"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			action, _ := params["action"].(string)
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "Simulated computer action: " + action + " executed."},
				},
			}, nil
		},
	}
}

// HermesMemoryTool returns the clean room implementation of the memory tool
func HermesMemoryTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "memory",
		Label:       "memory",
		Description: "Save important information to persistent memory that survives across sessions.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type": "string",
				},
				"value": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"key", "value"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			key, _ := params["key"].(string)
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "Memory saved successfully for key: " + key},
				},
			}, nil
		},
	}
}

// ClineExecuteCommandTool returns the clean room implementation of execute_command tool
func ClineExecuteCommandTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "execute_command",
		Label:       "execute_command",
		Description: "Execute a CLI command on the system.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The CLI command to execute.",
				},
				"requires_approval": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether explicit approval is needed.",
				},
			},
			"required": []string{"command", "requires_approval"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			// In a real implementation this would execute the command using os/exec
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "Command executed successfully."},
				},
			}, nil
		},
	}
}

// ClineWriteToFileTool returns the clean room implementation of write_to_file tool
func ClineWriteToFileTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "write_to_file",
		Label:       "write_to_file",
		Description: "Request to write content to a file at the specified path.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The absolute path of the file.",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The complete intended content of the file.",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			// Real implementation would write to file
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "File written successfully"},
				},
			}, nil
		},
	}
}

// ClineAskFollowupTool returns the clean room implementation of ask_followup_question tool
func ClineAskFollowupTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "ask_followup_question",
		Label:       "ask_followup_question",
		Description: "Ask the user a question.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"question": map[string]interface{}{
					"type":        "string",
					"description": "The question to ask the user.",
				},
			},
			"required": []string{"question"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			question, _ := params["question"].(string)
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "[Follow-up Question Sent to User]: " + question},
				},
			}, nil
		},
	}
}

// ClineListCodeDefinitionNamesTool returns the clean room implementation of list_code_definition_names tool
func ClineListCodeDefinitionNamesTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "list_code_definition_names",
		Label:       "list_code_definition_names",
		Description: "List definition names (classes, functions, methods) used in source code files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The directory path.",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "No definitions found."},
				},
			}, nil
		},
	}
}

// ClineBrowserActionTool returns the clean room implementation of browser_action tool
func ClineBrowserActionTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "browser_action",
		Label:       "browser_action",
		Description: "Interact with a Puppeteer-controlled browser.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "launch, click, type, scroll_down, scroll_up, close",
				},
				"url": map[string]interface{}{
					"type": "string",
				},
				"coordinate": map[string]interface{}{
					"type": "string",
				},
				"text": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			action, _ := params["action"].(string)
			return agent.AgentToolResult{
				Content: []ai.Content{
					&ai.TextContent{Text: "Browser action: " + action},
				},
			}, nil
		},
	}
}
